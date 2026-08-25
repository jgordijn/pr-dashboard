package tui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jgordijn/pr-dashboard/internal/hidden"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

type memoryHiddenStore struct {
	saved *hidden.State
	err   error
	saves int
}

func (s *memoryHiddenStore) Load() (*hidden.State, error) {
	if s.saved == nil {
		return hidden.NewState(), s.err
	}
	return s.saved.Clone(), s.err
}
func (s *memoryHiddenStore) Save(state *hidden.State) error {
	s.saves++
	if s.err != nil {
		return s.err
	}
	s.saved = state.Clone()
	return nil
}

func hiddenTestModel() Model {
	m := repositoryTestModel()
	store := &memoryHiddenStore{}
	m.HiddenStore = store
	m.HiddenState = hidden.NewState()
	m.HiddenLoadErr = nil
	m.ViewMode = ViewDashboard
	return m
}

func TestHideRepositoryFiltersEveryProjectionAndFuturePRs(t *testing.T) {
	m := hiddenTestModel()
	m.SelectedKey = repositoryFocusKey("acme", "api")
	got, cmd := m.hideFocusedItem()
	m = got.(Model)
	if cmd != nil || !m.HiddenState.IsRepositoryHidden("testuser", "acme", "api") {
		t.Fatal("repository not hidden")
	}
	for _, key := range m.visibleItemKeys() {
		if strings.Contains(key, "acme/api") {
			t.Fatalf("hidden key visible: %s", key)
		}
	}
	m.Groups[0].PRs = append(m.Groups[0].PRs, model.PullRequest{Key: "acme/api#99", Organization: "acme", Repository: "api", Number: 99, Title: "future"})
	if strings.Contains(stripRepositoryANSI(m.renderPRList()), "future") {
		t.Fatal("future PR in hidden repo visible")
	}
	if len(m.mouseTargets()) == 0 {
		t.Fatal("remaining items lost mouse targets")
	}
	m.GroupingMode = model.GroupingModeOrganization
	if strings.Contains(stripRepositoryANSI(m.renderPRList()), "Add retry budget") {
		t.Fatal("hidden repo visible in org view")
	}
}

func TestHideIndividualPRAndUndo(t *testing.T) {
	m := hiddenTestModel()
	m.SelectedKey = "acme/api#2"
	got, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'H'}})
	m = got.(Model)
	if !m.HiddenState.IsPRHidden("testuser", "acme", "api", 2) || m.HiddenState.IsPRHidden("testuser", "acme", "api", 1) {
		t.Fatal("wrong PR rules")
	}
	if m.LastHidden == nil || !strings.Contains(m.FlashMessage, "z undo") {
		t.Fatal("undo feedback missing")
	}
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	m = got.(Model)
	if m.HiddenState.IsPRHidden("testuser", "acme", "api", 2) || m.LastHidden != nil || m.SelectedKey != "acme/api#2" {
		t.Fatalf("undo failed selected=%q", m.SelectedKey)
	}
	got, _ = m.undoLastHide()
	if !strings.Contains(got.(Model).FlashMessage, "Nothing") {
		t.Fatal("empty undo feedback")
	}
}

func TestHideRejectsOrganizationDuplicateAndSaveFailure(t *testing.T) {
	m := hiddenTestModel()
	m.SelectedKey = organizationFocusKey("acme")
	got, _ := m.hideFocusedItem()
	if got.(Model).HiddenState.RuleCount("testuser") != 0 || !strings.Contains(got.(Model).FlashMessage, "cannot be hidden") {
		t.Fatal("organization hidden")
	}
	m.SelectedKey = "acme/api#2"
	m.HiddenState.HidePR("testuser", "acme", "api", 2, "x", time.Now())
	got, _ = m.hideFocusedItem()
	if !strings.Contains(got.(Model).FlashMessage, "already") {
		t.Fatal("duplicate feedback")
	}
	m = hiddenTestModel()
	m.SelectedKey = "acme/api#2"
	m.HiddenStore = &memoryHiddenStore{err: errors.New("disk full")}
	got, _ = m.hideFocusedItem()
	failed := got.(Model)
	if failed.HiddenState.RuleCount("testuser") != 0 || failed.Modal.Type != ModalError {
		t.Fatal("save failure was not rolled back")
	}
}

func TestIndependentRepositoryAndPRRules(t *testing.T) {
	m := hiddenTestModel()
	now := time.Now()
	m.HiddenState.HidePR("testuser", "acme", "api", 2, "title", now)
	m.HiddenState.HideRepository("testuser", "acme", "api", "api", now.Add(time.Second))
	entries := m.HiddenState.Entries("testuser")
	next := m.HiddenState.Clone()
	next.Unhide("testuser", entries[0])
	m.HiddenState = next
	if !m.HiddenState.IsPRHidden("testuser", "acme", "api", 2) {
		t.Fatal("explicit PR should remain hidden")
	}
}

func TestHiddenManagerBrowseSearchFilterAndUnhide(t *testing.T) {
	m := hiddenTestModel()
	now := time.Now()
	m.HiddenState.HideRepository("testuser", "acme", "api", "api", now)
	m.HiddenState.HidePR("testuser", "acme", "web", 3, "Callback validation", now.Add(-time.Hour))
	previous := m.SelectedKey
	got, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'M'}})
	m = got.(Model)
	if m.ViewMode != ViewHiddenItems || m.HiddenManager.PreviousKey != previous {
		t.Fatal("manager not opened")
	}
	view := stripRepositoryANSI(m.View())
	for _, want := range []string{"Hidden Items · 2 rules", "[REPO]", "[PR]", "acme/api", "acme/web#3", "/ search"} {
		if !strings.Contains(view, want) {
			t.Errorf("missing %q", want)
		}
	}
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = got.(Model)
	for _, r := range "callback" {
		got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		m = got.(Model)
	}
	if items := m.hiddenEntries(); len(items) != 1 || items[0].Kind != hidden.KindPullRequest {
		t.Fatalf("search=%v", items)
	}
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	m = got.(Model)
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'u'}})
	m = got.(Model)
	if m.HiddenState.RuleCount("testuser") != 1 || !strings.Contains(m.HiddenManager.Message, "Restored") {
		t.Fatal("manager unhide failed")
	}
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEscape})
	m = got.(Model)
	if m.HiddenManager.Query != "" || m.ViewMode != ViewHiddenItems {
		t.Fatal("first esc should clear search")
	}
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEscape})
	m = got.(Model)
	if m.ViewMode != ViewDashboard {
		t.Fatal("second esc should close")
	}
}

