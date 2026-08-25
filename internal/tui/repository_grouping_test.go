package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

func repositoryTestModel() Model {
	m := newTestModelWithStyles()
	m.IsLoading = false
	m.Width = 100
	m.GroupingMode = model.GroupingModeRepository
	m.RepositoryCollapsed = map[string]bool{}
	base := time.Now()
	m.Groups = []model.PRGroup{{Organization: "acme", PRs: []model.PullRequest{
		{Key: "acme/api#2", Organization: "acme", Repository: "api", Number: 2, Title: "Add retry budget", Author: "jeroen", UpdatedAt: base, DaysOpen: 2, CheckStatus: model.CheckStatusPassing, ReviewStatus: model.ReviewStatusApproved, MergeStatus: model.MergeStatusReady},
		{Key: "acme/api#1", Organization: "acme", Repository: "api", Number: 1, Title: "Remove old retry path", Author: "ada", UpdatedAt: base.Add(-time.Hour), DaysOpen: 3, CheckStatus: model.CheckStatusFailing, ReviewStatus: model.ReviewStatusChangesRequested, MergeStatus: model.MergeStatusBehind, UnresolvedThreads: "2", UnresolvedCount: 2},
		{Key: "acme/web#3", Organization: "acme", Repository: "web", Number: 3, Title: "Correct callback validation", Author: "jeroen", UpdatedAt: base.Add(-2 * time.Hour), DaysOpen: 4, CheckStatus: model.CheckStatusPassing, ReviewStatus: model.ReviewStatusReviewRequired, MergeStatus: model.MergeStatusConflicts},
	}}}
	m.TotalCount = 3
	m.SelectedKey = repositoryFocusKey("acme", "api")
	return m
}

func TestRefreshPreservesOrganizationAndRepositoryCollapse(t *testing.T) {
	m := repositoryTestModel()
	m.Groups[0].Collapsed = true
	m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] = true
	m.SelectedKey = repositoryFocusKey("acme", "api")
	msg := PRsLoadedMsg{Groups: []model.PRGroup{
		{Organization: "acme", PRs: append([]model.PullRequest(nil), m.Groups[0].PRs...)},
		{Organization: "new", PRs: []model.PullRequest{{Key: "new/repo#1", Organization: "new", Repository: "repo", Number: 1}}},
	}}
	got, _ := m.handlePRsLoaded(msg)
	updated := got.(Model)
	if !updated.Groups[0].Collapsed || !updated.RepositoryCollapsed[repositoryFocusKey("acme", "api")] {
		t.Fatal("collapse state lost on refresh")
	}
}

func TestRepositoryVisibleItemsAndCollapse(t *testing.T) {
	m := repositoryTestModel()
	want := []string{"repo:acme/api", "acme/api#2", "acme/api#1", "repo:acme/web", "acme/web#3"}
	if got := m.visibleItemKeys(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("items=%v want=%v", got, want)
	}
	m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] = true
	for _, pr := range m.visiblePRs() {
		if pr.Repository == "api" {
			t.Fatal("collapsed repository PR remained visible")
		}
	}
	want = []string{"repo:acme/api", "repo:acme/web", "acme/web#3"}
	if got := m.visibleItemKeys(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("collapsed items=%v", got)
	}
}

func TestToggleGroupingUsesCollapsedRepositoryParent(t *testing.T) {
	m := repositoryTestModel()
	m.GroupingMode = model.GroupingModeOrganization
	m.SelectedKey = "acme/api#1"
	m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] = true
	got, _ := m.toggleGrouping()
	updated := got.(Model)
	if updated.GroupingMode != model.GroupingModeRepository || updated.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatalf("toggle selected %q", updated.SelectedKey)
	}
}

func TestToggleGroupingPreservesPRAndLeavesRepositoryFocus(t *testing.T) {
	m := repositoryTestModel()
	m.SelectedKey = "acme/api#1"
	got, _ := m.toggleGrouping()
	repository := got.(Model)
	if repository.GroupingMode != model.GroupingModeOrganization || repository.SelectedKey != "acme/api#1" {
		t.Fatalf("toggle=%v %q", repository.GroupingMode, repository.SelectedKey)
	}
	repository.GroupingMode = model.GroupingModeRepository
	repository.SelectedKey = repositoryFocusKey("acme", "api")
	got, _ = repository.toggleGrouping()
	organization := got.(Model)
	if organization.GroupingMode != model.GroupingModeOrganization || organization.SelectedKey != "acme/api#2" {
		t.Fatalf("node toggle=%v %q", organization.GroupingMode, organization.SelectedKey)
	}
}

