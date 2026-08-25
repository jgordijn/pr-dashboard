package tui

import (
	"fmt"

	"github.com/jgordijn/pr-dashboard/internal/config"
	"github.com/jgordijn/pr-dashboard/internal/model"
	"github.com/jgordijn/pr-dashboard/internal/viewstate"
)

func defaultAccountView(cfg *config.Config) viewstate.AccountState {
	display := model.DisplayModeFull
	if cfg != nil {
		display = model.ParseDisplayMode(cfg.Display.InitialMode)
	}
	grouping := model.GroupingModeOrganization
	showDrafts := true
	if cfg != nil {
		grouping = model.ParseGroupingMode(cfg.Display.Grouping)
		showDrafts = cfg.Display.ShowDrafts
	}
	return viewstate.AccountState{GroupingMode: grouping, DisplayMode: display, ShowDrafts: showDrafts, SortField: model.SortFieldAge, SortDirection: model.SortAscending, OrganizationCollapsed: map[string]bool{}, TreeOrganizationCollapsed: map[string]bool{}, RepositoryCollapsed: map[string]bool{}}
}

func (m *Model) viewAccount() string {
	if m.Config == nil {
		return ""
	}
	return m.Config.General.Username
}

func (m *Model) currentAccountView() viewstate.AccountState {
	return viewstate.AccountState{GroupingMode: m.GroupingMode, DisplayMode: m.DisplayMode, ShowDrafts: m.ShowDrafts, SortField: m.SortField, SortDirection: m.SortDirection, SelectedKey: m.SelectedKey, OrganizationCollapsed: cloneBoolMap(m.OrganizationCollapsed), TreeOrganizationCollapsed: cloneBoolMap(m.TreeOrganizationCollapsed), RepositoryCollapsed: cloneBoolMap(m.RepositoryCollapsed)}
}

func (m *Model) applyAccountView(account viewstate.AccountState) {
	m.GroupingMode = account.GroupingMode
	m.DisplayMode = account.DisplayMode
	m.ShowDrafts = account.ShowDrafts
	m.SortField = account.SortField
	m.SortDirection = account.SortDirection
	m.SelectedKey = account.SelectedKey
	m.OrganizationCollapsed = cloneBoolMap(account.OrganizationCollapsed)
	m.TreeOrganizationCollapsed = cloneBoolMap(account.TreeOrganizationCollapsed)
	m.RepositoryCollapsed = cloneBoolMap(account.RepositoryCollapsed)
	for i := range m.Groups {
		m.Groups[i].Collapsed = m.OrganizationCollapsed[organizationFocusKey(m.Groups[i].Organization)]
	}
}

func (m *Model) restoreAccountView(login string) {
	account, ok := m.ViewState.Account(login)
	if !ok {
		account = defaultAccountView(m.Config)
	}
	m.applyAccountView(account)
}

func (m Model) persistViewState() Model {
	if m.ViewState == nil {
		m.ViewState = viewstate.NewState()
	}
	account := m.currentAccountView()
	if existing, ok := m.ViewState.Account(m.viewAccount()); ok && accountViewsEqual(existing, account) {
		if m.ViewLoadErr != nil {
			m.ViewStateWarning = "View state unavailable · " + m.ViewLoadErr.Error()
		} else {
			m.ViewStateWarning = ""
		}
		return m
	}
	next := m.ViewState.Clone()
	if err := next.SetAccount(m.viewAccount(), account); err != nil {
		m.ViewStateWarning = "View state not saved · " + err.Error()
		return m
	}
	if m.ViewLoadErr != nil {
		m.ViewState = next
		m.ViewStateWarning = "View state unavailable · " + m.ViewLoadErr.Error()
		return m
	}
	if m.ViewStore == nil {
		m.ViewState = next
		return m
	}
	if err := m.ViewStore.Save(next); err != nil {
		m.ViewStateWarning = "View state not saved · " + err.Error()
		return m
	}
	m.ViewState = next
	m.ViewStateWarning = ""
	return m
}

func (m Model) sortedPRs(prs []model.PullRequest) []model.PullRequest {
	return model.SortPullRequests(prs, m.SortField, m.SortDirection)
}
func (m *Model) sortedDisplayablePRs(prs []model.PullRequest) []model.PullRequest {
	filtered := make([]model.PullRequest, 0, len(prs))
	for _, pr := range prs {
		if m.isPRDisplayable(pr) {
			filtered = append(filtered, pr)
		}
	}
	return m.sortedPRs(filtered)
}

func (m Model) cycleSortField() Model {
	m.SortField = m.SortField.Cycle()
	m.SelectedKey = m.findNearestVisibleKey()
	return m.persistViewState()
}
func (m Model) toggleSortDirection() Model {
	m.SortDirection = m.SortDirection.Toggle()
	m.SelectedKey = m.findNearestVisibleKey()
	return m.persistViewState()
}

func (m Model) sortToken() string {
	arrow := "↑"
	if m.SortDirection == model.SortDescending {
		arrow = "↓"
	}
	if m.Config != nil && m.Config.Display.ASCII {
		if m.SortDirection == model.SortDescending {
			return "sort:" + m.SortField.String() + " desc"
		}
		return "sort:" + m.SortField.String() + " asc"
	}
	return m.SortField.String() + arrow
}

func cloneBoolMap(source map[string]bool) map[string]bool {
	clone := make(map[string]bool)
	for key, value := range source {
		if value {
			clone[key] = true
		}
	}
	return clone
}
func boolMapsEqual(left, right map[string]bool) bool {
	left = cloneBoolMap(left)
	right = cloneBoolMap(right)
	if len(left) != len(right) {
		return false
	}
	for key := range left {
		if !right[key] {
			return false
		}
	}
	return true
}
func accountViewsEqual(left, right viewstate.AccountState) bool {
	return left.GroupingMode == right.GroupingMode && left.DisplayMode == right.DisplayMode && left.ShowDrafts == right.ShowDrafts && left.SortField == right.SortField && left.SortDirection == right.SortDirection && left.SelectedKey == right.SelectedKey && boolMapsEqual(left.OrganizationCollapsed, right.OrganizationCollapsed) && boolMapsEqual(left.TreeOrganizationCollapsed, right.TreeOrganizationCollapsed) && boolMapsEqual(left.RepositoryCollapsed, right.RepositoryCollapsed)
}
func viewStateError(err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("View state unavailable · %v", err)
}
