package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jgordijn/pr-dashboard/internal/hidden"
)

func TestHiddenNilAndUnavailablePaths(t *testing.T) {
	m := hiddenTestModel()
	m.Config = nil
	m.HiddenState = nil
	m.HiddenStore = nil
	if m.hiddenAccount() != "" || len(m.hiddenEntries()) != 0 || m.hiddenRuleCount() != 0 {
		t.Fatal("nil helpers")
	}
	if err := m.saveHidden(hidden.NewState()); err == nil || !strings.Contains(err.Error(), "unavailable") {
		t.Fatal(err)
	}
}

func TestUndoFailureStaleAndRepositoryPaths(t *testing.T) {
	m := hiddenTestModel()
	entry := hidden.Entry{Kind: hidden.KindPullRequest, Organization: "acme", Repository: "api", Number: 2, HiddenAt: time.Now()}
	m.LastHidden = &entry
	got, _ := m.undoLastHide()
	stale := got.(Model)
	if stale.LastHidden != nil || !strings.Contains(stale.FlashMessage, "Nothing") {
		t.Fatal("stale undo")
	}
	m = hiddenTestModel()
	m.HiddenState.HidePR("testuser", "acme", "api", 2, "x", entry.HiddenAt)
	m.LastHidden = &entry
	m.HiddenStore = &memoryHiddenStore{err: errors.New("nope")}
	got, _ = m.undoLastHide()
	if got.(Model).Modal.Type != ModalError || got.(Model).HiddenState.RuleCount("testuser") != 1 {
		t.Fatal("undo save rollback")
	}
	m = hiddenTestModel()
	repo := hidden.Entry{Kind: hidden.KindRepository, Organization: "acme", Repository: "api", HiddenAt: time.Now()}
	m.HiddenState.HideRepository("testuser", "acme", "api", "api", repo.HiddenAt)
	m.LastHidden = &repo
	got, _ = m.undoLastHide()
	restored := got.(Model)
	if restored.HiddenState.RuleCount("testuser") != 0 || restored.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatalf("repo undo %q", restored.SelectedKey)
	}
}

func TestManagerSaveFailureAndCloseWithMissingFocus(t *testing.T) {
	m := hiddenTestModel()
	m.HiddenState.HidePR("testuser", "acme", "api", 2, "x", time.Now())
	m.HiddenStore = &memoryHiddenStore{err: errors.New("nope")}
	m, _ = modelFrom(m.openHiddenManager())
	got, _ := m.unhideManagerSelection()
	if !strings.Contains(got.(Model).HiddenManager.Message, "Could not restore") || got.(Model).HiddenState.RuleCount("testuser") != 1 {
		t.Fatal("manager rollback")
	}
	m.HiddenManager.PreviousKey = "gone/repo#9"
	m.HiddenStore = &memoryHiddenStore{}
	got, _ = m.closeHiddenManager()
	if got.(Model).ViewMode != ViewDashboard || got.(Model).SelectedKey == "gone/repo#9" {
		t.Fatal("close focus fallback")
	}
}