func TestRepositoryNavigationAndTreeActions(t *testing.T) {
	m := repositoryTestModel()
	got, _ := m.moveDown()
	m = got.(Model)
	if m.SelectedKey != "acme/api#2" {
		t.Fatalf("down=%q", m.SelectedKey)
	}
	got, _ = m.treeLeft()
	m = got.(Model)
	if m.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatalf("left=%q", m.SelectedKey)
	}
	got, _ = m.treeRight()
	m = got.(Model)
	if m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] {
		t.Fatal("expanded node unexpectedly collapsed")
	}
	if m.SelectedKey != "acme/api#2" {
		t.Fatalf("right child=%q", m.SelectedKey)
	}
	m.SelectedKey = repositoryFocusKey("acme", "api")
	got, _ = m.toggleCurrentOrg()
	m = got.(Model)
	if !m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] || m.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatal("repository toggle failed")
	}
	got, _ = m.toggleAllOrgs()
	m = got.(Model)
	if m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] {
		t.Fatal("O should expand all when any collapsed")
	}
	got, _ = m.toggleAllOrgs()
	m = got.(Model)
	if !m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] || !m.RepositoryCollapsed[repositoryFocusKey("acme", "web")] {
		t.Fatal("O should collapse all")
	}
}

func TestTreeKeyDispatch(t *testing.T) {
	m := repositoryTestModel()
	got, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	m = got.(Model)
	if !m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] {
		t.Fatal("h did not collapse")
	}
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	m = got.(Model)
	if m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] {
		t.Fatal("l did not expand")
	}
}

func TestEnterTogglesRepositoryAndPRActionsNoOp(t *testing.T) {
	m := repositoryTestModel()
	got, cmd := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	m = got.(Model)
	if cmd != nil || !m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] {
		t.Fatal("enter did not collapse repository")
	}
	got, cmd = m.handleUpdateBranch()
	updated := got.(Model)
	if cmd != nil || updated.ActionInProgress {
		t.Fatal("repository node triggered PR action")
	}
}

func TestRepositoryRenderingTreeAndTitleBeforeProject(t *testing.T) {
	m := repositoryTestModel()
	m.SelectedKey = "acme/api#2"
	plain := stripRepositoryANSI(m.renderPRList())
	for _, want := range []string{"acme/api 2", "acme/web 1", "Add retry budget", "#2", "api", "✓ ✓ ✓"} {
		if !strings.Contains(plain, want) {
			t.Errorf("missing %q:\n%s", want, plain)
		}
	}
	row := lineContaining(plain, "Add retry budget")
	if !(strings.Index(row, "#2") < strings.Index(row, "Add retry budget") && strings.Index(row, "Add retry budget") < strings.LastIndex(row, "api")) {
		t.Fatalf("wrong order: %q", row)
	}
	if lipgloss.Width(row) > m.Width {
		t.Fatalf("row width=%d", lipgloss.Width(row))
	}
}

func TestRepositoryRowsAlignTriadsAndASCIIIsPure(t *testing.T) {
	m := repositoryTestModel()
	plain := stripRepositoryANSI(m.renderPRList())
	rows := []string{lineContaining(plain, "Add retry budget"), lineContaining(plain, "Correct callback validation")}
	if strings.Index(rows[0], "✓ ✓ ✓") != strings.Index(rows[1], "✓ ? ≠") {
		t.Fatalf("triads not aligned:\n%s\n%s", rows[0], rows[1])
	}
	m.Config.Display.ASCII = true
	m.Symbols = ASCIISymbols
	plain = stripRepositoryANSI(m.renderPRList())
	for _, r := range plain {
		if r > 127 {
			t.Fatalf("non-ASCII rune %q in %q", r, plain)
		}
	}
	if !strings.Contains(plain, "|-") || !strings.Contains(plain, "+ + =") {
		t.Fatalf("missing ASCII tree/status:\n%s", plain)
	}
}

