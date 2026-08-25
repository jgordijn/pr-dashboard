package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jgordijn/pr-dashboard/internal/hidden"
	"github.com/jgordijn/pr-dashboard/internal/model"
	"github.com/jgordijn/pr-dashboard/internal/viewstate"
)

type memoryViewStore struct {
	saved *viewstate.State
	err   error
	saves int
}

func (s *memoryViewStore) Load() (*viewstate.State, error) {
	if s.saved == nil {
		return viewstate.NewState(), s.err
	}
	return s.saved.Clone(), s.err
}
func (s *memoryViewStore) Save(state *viewstate.State) error {
	s.saves++
	if s.err != nil {
		return s.err
	}
	s.saved = state.Clone()
	return nil
}

func TestRestoreCompleteAccountView(t *testing.T) {
	base := repositoryTestModel()
	state := viewstate.NewState()
	account := viewstate.AccountState{GroupingMode: model.GroupingModeRepository, DisplayMode: model.DisplayModeCompact, ShowDrafts: false, SortField: model.SortFieldState, SortDirection: model.SortDescending, SelectedKey: "acme/api#2", OrganizationCollapsed: map[string]bool{"org:acme": true}, TreeOrganizationCollapsed: map[string]bool{"org:tree": true}, RepositoryCollapsed: map[string]bool{"repo:acme/old": true}}
	if err := state.SetAccount("testuser", account); err != nil {
		t.Fatal(err)
	}
	m := NewModelWithState(base.Config, nil, nil, hidden.NewState(), nil, &memoryViewStore{}, state, nil)
	if m.GroupingMode != account.GroupingMode || m.DisplayMode != account.DisplayMode || m.ShowDrafts || m.SortField != account.SortField || m.SortDirection != account.SortDirection || m.SelectedKey != account.SelectedKey {
		t.Fatalf("restore=%+v", m.currentAccountView())
	}
	if !m.OrganizationCollapsed["org:acme"] || !m.TreeOrganizationCollapsed["org:tree"] || !m.RepositoryCollapsed["repo:acme/old"] {
		t.Fatal("collapse restore")
	}
}

func TestDefaultsAndLoadFailure(t *testing.T) {
	base := repositoryTestModel()
	base.Config.Display.Grouping = "organization"
	base.Config.Display.InitialMode = "minimal"
	base.Config.Display.ShowDrafts = false
	m := NewModelWithState(base.Config, nil, nil, hidden.NewState(), nil, nil, nil, errors.New("corrupt"))
	if m.GroupingMode != model.GroupingModeOrganization || m.DisplayMode != model.DisplayModeMinimal || m.ShowDrafts || m.SortField != model.SortFieldAge || m.SortDirection != model.SortAscending {
		t.Fatal("defaults")
	}
	if !strings.Contains(m.ViewStateWarning, "unavailable") {
		t.Fatal(m.ViewStateWarning)
	}
}

func TestSortingIntegratesBothProjectionsAndStableFocus(t *testing.T) {
	m := repositoryTestModel()
	m.ViewStore = &memoryViewStore{}
	m.ViewState = viewstate.NewState()
	m.SelectedKey = "acme/api#1"
	m.Groups[0].PRs[0].Title = "Zulu"
	m.Groups[0].PRs[1].Title = "alpha"
	m.Groups[0].PRs[2].Title = "Middle"
	m.SortField = model.SortFieldName
	m.SortDirection = model.SortAscending
	m.GroupingMode = model.GroupingModeOrganization
	if got := m.visiblePRsOrganization(); got[0].Key != "acme/api#1" || got[2].Key != "acme/api#2" {
		t.Fatalf("name order=%v", keysOf(got))
	}
	m.SortDirection = model.SortDescending
	if got := m.visiblePRsOrganization(); got[0].Key != "acme/api#2" {
		t.Fatalf("desc=%v", keysOf(got))
	}
	m.GroupingMode = model.GroupingModeRepository
	m.SortDirection = model.SortAscending
	groups := m.visibleRepositoryGroups()
	if groups[0].PRs[0].Key != "acme/api#1" {
		t.Fatalf("repo order=%v", keysOf(groups[0].PRs))
	}
	before := m.SelectedKey
	m = m.cycleSortField()
	if m.SelectedKey != before || m.SortField != model.SortFieldAge {
		t.Fatal("cycle sort")
	}
	m = m.toggleSortDirection()
	if m.SortDirection != model.SortDescending {
		t.Fatal("toggle direction")
	}
}

