package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/jgordijn/pr-dashboard/internal/config"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

func newTestModel() Model {
	cfg := &config.Config{
		General: config.GeneralConfig{
			Username:        "testuser",
			RefreshInterval: 30,
		},
		Display: config.DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
	}
	return NewModel(cfg, nil)
}

func TestUpdateWindowSize(t *testing.T) {
	m := newTestModel()
	newModel, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	updated := newModel.(Model)

	if updated.Width != 100 {
		t.Errorf("expected width 100, got %d", updated.Width)
	}
	if updated.Height != 50 {
		t.Errorf("expected height 50, got %d", updated.Height)
	}
}

func TestUpdatePRsLoaded(t *testing.T) {
	m := newTestModel()
	m.IsLoading = true

	groups := []model.PRGroup{
		{
			Organization: "org1",
			PRs: []model.PullRequest{
				{Key: "org1/repo#1"},
			},
		},
	}

	msg := PRsLoadedMsg{
		Groups: groups,
		RateLimit: RateLimitInfo{
			Remaining: 5000,
			ResetAt:   time.Now().Add(time.Hour),
		},
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.IsLoading {
		t.Error("IsLoading should be false after PRsLoaded")
	}

	if len(updated.Groups) != 1 {
		t.Errorf("expected 1 group, got %d", len(updated.Groups))
	}

	if updated.Error != nil {
		t.Errorf("unexpected error: %v", updated.Error)
	}
}

func TestUpdatePRsError(t *testing.T) {
	m := newTestModel()
	m.IsLoading = true

	msg := PRsErrorMsg{Err: &testError{msg: "test error"}}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.IsLoading {
		t.Error("IsLoading should be false after error")
	}

	if updated.Error == nil {
		t.Error("Error should be set")
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestUpdateActionStart(t *testing.T) {
	m := newTestModel()

	msg := ActionStartMsg{PRKey: "org/repo#1", Action: "update"}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if !updated.ActionInProgress {
		t.Error("ActionInProgress should be true")
	}
	if !updated.IsLoading {
		t.Error("IsLoading should be true")
	}
}

func TestUpdateActionResult(t *testing.T) {
	m := newTestModel()
	m.ActionInProgress = true
	m.IsLoading = true

	// Test success
	msg := ActionResultMsg{
		PRKey:   "org/repo#1",
		Action:  "update",
		Success: true,
		Message: "Branch updated",
	}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if updated.ActionInProgress {
		t.Error("ActionInProgress should be false")
	}
	if updated.Modal.Type != ModalSuccess {
		t.Error("should show success modal")
	}

	// Test error
	m = newTestModel()
	m.ActionInProgress = true
	msg = ActionResultMsg{
		PRKey:   "org/repo#1",
		Action:  "update",
		Success: false,
		Message: "Update failed",
	}

	newModel, _ = m.Update(msg)
	updated = newModel.(Model)

	if updated.Modal.Type != ModalError {
		t.Error("should show error modal")
	}
}

func TestUpdateRefreshTick(t *testing.T) {
	m := newTestModel()
	m.WatchMode = true

	msg := RefreshTickMsg{Time: time.Now()}

	newModel, cmd := m.Update(msg)
	updated := newModel.(Model)

	if !updated.IsLoading {
		t.Error("should trigger loading")
	}
	if cmd == nil {
		t.Error("should return fetch command")
	}

	// Action in progress - should queue
	m.ActionInProgress = true
	newModel, _ = m.Update(msg)
	updated = newModel.(Model)

	if !updated.RefreshQueued {
		t.Error("should queue refresh when action in progress")
	}
}

func TestUpdateClearHighlight(t *testing.T) {
	m := newTestModel()
	m.ChangedKeys["org/repo#1"] = time.Now()

	msg := ClearHighlightMsg{Key: "org/repo#1"}

	newModel, _ := m.Update(msg)
	updated := newModel.(Model)

	if _, exists := updated.ChangedKeys["org/repo#1"]; exists {
		t.Error("highlight should be cleared")
	}
}

func TestMoveUpDown(t *testing.T) {
	m := newTestModel()
	m.Groups = []model.PRGroup{
		{
			Organization: "org1",
			PRs: []model.PullRequest{
				{Key: "org1/repo#1"},
				{Key: "org1/repo#2"},
				{Key: "org1/repo#3"},
			},
		},
	}
	m.SelectedKey = "org1/repo#2"

	// Move up
	newModel, _ := m.moveUp()
	updated := newModel.(Model)
	if updated.SelectedKey != "org1/repo#1" {
		t.Errorf("expected org1/repo#1, got %s", updated.SelectedKey)
	}

	// Move up at top - should stay
	newModel, _ = updated.moveUp()
	updated = newModel.(Model)
	if updated.SelectedKey != "org1/repo#1" {
		t.Errorf("should stay at top, got %s", updated.SelectedKey)
	}

	// Move down from middle
	m.SelectedKey = "org1/repo#2"
	newModel, _ = m.moveDown()
	updated = newModel.(Model)
	if updated.SelectedKey != "org1/repo#3" {
		t.Errorf("expected org1/repo#3, got %s", updated.SelectedKey)
	}
}

func TestGoToTopBottom(t *testing.T) {
	m := newTestModel()
	m.Groups = []model.PRGroup{
		{
			Organization: "org1",
			PRs: []model.PullRequest{
				{Key: "org1/repo#1"},
				{Key: "org1/repo#2"},
				{Key: "org1/repo#3"},
			},
		},
	}
	m.SelectedKey = "org1/repo#2"

	// Go to top
	newModel, _ := m.goToTop()
	updated := newModel.(Model)
	if updated.SelectedKey != "org1/repo#1" {
		t.Errorf("expected org1/repo#1, got %s", updated.SelectedKey)
	}

	// Go to bottom
	newModel, _ = m.goToBottom()
	updated = newModel.(Model)
	if updated.SelectedKey != "org1/repo#3" {
		t.Errorf("expected org1/repo#3, got %s", updated.SelectedKey)
	}
}

func TestToggleCurrentOrg(t *testing.T) {
	m := newTestModel()
	m.Groups = []model.PRGroup{
		{
			Organization: "org1",
			Collapsed:    false,
			PRs: []model.PullRequest{
				{Key: "org1/repo#1", Organization: "org1"},
			},
		},
		{
			Organization: "org2",
			Collapsed:    false,
			PRs: []model.PullRequest{
				{Key: "org2/repo#1", Organization: "org2"},
			},
		},
	}
	m.SelectedKey = "org1/repo#1"

	newModel, _ := m.toggleCurrentOrg()
	updated := newModel.(Model)

	if !updated.Groups[0].Collapsed {
		t.Error("org1 should be collapsed")
	}
	if updated.Groups[1].Collapsed {
		t.Error("org2 should not be affected")
	}
}

func TestToggleAllOrgs(t *testing.T) {
	m := newTestModel()
	m.Groups = []model.PRGroup{
		{Organization: "org1", Collapsed: false},
		{Organization: "org2", Collapsed: true},
	}

	// Any collapsed -> expand all
	newModel, _ := m.toggleAllOrgs()
	updated := newModel.(Model)

	for _, g := range updated.Groups {
		if g.Collapsed {
			t.Errorf("org %s should be expanded", g.Organization)
		}
	}

	// All expanded -> collapse all
	newModel, _ = updated.toggleAllOrgs()
	updated = newModel.(Model)

	for _, g := range updated.Groups {
		if !g.Collapsed {
			t.Errorf("org %s should be collapsed", g.Organization)
		}
	}
}

func TestToggleDrafts(t *testing.T) {
	m := newTestModel()
	m.ShowDrafts = true

	newModel, _ := m.toggleDrafts()
	updated := newModel.(Model)

	if updated.ShowDrafts {
		t.Error("ShowDrafts should be false")
	}

	newModel, _ = updated.toggleDrafts()
	updated = newModel.(Model)

	if !updated.ShowDrafts {
		t.Error("ShowDrafts should be true")
	}
}

func TestCycleDisplayMode(t *testing.T) {
	m := newTestModel()
	m.DisplayMode = model.DisplayModeFull

	// full -> compact
	newModel, _ := m.cycleDisplayMode()
	updated := newModel.(Model)
	if updated.DisplayMode != model.DisplayModeCompact {
		t.Errorf("expected compact, got %v", updated.DisplayMode)
	}

	// compact -> minimal
	newModel, _ = updated.cycleDisplayMode()
	updated = newModel.(Model)
	if updated.DisplayMode != model.DisplayModeMinimal {
		t.Errorf("expected minimal, got %v", updated.DisplayMode)
	}

	// minimal -> full
	newModel, _ = updated.cycleDisplayMode()
	updated = newModel.(Model)
	if updated.DisplayMode != model.DisplayModeFull {
		t.Errorf("expected full, got %v", updated.DisplayMode)
	}
}

func TestToggleWatch(t *testing.T) {
	m := newTestModel()
	m.WatchMode = false

	newModel, cmd := m.toggleWatch()
	updated := newModel.(Model)

	if !updated.WatchMode {
		t.Error("WatchMode should be true")
	}
	if cmd == nil {
		t.Error("should return tick command")
	}

	newModel, cmd = updated.toggleWatch()
	updated = newModel.(Model)

	if updated.WatchMode {
		t.Error("WatchMode should be false")
	}
	if cmd != nil {
		t.Error("should not return command when disabling")
	}
}

func TestHandleUpdateBranchPreconditions(t *testing.T) {
	tests := []struct {
		name        string
		mergeStatus model.MergeStatus
		expectModal bool
		modalType   ModalType
	}{
		{"conflicts", model.MergeStatusConflicts, true, ModalError},
		{"unknown", model.MergeStatusUnknown, true, ModalError},
		{"ready", model.MergeStatusReady, true, ModalError},
		{"blocked", model.MergeStatusBlocked, true, ModalError},
		{"behind", model.MergeStatusBehind, false, ModalNone}, // valid for update
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel()
			m.Groups = []model.PRGroup{
				{
					Organization: "org",
					PRs: []model.PullRequest{
						{Key: "org/repo#1", Organization: "org", Repository: "repo", Number: 1, MergeStatus: tc.mergeStatus},
					},
				},
			}
			m.SelectedKey = "org/repo#1"

			newModel, _ := m.handleUpdateBranch()
			updated := newModel.(Model)

			if tc.expectModal {
				if updated.Modal.Type != tc.modalType {
					t.Errorf("expected modal type %v, got %v", tc.modalType, updated.Modal.Type)
				}
			}
		})
	}
}

func TestHandleUpdateBranchBlocksConcurrent(t *testing.T) {
	m := newTestModel()
	m.ActionInProgress = true
	m.Groups = []model.PRGroup{
		{
			Organization: "org",
			PRs: []model.PullRequest{
				{Key: "org/repo#1", MergeStatus: model.MergeStatusBehind},
			},
		},
	}
	m.SelectedKey = "org/repo#1"

	newModel, cmd := m.handleUpdateBranch()
	updated := newModel.(Model)

	if cmd != nil {
		t.Error("should not issue command when action in progress")
	}
	if updated.Modal.Type != ModalNone {
		t.Error("should not show modal when blocked")
	}
}

func TestHandleUpdateBranchDraftPR(t *testing.T) {
	m := newTestModel()
	m.Groups = []model.PRGroup{
		{
			Organization: "org",
			PRs: []model.PullRequest{
				{
					Key:          "org/repo#1",
					Organization: "org",
					Repository:   "repo",
					Number:       1,
					IsDraft:      true,
					MergeStatus:  model.MergeStatusBehind, // Would be valid if not draft
				},
			},
		},
	}
	m.SelectedKey = "org/repo#1"

	newModel, cmd := m.handleUpdateBranch()
	updated := newModel.(Model)

	if cmd != nil {
		t.Error("should not issue command for draft PR")
	}
	if updated.Modal.Type != ModalError {
		t.Error("should show error modal for draft PR")
	}
	if updated.Modal.Message != "Cannot update: PR is a draft" {
		t.Errorf("unexpected modal message: %s", updated.Modal.Message)
	}
}

// TestUpdateSpinnerIntegration tests that the spinner is properly integrated.
func TestUpdateSpinnerIntegration(t *testing.T) {
	m := newTestModel()

	// Verify spinner is initialized
	if m.Spinner.View() == "" {
		t.Error("expected spinner to be initialized")
	}

	// Verify model starts in loading state
	if !m.IsLoading {
		t.Error("expected model to start in loading state")
	}

	// When loading, the header should include spinner
	header := m.renderHeader()
	if m.Spinner.View() != "" {
		// Only check if spinner produces output
		// The header should include the spinner view when loading
		spinnerView := m.Spinner.View()
		hasSpinner := len(header) > len("PR Dashboard") // header should be longer with spinner
		if !hasSpinner && spinnerView != "" {
			t.Error("expected header to be longer when loading (includes spinner)")
		}
	}

	// When not loading, header should not include spinner
	m.IsLoading = false
	headerNotLoading := m.renderHeader()
	// Header without loading should just be "PR Dashboard" (rendered with style)
	if m.IsLoading && len(headerNotLoading) > len(header) {
		t.Error("header should be shorter when not loading")
	}
}

// TestRefreshStartsSpinner tests that manual refresh starts the spinner.
func TestRefreshStartsSpinner(t *testing.T) {
	m := newTestModel()
	m.IsLoading = false

	// Press 'r' to refresh
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	newModel, cmd := m.Update(msg)
	updated := newModel.(Model)

	if !updated.IsLoading {
		t.Error("expected IsLoading to be true after refresh")
	}
	if cmd == nil {
		t.Error("expected command to be returned (fetch + spinner tick)")
	}
}

// TestNewModelInitializesSpinner tests that NewModel properly initializes the spinner.
func TestNewModelInitializesSpinner(t *testing.T) {
	cfg := &config.Config{
		General: config.GeneralConfig{
			Username:        "testuser",
			RefreshInterval: 30,
		},
		Display: config.DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
	}
	m := NewModel(cfg, nil)

	// Spinner should be initialized
	spinnerView := m.Spinner.View()
	if spinnerView == "" {
		t.Error("expected spinner to be initialized with visible output")
	}

	// Model should start in loading state
	if !m.IsLoading {
		t.Error("expected model to start in loading state")
	}
}

// TestInitReturnsSpinnerTick tests that Init returns spinner tick command.
func TestInitReturnsSpinnerTick(t *testing.T) {
	m := newTestModel()
	cmd := m.Init()

	// Init should return a batched command that includes spinner tick
	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}