func TestRepositoryDraftFilteringAndCollapsedRollup(t *testing.T) {
	m := repositoryTestModel()
	m.Groups[0].PRs[2].IsDraft = true
	m.ShowDrafts = false
	if strings.Contains(stripRepositoryANSI(m.renderPRList()), "acme/web") {
		t.Fatal("draft-only repository should be omitted")
	}
	m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] = true
	header := stripRepositoryANSI(m.renderPRList())
	for _, want := range []string{"✗1", "↓1", "◈2"} {
		if !strings.Contains(header, want) {
			t.Errorf("rollup missing %q: %s", want, header)
		}
	}
}

func TestRepositoryTreeEdgePaths(t *testing.T) {
	m := repositoryTestModel()
	if _, _, ok := parseRepositoryFocusKey("pr:bad"); ok {
		t.Fatal("invalid prefix parsed")
	}
	if _, _, ok := parseRepositoryFocusKey("repo:/bad"); ok {
		t.Fatal("missing owner parsed")
	}
	if _, _, ok := parseRepositoryFocusKey("repo:bad/"); ok {
		t.Fatal("missing repo parsed")
	}
	if got := m.firstVisiblePRInRepository("acme", "missing"); got != "" {
		t.Fatal(got)
	}

	m.GroupingMode = model.GroupingModeOrganization
	got, _ := m.toggleGrouping()
	m = got.(Model)
	if m.GroupingMode != model.GroupingModeRepository {
		t.Fatal("organization did not toggle to repository")
	}
	m.GroupingMode = model.GroupingModeOrganization
	before := m.SelectedKey
	got, _ = m.treeLeft()
	if got.(Model).SelectedKey != before {
		t.Fatal("left changed organization mode")
	}
	got, _ = m.treeRight()
	if got.(Model).SelectedKey != before {
		t.Fatal("right changed organization mode")
	}

	m = repositoryTestModel()
	m.RepositoryCollapsed = nil
	got, _ = m.treeRight()
	rightWithNil := got.(Model)
	if rightWithNil.SelectedKey != "acme/api#2" || rightWithNil.RepositoryCollapsed == nil {
		t.Fatal("right nil-map path")
	}
	m = repositoryTestModel()
	m.RepositoryCollapsed = nil
	got, _ = m.treeLeft()
	m = got.(Model)
	if !m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] {
		t.Fatal("left did not collapse focused repository")
	}
	got, _ = m.treeRight()
	m = got.(Model)
	if m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] || m.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatal("right did not expand in place")
	}
	got, _ = m.treeRight()
	m = got.(Model)
	if m.SelectedKey != "acme/api#2" {
		t.Fatal("second right did not enter child")
	}
	got, _ = m.treeRight()
	if got.(Model).SelectedKey != "acme/api#2" {
		t.Fatal("right changed PR leaf")
	}

	m.SelectedKey = "invalid"
	m.RepositoryCollapsed = nil
	if got := m.toggleRepository("invalid"); got.SelectedKey != "invalid" || got.RepositoryCollapsed == nil {
		t.Fatal("invalid toggle path")
	}
	m = repositoryTestModel()
	m.RepositoryCollapsed = nil
	got, _ = m.toggleAllOrgs()
	if got.(Model).RepositoryCollapsed == nil {
		t.Fatal("toggle all nil-map path")
	}
	m = repositoryTestModel()
	m.SelectedKey = "acme/api#2"
	m.RepositoryCollapsed = nil
	m = m.toggleRepository(m.SelectedKey)
	if !m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] || m.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatal("PR parent toggle path")
	}
}

func TestRepositoryRowsNeverOverflowWithLargeFixedFields(t *testing.T) {
	for width := 24; width <= 30; width++ {
		m := repositoryTestModel()
		m.Width = width
		m.Groups[0].PRs[0].Repository = "verylongreponame"
		m.Groups[0].PRs[0].Number = 1234567
		m.Groups[0].PRs[0].Title = "title"
		row := m.renderRepositoryPRRow(m.Groups[0].PRs[0], true)
		if lipgloss.Width(row) > width {
			t.Fatalf("width %d rendered %d: %q", width, lipgloss.Width(row), stripRepositoryANSI(row))
		}
	}
}

