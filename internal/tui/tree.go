package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

const (
	organizationKeyPrefix = "org:"
	repositoryKeyPrefix   = "repo:"
)

func organizationFocusKey(organization string) string { return organizationKeyPrefix + organization }
func parseOrganizationFocusKey(key string) (string, bool) {
	if !strings.HasPrefix(key, organizationKeyPrefix) {
		return "", false
	}
	organization := strings.TrimPrefix(key, organizationKeyPrefix)
	return organization, organization != ""
}
func repositoryFocusKey(organization, repository string) string {
	return repositoryKeyPrefix + organization + "/" + repository
}
func parseRepositoryFocusKey(key string) (string, string, bool) {
	if !strings.HasPrefix(key, repositoryKeyPrefix) {
		return "", "", false
	}
	ownerRepo := strings.TrimPrefix(key, repositoryKeyPrefix)
	at := strings.Index(ownerRepo, "/")
	if at <= 0 || at == len(ownerRepo)-1 {
		return "", "", false
	}
	return ownerRepo[:at], ownerRepo[at+1:], true
}

type repositoryTreeOrganization struct {
	Organization string
	Repositories []model.RepositoryGroup
}

func (o repositoryTreeOrganization) PRs() []model.PullRequest {
	var prs []model.PullRequest
	for _, repository := range o.Repositories {
		prs = append(prs, repository.PRs...)
	}
	return prs
}

func (m *Model) allPRs() []model.PullRequest {
	var prs []model.PullRequest
	for _, group := range m.Groups {
		prs = append(prs, group.PRs...)
	}
	return prs
}

func (m *Model) visibleRepositoryGroups() []model.RepositoryGroup {
	groups := model.GroupByRepository(m.allPRs())
	visible := make([]model.RepositoryGroup, 0, len(groups))
	for _, group := range groups {
		filtered := make([]model.PullRequest, 0, len(group.PRs))
		for _, pr := range group.PRs {
			if m.isPRDisplayable(pr) {
				filtered = append(filtered, pr)
			}
		}
		if len(filtered) > 0 {
			group.PRs = m.sortedPRs(filtered)
			visible = append(visible, group)
		}
	}
	return visible
}

func (m *Model) visibleTreeOrganizations() []repositoryTreeOrganization {
	var organizations []repositoryTreeOrganization
	for _, repository := range m.visibleRepositoryGroups() {
		if len(organizations) == 0 || organizations[len(organizations)-1].Organization != repository.Organization {
			organizations = append(organizations, repositoryTreeOrganization{Organization: repository.Organization})
		}
		last := len(organizations) - 1
		organizations[last].Repositories = append(organizations[last].Repositories, repository)
	}
	return organizations
}

func (m *Model) visibleItemKeys() []string {
	if m.GroupingMode == model.GroupingModeOrganization {
		prs := m.visiblePRsOrganization()
		keys := make([]string, len(prs))
		for i := range prs {
			keys[i] = prs[i].Key
		}
		return keys
	}
	var keys []string
	for _, organization := range m.visibleTreeOrganizations() {
		organizationKey := organizationFocusKey(organization.Organization)
		keys = append(keys, organizationKey)
		if m.TreeOrganizationCollapsed[organizationKey] {
			continue
		}
		for _, repository := range organization.Repositories {
			repositoryKey := repositoryFocusKey(repository.Organization, repository.Repository)
			keys = append(keys, repositoryKey)
			if m.RepositoryCollapsed[repositoryKey] {
				continue
			}
			for _, pr := range repository.PRs {
				keys = append(keys, pr.Key)
			}
		}
	}
	return keys
}

func (m *Model) visiblePRsOrganization() []model.PullRequest {
	var visible []model.PullRequest
	for _, group := range m.Groups {
		if group.Collapsed {
			continue
		}
		visible = append(visible, m.sortedDisplayablePRs(group.PRs)...)
	}
	return visible
}

func (m *Model) firstVisiblePRInRepository(organization, repository string) string {
	for _, group := range m.visibleRepositoryGroups() {
		if group.Organization == organization && group.Repository == repository && len(group.PRs) > 0 {
			return group.PRs[0].Key
		}
	}
	return ""
}
func (m *Model) firstVisibleRepositoryInOrganization(organization string) string {
	for _, group := range m.visibleRepositoryGroups() {
		if group.Organization == organization {
			return repositoryFocusKey(group.Organization, group.Repository)
		}
	}
	return ""
}
func (m *Model) firstVisiblePRInOrganization(organization string) string {
	for _, pr := range m.visiblePRsOrganization() {
		if pr.Organization == organization {
			return pr.Key
		}
	}
	return ""
}