func TestStateSortingWorstFirstAndAge(t *testing.T) {
	m := repositoryTestModel()
	m.SortField = model.SortFieldState
	m.SortDirection = model.SortDescending
	m.Groups[0].PRs[0].CheckStatus = model.CheckStatusPassing
	m.Groups[0].PRs[0].ReviewStatus = model.ReviewStatusApproved
	m.Groups[0].PRs[0].MergeStatus = model.MergeStatusReady
	m.Groups[0].PRs[1].CheckStatus = model.CheckStatusFailing
	if got := m.sortedDisplayablePRs(m.Groups[0].PRs); got[0].Key != "acme/api#1" {
		t.Fatalf("state=%v", keysOf(got))
	}
	m.SortField = model.SortFieldAge
	m.SortDirection = model.SortAscending
	now := time.Now()
	m.Groups[0].PRs[0].CreatedAt = now.Add(-time.Hour)
	m.Groups[0].PRs[1].CreatedAt = now.Add(-48 * time.Hour)
	if got := m.sortedDisplayablePRs(m.Groups[0].PRs[:2]); got[0].Key != "acme/api#2" {
		t.Fatalf("age=%v", keysOf(got))
	}
}

func TestViewMutationsPersistAndAvoidDuplicateWrites(t *testing.T) {
	m := repositoryTestModel()
	store := &memoryViewStore{}
	m.ViewStore = store
	m.ViewState = viewstate.NewState()
	m.OrganizationCollapsed = map[string]bool{}
	got, _ := m.toggleDrafts()
	m = got.(Model)
	if store.saves != 1 {
		t.Fatal(store.saves)
	}
	m = m.persistViewState()
	if store.saves != 1 {
		t.Fatal("unchanged state wrote")
	}
	got, _ = m.moveDown()
	m = got.(Model)
	if store.saves != 2 {
		t.Fatal("focus not saved")
	}
	got, _ = m.toggleGrouping()
	m = got.(Model)
	if store.saves != 3 {
		t.Fatal("group not saved")
	}
	got, _ = m.cycleDisplayMode()
	m = got.(Model)
	if store.saves != 4 {
		t.Fatal("mode not saved")
	}
	account, ok := store.saved.Account("testuser")
	if !ok || account.SelectedKey != m.SelectedKey || account.DisplayMode != m.DisplayMode {
		t.Fatal("snapshot mismatch")
	}
}