func TestManagerEveryKeyPathAndRendering(t *testing.T) {
	m := hiddenTestModel()
	now := time.Now()
	m.HiddenState.HideRepository("testuser", "a", "repo", "repo", now)
	m.HiddenState.HidePR("testuser", "a", "repo", 2, "Title", now.Add(-time.Minute))
	m, _ = modelFrom(m.openHiddenManager())
	// Bounds, top/bottom, filters, ignored key, and close by q.
	for _, msg := range []tea.KeyMsg{{Type: tea.KeyUp}, {Type: tea.KeyDown}, {Type: tea.KeyDown}, {Type: tea.KeyDown}, {Type: tea.KeyRunes, Runes: []rune{'G'}}, {Type: tea.KeyRunes, Runes: []rune{'g'}}, {Type: tea.KeyTab}, {Type: tea.KeyShiftTab}, {Type: tea.KeyRunes, Runes: []rune{'?'}}} {
		got, _ := m.handleHiddenManagerKey(msg)
		m = got.(Model)
	}
	m.HiddenManager.Searching = true
	if !strings.Contains(stripRepositoryANSI(m.renderHiddenManager()), "Search hidden:") {
		t.Fatal("search prompt")
	}
	for _, msg := range []tea.KeyMsg{{Type: tea.KeyBackspace}, {Type: tea.KeyRunes, Runes: []rune{'x'}}, {Type: tea.KeyEnter}} {
		got, _ := m.handleHiddenManagerKey(msg)
		m = got.(Model)
	}
	if m.HiddenManager.Searching || m.HiddenManager.Query != "x" {
		t.Fatalf("search %q", m.HiddenManager.Query)
	}
	m.HiddenManager.Searching = true
	got, _ := m.handleHiddenManagerKey(tea.KeyMsg{Type: tea.KeyEscape})
	m = got.(Model)
	if m.HiddenManager.Searching || m.HiddenManager.Query != "" {
		t.Fatal("search escape")
	}
	m.HiddenManager.Filter = hiddenFilterRepositories
	m.HiddenManager.Message = "Restored item"
	view := stripRepositoryANSI(m.renderHiddenManager())
	if !strings.Contains(view, "[REPO]") || strings.Contains(view, "[PR]") {
		t.Fatal(view)
	}
	got, _ = m.handleHiddenManagerKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if got.(Model).ViewMode != ViewDashboard {
		t.Fatal("q close")
	}
}

func TestHiddenHelperRemainingBranches(t *testing.T) {
	m := hiddenTestModel()
	m.HiddenState.HideRepository("testuser", "a", "r", "r", time.Now())
	m.HiddenState.HidePR("testuser", "a", "r", 1, "t", time.Now())
	m.HiddenManager.Filter = hiddenFilterPRs
	if items := m.hiddenEntries(); len(items) != 1 || items[0].Kind != hidden.KindPullRequest {
		t.Fatal(items)
	}
	all, repos, prs := m.hiddenCounts()
	if all != 2 || repos != 1 || prs != 1 {
		t.Fatalf("counts %d %d %d", all, repos, prs)
	}
	m.HiddenState = nil
	all, repos, prs = m.hiddenCounts()
	if all+repos+prs != 0 {
		t.Fatal("nil counts")
	}
	if hiddenEntryKey(hidden.Entry{Kind: hidden.KindRepository, Organization: "a", Repository: "r"}) != repositoryFocusKey("a", "r") {
		t.Fatal("repo key")
	}
	for _, tc := range []struct{ cursor, length, height int }{{0, 1, 30}, {9, 10, 9}} {
		start, end := managerWindow(tc.cursor, tc.length, tc.height)
		if start < 0 || end > tc.length {
			t.Fatal("window")
		}
	}
}

func TestHiddenManagerChromeFitsWidthAndHeight(t *testing.T) {
	m := hiddenTestModel()
	for i := 0; i < 20; i++ {
		m.HiddenState.HidePR("testuser", "very-long-organization", "very-long-repository", i+1, "A very long hidden pull request title", time.Now().Add(-time.Duration(i)*time.Minute))
	}
	m, _ = modelFrom(m.openHiddenManager())
	m.HiddenManager.Searching = true
	m.HiddenManager.Query = "very"
	m.HiddenManager.Message = "A long restoration status message"
	for _, width := range []int{20, 40, 80} {
		m.Width = width
		m.Height = 12
		view := stripRepositoryANSI(m.renderHiddenManager())
		for _, line := range strings.Split(view, "\n") {
			if len([]rune(line)) > width {
				t.Fatalf("width %d line %d: %q", width, len([]rune(line)), line)
			}
		}
		if lines := strings.Count(view, "\n") + 1; lines > m.Height {
			t.Fatalf("height %d lines %d", m.Height, lines)
		}
	}
}