func (m Model) toggleGrouping() (tea.Model, tea.Cmd) {
	if m.GroupingMode == model.GroupingModeOrganization {
		m.GroupingMode = model.GroupingModeRepository
	} else {
		m.GroupingMode = model.GroupingModeOrganization
	}
	m.SelectedKey = m.findNearestVisibleKey()
	return m.persistViewState(), nil
}

func (m Model) treeLeft() (tea.Model, tea.Cmd) {
	if m.GroupingMode != model.GroupingModeRepository {
		return m, nil
	}
	m.ensureTreeCollapseMaps()
	if organization, ok := parseOrganizationFocusKey(m.SelectedKey); ok {
		key := organizationFocusKey(organization)
		if !m.TreeOrganizationCollapsed[key] {
			m.TreeOrganizationCollapsed[key] = true
		}
		return m.persistViewState(), nil
	}
	if owner, repo, ok := parseRepositoryFocusKey(m.SelectedKey); ok {
		key := repositoryFocusKey(owner, repo)
		if !m.RepositoryCollapsed[key] {
			m.RepositoryCollapsed[key] = true
		} else {
			m.SelectedKey = organizationFocusKey(owner)
		}
		return m.persistViewState(), nil
	}
	if pr := m.SelectedPR(); pr != nil {
		m.SelectedKey = repositoryFocusKey(pr.Organization, pr.Repository)
	}
	return m.persistViewState(), nil
}

func (m Model) treeRight() (tea.Model, tea.Cmd) {
	if m.GroupingMode != model.GroupingModeRepository {
		return m, nil
	}
	m.ensureTreeCollapseMaps()
	if organization, ok := parseOrganizationFocusKey(m.SelectedKey); ok {
		key := organizationFocusKey(organization)
		if m.TreeOrganizationCollapsed[key] {
			m.TreeOrganizationCollapsed[key] = false
			return m.persistViewState(), nil
		}
		if first := m.firstVisibleRepositoryInOrganization(organization); first != "" {
			m.SelectedKey = first
		}
		return m.persistViewState(), nil
	}
	owner, repo, ok := parseRepositoryFocusKey(m.SelectedKey)
	if !ok {
		return m, nil
	}
	key := repositoryFocusKey(owner, repo)
	if m.RepositoryCollapsed[key] {
		m.RepositoryCollapsed[key] = false
		return m.persistViewState(), nil
	}
	if first := m.firstVisiblePRInRepository(owner, repo); first != "" {
		m.SelectedKey = first
	}
	return m.persistViewState(), nil
}

func (m *Model) ensureTreeCollapseMaps() {
	if m.RepositoryCollapsed == nil {
		m.RepositoryCollapsed = make(map[string]bool)
	}
	if m.TreeOrganizationCollapsed == nil {
		m.TreeOrganizationCollapsed = make(map[string]bool)
	}
}

func (m Model) toggleOrganizationViewNode(key string) Model {
	organization, ok := parseOrganizationFocusKey(key)
	if !ok {
		return m
	}
	for i := range m.Groups {
		if m.Groups[i].Organization == organization {
			m.Groups[i].Collapsed = !m.Groups[i].Collapsed
			if m.OrganizationCollapsed == nil {
				m.OrganizationCollapsed = make(map[string]bool)
			}
			m.OrganizationCollapsed[key] = m.Groups[i].Collapsed
			m.SelectedKey = key
			break
		}
	}
	return m.persistViewState()
}

func (m Model) toggleTreeOrganization(key string) Model {
	m.ensureTreeCollapseMaps()
	organization, ok := parseOrganizationFocusKey(key)
	if !ok {
		return m
	}
	key = organizationFocusKey(organization)
	m.TreeOrganizationCollapsed[key] = !m.TreeOrganizationCollapsed[key]
	if m.TreeOrganizationCollapsed[key] {
		m.SelectedKey = key
	}
	return m.persistViewState()
}
func (m Model) toggleRepository(key string) Model {
	m.ensureTreeCollapseMaps()
	owner, repo, ok := parseRepositoryFocusKey(key)
	if !ok {
		if pr := m.SelectedPR(); pr != nil {
			owner, repo, ok = pr.Organization, pr.Repository, true
		}
	}
	if !ok {
		return m
	}
	key = repositoryFocusKey(owner, repo)
	m.RepositoryCollapsed[key] = !m.RepositoryCollapsed[key]
	if m.RepositoryCollapsed[key] {
		m.SelectedKey = key
	}
	return m.persistViewState()
}
func (m Model) toggleFocusedTreeNode() Model {
	if _, ok := parseOrganizationFocusKey(m.SelectedKey); ok {
		return m.toggleTreeOrganization(m.SelectedKey)
	}
	return m.toggleRepository(m.SelectedKey)
}