func TestRepositoryResponsiveAndStylePaths(t *testing.T) {
	for _, tc := range []struct {
		width int
		mode  model.DisplayMode
	}{{100, model.DisplayModeFull}, {70, model.DisplayModeCompact}, {45, model.DisplayModeMinimal}, {36, model.DisplayModeFull}, {23, model.DisplayModeFull}} {
		m := repositoryTestModel()
		m.Width = tc.width
		m.DisplayMode = tc.mode
		m.SelectedKey = "acme/api#2"
		m.ChangedKeys["acme/api#2"] = time.Now()
		row := m.renderRepositoryPRRow(m.Groups[0].PRs[0], false)
		if lipgloss.Width(row) > tc.width {
			t.Fatalf("width %d rendered %d: %q", tc.width, lipgloss.Width(row), stripRepositoryANSI(row))
		}
	}
	m := repositoryTestModel()
	m.SelectedKey = ""
	m.Groups[0].PRs[0].IsDraft = true
	_ = m.renderRepositoryPRRow(m.Groups[0].PRs[0], true)
	m.Groups[0].PRs[0].IsDraft = false
	m.ChangedKeys[m.Groups[0].PRs[0].Key] = time.Now()
	_ = m.renderRepositoryPRRow(m.Groups[0].PRs[0], true)
	empty := m
	empty.Groups = nil
	empty.RepositoryCollapsed = nil
	_ = empty.repositoryLayoutFor(model.PullRequest{Organization: "x", Repository: "long-repository", Number: 9, Title: "x"})
	m.SelectedKey = "acme/api#2"
	_ = m.renderRepositoryHeader(m.visibleRepositoryGroups()[0])
	m.SelectedKey = ""
	m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] = true
	m.Width = 10
	if got := m.renderRepositoryHeader(m.visibleRepositoryGroups()[0]); lipgloss.Width(got) > 10 {
		t.Fatal("narrow header overflow")
	}
	m = repositoryTestModel()
	m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] = true
	if got := stripRepositoryANSI(m.renderRepositoryHeader(m.visibleRepositoryGroups()[0])); strings.Contains(got, "✗1") == false {
		t.Fatal("collapsed risk missing")
	}
	m.Config.Display.ASCII = true
	m.Symbols = ASCIISymbols
	if got := stripRepositoryANSI(m.renderRepositoryHeader(m.visibleRepositoryGroups()[0])); !strings.Contains(got, "---") {
		t.Fatal("ASCII header fill missing")
	}
	if got := stripRepositoryANSI(m.renderStatusBar()); !strings.Contains(got, "repo") {
		t.Fatal("repository status label missing")
	}
}

func TestRepositoryASCIITruncationPaths(t *testing.T) {
	m := repositoryTestModel()
	m.Config.Display.ASCII = true
	for _, tc := range []struct {
		value string
		width int
		want  string
	}{{"abc", 0, ""}, {"abcdef", 2, ".."}, {"abc", 5, "abc"}, {"abcdef", 5, "ab..."}} {
		if got := m.truncateForMode(tc.value, tc.width); got != tc.want {
			t.Errorf("truncate(%q,%d)=%q want %q", tc.value, tc.width, got, tc.want)
		}
	}
	m.Config.Display.ASCII = false
	if got := m.truncateForMode("abcdef", 4); got != "abc…" {
		t.Fatal(got)
	}
}

func TestRepositoryKeyBindingsAndHelp(t *testing.T) {
	m := repositoryTestModel()
	got, _ := m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}})
	toggled := got.(Model)
	if toggled.GroupingMode != model.GroupingModeOrganization {
		t.Fatal("v did not toggle grouping")
	}
	help := stripRepositoryANSI(m.renderHelpModal())
	for _, want := range []string{"v view", "h/l", "repository"} {
		if !strings.Contains(help, want) {
			t.Errorf("help missing %q", want)
		}
	}
	if strings.Count(help, "\n")+1 > 24 {
		t.Fatalf("help exceeds 24 lines")
	}
}

func lineContaining(s, needle string) string {
	for _, line := range strings.Split(s, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}

func stripRepositoryANSI(s string) string {
	return regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(s, "")
}