func TestFilteredEmptyExplainsMixedCausesAccurately(t *testing.T) {
	m := hiddenTestModel()
	m.ShowDrafts = false
	m.Groups[0].PRs[0].IsDraft = true
	m.Groups[0].PRs[1].IsDraft = true
	m.HiddenState.HidePR("testuser", "acme", "web", 3, "x", time.Now())
	text := stripRepositoryANSI(m.renderFilteredEmpty())
	if !strings.Contains(text, "1 hidden") || !strings.Contains(text, "2 drafts") || strings.Contains(text, "Everything here is hidden") {
		t.Fatal(text)
	}
}

func TestFlashClearsOnOrdinaryInputAndRefresh(t *testing.T) {
	m := hiddenTestModel()
	m.FlashMessage = "stale"
	got, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if got.(Model).FlashMessage != "" {
		t.Fatal("key did not clear flash")
	}
	m.FlashMessage = "stale"
	got, _ = m.handlePRsLoaded(PRsLoadedMsg{Groups: m.Groups})
	if got.(Model).FlashMessage != "" {
		t.Fatal("refresh did not clear flash")
	}
}

func TestHiddenManagerShowsPersistenceLoadError(t *testing.T) {
	m := hiddenTestModel()
	m.HiddenLoadErr = errors.New("corrupt state")
	m.HiddenState.HidePR("testuser", "a", "repo", 1, "title", time.Now())
	m.Height = 10
	if view := stripRepositoryANSI(m.renderHiddenManager()); !strings.Contains(view, "Persistence unavailable") {
		t.Fatal(view)
	}
}

func TestModalRendersOverHiddenManager(t *testing.T) {
	m := hiddenTestModel()
	m.ViewMode = ViewHiddenItems
	m.Modal = ModalState{Type: ModalError, Title: "Async error", Message: "failed"}
	view := stripRepositoryANSI(m.View())
	if !strings.Contains(view, "Async error") || !strings.Contains(view, "Hidden Items") {
		t.Fatal(view)
	}
	got, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEscape})
	updated := got.(Model)
	if updated.Modal.Type != ModalNone || updated.ViewMode != ViewHiddenItems {
		t.Fatal("modal did not capture key over manager")
	}
}

func TestSearchAndFilterPreserveSelectedRule(t *testing.T) {
	m := hiddenTestModel()
	now := time.Now()
	m.HiddenState.HideRepository("testuser", "a", "one", "", now)
	m.HiddenState.HideRepository("testuser", "b", "two", "", now.Add(-time.Minute))
	m.HiddenState.HidePR("testuser", "b", "two", 2, "target", now.Add(-2*time.Minute))
	m, _ = modelFrom(m.openHiddenManager())
	m.HiddenManager.Cursor = 1
	selected := m.hiddenEntries()[1]
	got, _ := m.handleHiddenManagerKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = got.(Model)
	for _, r := range "o" {
		got, _ = m.handleHiddenManagerKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = got.(Model)
	}
	items := m.hiddenEntries()
	if m.HiddenManager.Cursor >= len(items) || hiddenEntryLabel(items[m.HiddenManager.Cursor]) != hiddenEntryLabel(selected) {
		t.Fatalf("cursor=%d items=%v", m.HiddenManager.Cursor, items)
	}
}

func TestNewModelWithNilHiddenStateInitializesState(t *testing.T) {
	base := repositoryTestModel()
	m := NewModelWithHidden(base.Config, nil, nil, nil, nil)
	if m.HiddenState == nil || m.HiddenState.Version != hidden.CurrentVersion {
		t.Fatal("state not initialized")
	}
}

func TestHiddenViewDisablesDashboardMouseAndDraftEmpty(t *testing.T) {
	m := hiddenTestModel()
	m.ViewMode = ViewHiddenItems
	if targets := m.mouseTargets(); len(targets) != 0 {
		t.Fatal(targets)
	}
	m.ViewMode = ViewDashboard
	m.HiddenState = hidden.NewState()
	for i := range m.Groups[0].PRs {
		m.Groups[0].PRs[i].IsDraft = true
	}
	m.ShowDrafts = false
	if !strings.Contains(stripRepositoryANSI(m.renderFilteredEmpty()), "drafts") {
		t.Fatal("draft empty")
	}
}