func TestCollapseAndMouseFocusPersist(t *testing.T) {
	m := repositoryTestModel()
	store := &memoryViewStore{}
	m.ViewStore = store
	m.ViewState = viewstate.NewState()
	m.SelectedKey = organizationFocusKey("acme")
	got, _ := m.toggleCurrentOrg()
	m = got.(Model)
	account, _ := store.saved.Account("testuser")
	if !account.TreeOrganizationCollapsed[organizationFocusKey("acme")] {
		t.Fatal("tree org collapse not saved")
	}
	m.TreeOrganizationCollapsed[organizationFocusKey("acme")] = false
	m.SelectedKey = "acme/api#2"
	target := findMouseTarget(t, m, "Remove old retry path")
	got, _ = m.handleMouseMsg(tea.MouseMsg{X: target.X, Y: target.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = got.(Model)
	account, _ = store.saved.Account("testuser")
	if account.SelectedKey != "acme/api#1" {
		t.Fatalf("mouse focus=%q", account.SelectedKey)
	}
}

func TestViewWarningComposesWithFlashAndClearsWhenRuntimeMatchesDisk(t *testing.T) {
	m := repositoryTestModel()
	store := &memoryViewStore{}
	m.ViewStore = store
	m.ViewState = viewstate.NewState()
	m = m.persistViewState()
	store.err = errors.New("disk")
	m.SortField = m.SortField.Cycle()
	m = m.persistViewState()
	if m.ViewStateWarning == "" {
		t.Fatal("warning missing")
	}
	m.FlashMessage = "Hidden acme/api#2 · z undo"
	m.Width = 200
	bar := stripRepositoryANSI(m.renderStatusBar())
	if !strings.Contains(bar, "Hidden acme/api#2") || !strings.Contains(bar, "View state not saved") {
		t.Fatal(bar)
	}
	m.SortField = model.SortFieldAge
	m = m.persistViewState()
	if m.ViewStateWarning != "" {
		t.Fatal("matching disk warning stuck")
	}
}

func TestCorruptViewWarningRemainsOnEqualRuntime(t *testing.T) {
	m := repositoryTestModel()
	m.ViewState = viewstate.NewState()
	m.ViewLoadErr = errors.New("corrupt")
	m = m.persistViewState()
	m = m.persistViewState()
	if !strings.Contains(m.ViewStateWarning, "unavailable") {
		t.Fatal(m.ViewStateWarning)
	}
}

func TestStaleAccountFetchIsIgnored(t *testing.T) {
	m := repositoryTestModel()
	store := &memoryViewStore{}
	m.ViewStore = store
	m.ViewState = viewstate.NewState()
	m.Config.General.Username = "second"
	before := m.Groups
	got, _ := m.handlePRsLoaded(PRsLoadedMsg{Account: "first", Groups: []model.PRGroup{{Organization: "wrong", PRs: []model.PullRequest{{Key: "wrong/repo#1"}}}}})
	updated := got.(Model)
	if updated.Groups[0].Organization != before[0].Organization || store.saves != 0 {
		t.Fatal("stale fetch applied or persisted")
	}
	updated.IsLoading = true
	result, _ := updated.Update(PRsErrorMsg{Account: "first", Err: errors.New("old")})
	after := result.(Model)
	if !after.IsLoading || after.Error != nil {
		t.Fatal("stale error applied")
	}
}

func TestPersistenceFailureKeepsRuntimeAndRetries(t *testing.T) {
	m := repositoryTestModel()
	store := &memoryViewStore{err: errors.New("disk")}
	m.ViewStore = store
	m.ViewState = viewstate.NewState()
	old := m.SortField
	m = m.cycleSortField()
	if m.SortField == old || !strings.Contains(m.ViewStateWarning, "not saved") {
		t.Fatal("failure behavior")
	}
	store.err = nil
	m = m.toggleSortDirection()
	if m.ViewStateWarning != "" || store.saved == nil {
		t.Fatal("retry did not recover")
	}
}

func TestRefreshRestoresCollapseAndReconcilesFocus(t *testing.T) {
	m := repositoryTestModel()
	store := &memoryViewStore{}
	m.ViewStore = store
	m.ViewState = viewstate.NewState()
	m.OrganizationCollapsed = map[string]bool{"org:acme": true}
	m.SelectedKey = "gone/repo#9"
	msg := PRsLoadedMsg{Groups: []model.PRGroup{{Organization: "acme", PRs: m.Groups[0].PRs}}}
	got, _ := m.handlePRsLoaded(msg)
	updated := got.(Model)
	if !updated.Groups[0].Collapsed || updated.SelectedKey != organizationFocusKey("acme") {
		t.Fatalf("refresh collapsed=%v selected=%q", updated.Groups[0].Collapsed, updated.SelectedKey)
	}
	account, _ := store.saved.Account("testuser")
	if account.SelectedKey != organizationFocusKey("acme") {
		t.Fatal("fallback not persisted")
	}
}

func TestSortKeysAndIndicator(t *testing.T) {
	m := repositoryTestModel()
	m.ViewStore = &memoryViewStore{}
	m.ViewState = viewstate.NewState()
	got, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = got.(Model)
	if m.SortField != model.SortFieldState {
		t.Fatal(m.SortField)
	}
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	m = got.(Model)
	if m.SortDirection != model.SortDescending {
		t.Fatal(m.SortDirection)
	}
	if bar := stripRepositoryANSI(m.renderStatusBar()); !strings.Contains(bar, "state↓") {
		t.Fatal(bar)
	}
	m.Config.Display.ASCII = true
	if token := m.sortToken(); token != "sort:state desc" {
		t.Fatal(token)
	}
}

func TestAccountScopedRestoration(t *testing.T) {
	m := repositoryTestModel()
	state := viewstate.NewState()
	state.SetAccount("other", viewstate.AccountState{GroupingMode: model.GroupingModeOrganization, DisplayMode: model.DisplayModeMinimal, ShowDrafts: false, SortField: model.SortFieldName, SortDirection: model.SortDescending, OrganizationCollapsed: map[string]bool{}, TreeOrganizationCollapsed: map[string]bool{}, RepositoryCollapsed: map[string]bool{}})
	m.ViewState = state
	m.ViewStore = &memoryViewStore{}
	got, cmd := m.handleAccountSwitched(AccountSwitchedMsg{Login: "other", Token: "token"})
	updated := got.(Model)
	if cmd == nil || updated.Config.General.Username != "other" || updated.DisplayMode != model.DisplayModeMinimal || updated.SortField != model.SortFieldName || len(updated.Groups) != 0 {
		t.Fatalf("switch=%+v", updated.currentAccountView())
	}
}

func TestViewStatePersistenceEdgePaths(t *testing.T) {
	m := repositoryTestModel()
	m.ViewState = nil
	m.ViewStore = nil
	account := viewstate.AccountState{GroupingMode: model.GroupingModeRepository, DisplayMode: model.DisplayModeFull, ShowDrafts: true, SortField: model.SortFieldAge, SortDirection: model.SortAscending, OrganizationCollapsed: map[string]bool{"org:acme": true}, TreeOrganizationCollapsed: map[string]bool{}, RepositoryCollapsed: map[string]bool{}}
	m.applyAccountView(account)
	if !m.Groups[0].Collapsed {
		t.Fatal("apply collapse")
	}
	m = m.persistViewState()
	if m.ViewState == nil {
		t.Fatal("nil state")
	}
	m = repositoryTestModel()
	m.ViewState = viewstate.NewState()
	m.ViewLoadErr = errors.New("corrupt")
	m = m.persistViewState()
	if !strings.Contains(m.ViewStateWarning, "unavailable") {
		t.Fatal(m.ViewStateWarning)
	}
	m = repositoryTestModel()
	m.ViewState = viewstate.NewState()
	m.Config = nil
	m = m.persistViewState()
	if !strings.Contains(m.ViewStateWarning, "account") {
		t.Fatal(m.ViewStateWarning)
	}
}

func TestViewStateHelpers(t *testing.T) {
	if got := viewStateError(nil); got != "" {
		t.Fatal(got)
	}
	if got := viewStateError(errors.New("x")); !strings.Contains(got, "x") {
		t.Fatal(got)
	}
	if !boolMapsEqual(map[string]bool{"a": true, "x": false}, map[string]bool{"a": true}) || boolMapsEqual(map[string]bool{"a": true}, map[string]bool{"b": true}) {
		t.Fatal("map equality")
	}
	m := repositoryTestModel()
	m.Config = nil
	if m.viewAccount() != "" {
		t.Fatal("nil account")
	}
	defaults := defaultAccountView(nil)
	if !defaults.ShowDrafts || defaults.SortField != model.SortFieldAge {
		t.Fatal(defaults)
	}
}

func keysOf(prs []model.PullRequest) []string {
	keys := make([]string, len(prs))
	for i := range prs {
		keys[i] = prs[i].Key
	}
	return keys
}