func TestHiddenManagerNavigationFiltersAndEmptyStates(t *testing.T) {
	m := hiddenTestModel()
	now := time.Now()
	m.HiddenState.HideRepository("testuser", "a", "one", "one", now)
	m.HiddenState.HideRepository("testuser", "b", "two", "two", now.Add(-time.Hour))
	m.HiddenState.HidePR("testuser", "c", "three", 3, "Three", now.Add(-2*time.Hour))
	m, _ = modelFrom(m.openHiddenManager())
	for _, key := range []string{"j", "G", "k", "g", "tab", "shift+tab"} {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)}
		if key == "tab" {
			msg = tea.KeyMsg{Type: tea.KeyTab}
		}
		if key == "shift+tab" {
			msg = tea.KeyMsg{Type: tea.KeyShiftTab}
		}
		got, _ := m.handleHiddenManagerKey(msg)
		m = got.(Model)
	}
	m.HiddenManager.Searching = true
	m.HiddenManager.Query = "abc"
	got, _ := m.handleHiddenManagerKey(tea.KeyMsg{Type: tea.KeyBackspace})
	m = got.(Model)
	if m.HiddenManager.Query != "ab" {
		t.Fatal(m.HiddenManager.Query)
	}
	got, _ = m.handleHiddenManagerKey(tea.KeyMsg{Type: tea.KeyEscape})
	m = got.(Model)
	if m.HiddenManager.Searching || m.HiddenManager.Query != "" {
		t.Fatal("escape search")
	}
	m.HiddenManager.Query = "nomatch"
	if !strings.Contains(m.renderHiddenManager(), "No hidden items match") {
		t.Fatal("no-result state")
	}
	m.HiddenState = hidden.NewState()
	m.HiddenManager.Query = ""
	if !strings.Contains(m.renderHiddenManager(), "Nothing is hidden") {
		t.Fatal("empty state")
	}
	got, _ = m.unhideManagerSelection()
	if got.(Model).HiddenState.RuleCount("testuser") != 0 {
		t.Fatal("empty unhide")
	}
}

func TestHiddenFilteringCountsAndEmptyDashboard(t *testing.T) {
	m := hiddenTestModel()
	for _, pr := range m.Groups[0].PRs {
		m.HiddenState.HidePR("testuser", pr.Organization, pr.Repository, pr.Number, pr.Title, time.Now())
	}
	if m.countVisiblePRs() != 0 {
		t.Fatal("hidden count")
	}
	view := stripRepositoryANSI(m.View())
	if !strings.Contains(view, "Everything here is hidden") || !strings.Contains(view, "Press M") {
		t.Fatal(view)
	}
	header := stripRepositoryANSI(m.renderHeader())
	if !strings.Contains(header, "0/3 visible") || !strings.Contains(header, "H3") {
		t.Fatal(header)
	}
	bar := stripRepositoryANSI(m.renderStatusBar())
	if !strings.Contains(bar, "M hidden(3)") {
		t.Fatal(bar)
	}
}

func TestHiddenStateAccountScopeAndLoadFailure(t *testing.T) {
	m := hiddenTestModel()
	m.HiddenState.HideRepository("other", "acme", "api", "api", time.Now())
	if m.isPRHidden(m.Groups[0].PRs[0]) {
		t.Fatal("other account leaked")
	}
	loaded := NewModelWithHidden(m.Config, nil, nil, hidden.NewState(), errors.New("corrupt"))
	if loaded.HiddenLoadErr == nil || !strings.Contains(loaded.FlashMessage, "unavailable") {
		t.Fatal("load error hidden")
	}
	loaded.Groups = m.Groups
	loaded.SelectedKey = "acme/api#2"
	got, _ := loaded.hideFocusedItem()
	if got.(Model).Modal.Type != ModalError {
		t.Fatal("disabled persistence did not error")
	}
}

func TestHiddenManagerHelpers(t *testing.T) {
	if got := nearestKeyAt(nil, 4); got != "" {
		t.Fatal(got)
	}
	if got := nearestKeyAt([]string{"a", "b"}, 9); got != "b" {
		t.Fatal(got)
	}
	if got := nearestKeyAt([]string{"a"}, -1); got != "a" {
		t.Fatal(got)
	}
	for _, tc := range []struct{ cursor, length, want int }{{-1, 2, 0}, {9, 2, 1}, {1, 0, 0}, {1, 3, 1}} {
		if got := clampCursor(tc.cursor, tc.length); got != tc.want {
			t.Errorf("clamp=%d", got)
		}
	}
	for _, height := range []int{0, 8, 10} {
		start, end := managerWindow(5, 10, height)
		if start < 0 || end > 10 || start > end {
			t.Fatalf("window %d,%d", start, end)
		}
	}
	if got := indexOfKey([]string{"a"}, "x"); got != 1 {
		t.Fatal(got)
	}
}

func modelFrom(value tea.Model, _ tea.Cmd) (Model, tea.Cmd) { return value.(Model), nil }
