package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

const repositoryKeyPrefix = "repo:"

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
			if m.ShowDrafts || !pr.IsDraft {
				filtered = append(filtered, pr)
			}
		}
		if len(filtered) > 0 {
			group.PRs = filtered
			visible = append(visible, group)
		}
	}
	return visible
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
	for _, group := range m.visibleRepositoryGroups() {
		key := repositoryFocusKey(group.Organization, group.Repository)
		keys = append(keys, key)
		if m.RepositoryCollapsed[key] {
			continue
		}
		for _, pr := range group.PRs {
			keys = append(keys, pr.Key)
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
		for _, pr := range group.PRs {
			if m.ShowDrafts || !pr.IsDraft {
				visible = append(visible, pr)
			}
		}
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

func (m Model) toggleGrouping() (tea.Model, tea.Cmd) {
	if m.GroupingMode == model.GroupingModeOrganization {
		m.GroupingMode = model.GroupingModeRepository
	} else {
		m.GroupingMode = model.GroupingModeOrganization
	}
	m.SelectedKey = m.findNearestVisibleKey()
	return m, nil
}

func (m Model) treeLeft() (tea.Model, tea.Cmd) {
	if m.GroupingMode != model.GroupingModeRepository {
		return m, nil
	}
	if m.RepositoryCollapsed == nil {
		m.RepositoryCollapsed = make(map[string]bool)
	}
	if owner, repo, ok := parseRepositoryFocusKey(m.SelectedKey); ok {
		key := repositoryFocusKey(owner, repo)
		if !m.RepositoryCollapsed[key] {
			m.RepositoryCollapsed[key] = true
		}
		return m, nil
	}
	if pr := m.SelectedPR(); pr != nil {
		m.SelectedKey = repositoryFocusKey(pr.Organization, pr.Repository)
	}
	return m, nil
}

func (m Model) treeRight() (tea.Model, tea.Cmd) {
	if m.GroupingMode != model.GroupingModeRepository {
		return m, nil
	}
	owner, repo, ok := parseRepositoryFocusKey(m.SelectedKey)
	if !ok {
		return m, nil
	}
	key := repositoryFocusKey(owner, repo)
	if m.RepositoryCollapsed == nil {
		m.RepositoryCollapsed = make(map[string]bool)
	}
	if m.RepositoryCollapsed[key] {
		m.RepositoryCollapsed[key] = false
		return m, nil
	}
	if first := m.firstVisiblePRInRepository(owner, repo); first != "" {
		m.SelectedKey = first
	}
	return m, nil
}

func (m Model) toggleRepository(key string) Model {
	if m.RepositoryCollapsed == nil {
		m.RepositoryCollapsed = make(map[string]bool)
	}
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
	return m
}
