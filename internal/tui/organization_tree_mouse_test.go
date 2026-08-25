package tui

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

func TestOrganizationNodeKeyRoundTrip(t *testing.T) {
	key := organizationFocusKey("RoyalAholdDelhaize")
	if key != "org:RoyalAholdDelhaize" {
		t.Fatal(key)
	}
	if got, ok := parseOrganizationFocusKey(key); !ok || got != "RoyalAholdDelhaize" {
		t.Fatalf("parse=%q %v", got, ok)
	}
	for _, invalid := range []string{"", "org:", "repo:RoyalAholdDelhaize/x"} {
		if _, ok := parseOrganizationFocusKey(invalid); ok {
			t.Fatalf("parsed %q", invalid)
		}
	}
}

func TestRepositoryProjectionContainsOrganizationRoots(t *testing.T) {
	m := repositoryTestModel()
	m.Groups[0].Organization = "RoyalAholdDelhaize"
	for i := range m.Groups[0].PRs {
		m.Groups[0].PRs[i].Organization = "RoyalAholdDelhaize"
		m.Groups[0].PRs[i].Key = strings.Replace(m.Groups[0].PRs[i].Key, "acme/", "RoyalAholdDelhaize/", 1)
	}
	m.SelectedKey = organizationFocusKey("RoyalAholdDelhaize")
	keys := m.visibleItemKeys()
	want := []string{"org:RoyalAholdDelhaize", "repo:RoyalAholdDelhaize/api", "RoyalAholdDelhaize/api#2", "RoyalAholdDelhaize/api#1", "repo:RoyalAholdDelhaize/web", "RoyalAholdDelhaize/web#3"}
	if strings.Join(keys, "|") != strings.Join(want, "|") {
		t.Fatalf("keys=%v", keys)
	}
	plain := stripRepositoryANSI(m.renderPRList())
	lines := strings.Split(strings.TrimSpace(plain), "\n")
	if !strings.Contains(lines[0], "RoyalAholdDelhaize 3") {
		t.Fatalf("org root=%q", lines[0])
	}
	if strings.Contains(lines[1], "RoyalAholdDelhaize/api") || !strings.Contains(lines[1], "api 2") {
		t.Fatalf("repo child=%q", lines[1])
	}
}

func TestOrganizationCollapseHidesRepositoriesAndPRs(t *testing.T) {
	m := repositoryTestModel()
	key := organizationFocusKey("acme")
	m.TreeOrganizationCollapsed[key] = true
	if got := m.visibleItemKeys(); len(got) != 1 || got[0] != key {
		t.Fatalf("items=%v", got)
	}
	plain := stripRepositoryANSI(m.renderPRList())
	if strings.Contains(plain, "api 2") || !strings.Contains(plain, "acme 3") {
		t.Fatalf("tree=%s", plain)
	}
}

func TestOrganizationTreeEdgePaths(t *testing.T) {
	m := repositoryTestModel()
	if got := m.firstVisibleRepositoryInOrganization("missing"); got != "" {
		t.Fatal(got)
	}
	if got := m.firstVisiblePRInOrganization("acme"); got != "acme/api#2" {
		t.Fatal(got)
	}
	if got := m.firstVisiblePRInOrganization("missing"); got != "" {
		t.Fatal(got)
	}
	m.TreeOrganizationCollapsed[organizationFocusKey("acme")] = true
	if prs := m.visiblePRs(); len(prs) != 0 {
		t.Fatalf("collapsed org prs=%v", prs)
	}
	m.SelectedKey = organizationFocusKey("acme")
	m.TreeOrganizationCollapsed = nil
	m.RepositoryCollapsed = nil
	got, _ := m.treeRight()
	m = got.(Model)
	if m.TreeOrganizationCollapsed == nil || m.RepositoryCollapsed == nil || m.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatal("right did not initialize/enter")
	}
	m.SelectedKey = organizationFocusKey("acme")
	m.TreeOrganizationCollapsed[m.SelectedKey] = true
	got, _ = m.treeRight()
	m = got.(Model)
	if m.TreeOrganizationCollapsed[organizationFocusKey("acme")] || m.SelectedKey != organizationFocusKey("acme") {
		t.Fatal("right did not expand in place")
	}
	got, _ = m.treeRight()
	m = got.(Model)
	if m.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatal(m.SelectedKey)
	}
	m.SelectedKey = "bad"
	before := m
	if got := m.toggleTreeOrganization("bad"); got.SelectedKey != before.SelectedKey {
		t.Fatal("invalid org toggled")
	}
}

func TestToggleOrganizationNodeWithKeyboardAndProjection(t *testing.T) {
	m := repositoryTestModel()
	m.SelectedKey = organizationFocusKey("acme")
	got, _ := m.toggleCurrentOrg()
	m = got.(Model)
	if !m.TreeOrganizationCollapsed[organizationFocusKey("acme")] {
		t.Fatal("o did not collapse org")
	}
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	m = got.(Model)
	if m.TreeOrganizationCollapsed[organizationFocusKey("acme")] {
		t.Fatal("enter did not expand org")
	}
	m.GroupingMode = model.GroupingModeRepository
	m.SelectedKey = organizationFocusKey("acme")
	got, _ = m.toggleGrouping()
	m = got.(Model)
	if m.GroupingMode != model.GroupingModeOrganization || m.SelectedKey != "acme/api#2" {
		t.Fatalf("projection selection=%q", m.SelectedKey)
	}
}

func TestOrganizationHeaderCollapsedASCIIAndWidth(t *testing.T) {
	m := repositoryTestModel()
	m.TreeOrganizationCollapsed[organizationFocusKey("acme")] = true
	m.Config.Display.ASCII = true
	m.Symbols = ASCIISymbols
	m.Width = 30
	header := stripRepositoryANSI(m.renderTreeOrganizationHeader(m.visibleTreeOrganizations()[0]))
	for _, want := range []string{"acme 3", "x1", "v1", "t2"} {
		if !strings.Contains(header, want) {
			t.Errorf("missing %q: %s", want, header)
		}
	}
	for _, r := range header {
		if r > 127 {
			t.Fatalf("non-ASCII %q", r)
		}
	}
	if len(header) > 30 {
		t.Fatalf("header width=%d", len(header))
	}
	m.Width = 10
	if narrow := stripRepositoryANSI(m.renderTreeOrganizationHeader(m.visibleTreeOrganizations()[0])); len(narrow) > 10 {
		t.Fatalf("narrow=%q", narrow)
	}
}

func TestFindNearestUsesOrganizationAncestorAndToggleAll(t *testing.T) {
	m := repositoryTestModel()
	m.SelectedKey = "acme/api#2"
	m.TreeOrganizationCollapsed[organizationFocusKey("acme")] = true
	if got := m.findNearestVisibleKey(); got != organizationFocusKey("acme") {
		t.Fatalf("nearest=%q", got)
	}
	m.SelectedKey = organizationFocusKey("acme")
	got, _ := m.toggleAllOrgs()
	m = got.(Model)
	if m.TreeOrganizationCollapsed[organizationFocusKey("acme")] {
		t.Fatal("any collapsed should expand all")
	}
	got, _ = m.toggleAllOrgs()
	m = got.(Model)
	if !m.TreeOrganizationCollapsed[organizationFocusKey("acme")] {
		t.Fatal("all expanded should collapse all")
	}
}

func TestThreeLevelLeftRightNavigation(t *testing.T) {
	m := repositoryTestModel()
	m.SelectedKey = organizationFocusKey("acme")
	got, _ := m.treeRight()
	m = got.(Model)
	if m.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatal(m.SelectedKey)
	}
	got, _ = m.treeRight()
	m = got.(Model)
	if m.SelectedKey != "acme/api#2" {
		t.Fatal(m.SelectedKey)
	}
	got, _ = m.treeLeft()
	m = got.(Model)
	if m.SelectedKey != repositoryFocusKey("acme", "api") {
		t.Fatal(m.SelectedKey)
	}
	m.RepositoryCollapsed[m.SelectedKey] = true
	got, _ = m.treeLeft()
	m = got.(Model)
	if m.SelectedKey != organizationFocusKey("acme") {
		t.Fatal(m.SelectedKey)
	}
	got, _ = m.treeLeft()
	m = got.(Model)
	if !m.TreeOrganizationCollapsed[organizationFocusKey("acme")] {
		t.Fatal("org not collapsed")
	}
}

func TestMouseClickOpensExactPR(t *testing.T) {
	m := repositoryTestModel()
	m.SelectedKey = "acme/api#1"
	var opened string
	m.OpenPR = func(org, repo string, number int) error {
		opened = org + "/" + repo + "#" + string(rune('0'+number))
		return nil
	}
	target := findMouseTarget(t, m, "Correct callback validation")
	got, cmd := m.handleMouseMsg(tea.MouseMsg{X: target.X, Y: target.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = got.(Model)
	if m.SelectedKey != "acme/web#3" || cmd == nil {
		t.Fatalf("selected=%q cmd=%v", m.SelectedKey, cmd)
	}
	if msg := cmd(); msg != nil || opened != "acme/web#3" {
		t.Fatalf("msg=%#v opened=%q", msg, opened)
	}
}

func TestMissingBrowserOpenerReturnsError(t *testing.T) {
	m := repositoryTestModel()
	m.OpenPR = nil
	result := m.openBrowserCmd(&m.Groups[0].PRs[0])().(ActionResultMsg)
	if result.Err == nil || result.Success {
		t.Fatalf("result=%#v", result)
	}
}

func TestMouseOpenErrorTargetsClickedPR(t *testing.T) {
	m := repositoryTestModel()
	m.OpenPR = func(org, repo string, number int) error { return errors.New("boom") }
	target := findMouseTarget(t, m, "Add retry budget")
	_, cmd := m.handleMouseMsg(tea.MouseMsg{X: target.X, Y: target.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	result := cmd().(ActionResultMsg)
	if result.Success || result.PRKey != "acme/api#2" || result.Err == nil {
		t.Fatalf("result=%#v", result)
	}
}

func TestMouseOrganizationProjectionOpensLeavesAndTogglesHeaders(t *testing.T) {
	m := repositoryTestModel()
	m.GroupingMode = model.GroupingModeOrganization
	m.SelectedKey = "acme/api#1"
	var opened string
	m.OpenPR = func(org, repo string, number int) error {
		opened = org + "/" + repo
		return nil
	}

	pr := findMouseTarget(t, m, "Correct callback validation")
	got, cmd := m.handleMouseMsg(tea.MouseMsg{X: pr.X, Y: pr.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = got.(Model)
	if m.SelectedKey != "acme/web#3" || cmd == nil {
		t.Fatalf("selected=%q", m.SelectedKey)
	}
	_ = cmd()
	if opened != "acme/web" {
		t.Fatalf("opened=%q", opened)
	}

	org := findMouseTarget(t, m, "acme 3")
	got, cmd = m.handleMouseMsg(tea.MouseMsg{X: org.X, Y: org.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = got.(Model)
	if cmd != nil || m.SelectedKey != organizationFocusKey("acme") || !m.Groups[0].Collapsed {
		t.Fatal("organization header click")
	}
}

func TestOrganizationToggleDefensiveNoSelectionPaths(t *testing.T) {
	m := repositoryTestModel()
	m.GroupingMode = model.GroupingModeOrganization
	for _, key := range []string{"", "missing/repo#99"} {
		m.SelectedKey = key
		got, cmd := m.toggleCurrentOrg()
		if got.(Model).SelectedKey != key || cmd != nil {
			t.Fatalf("key %q changed", key)
		}
	}
}

func TestOrganizationViewToggleRejectsInvalidAndMissingKeys(t *testing.T) {
	m := repositoryTestModel()
	before := m
	for _, key := range []string{"bad", organizationFocusKey("missing")} {
		got := m.toggleOrganizationViewNode(key)
		if got.SelectedKey != before.SelectedKey || got.Groups[0].Collapsed != before.Groups[0].Collapsed {
			t.Fatalf("key %q changed state", key)
		}
	}
}

func TestOrganizationModeClickedHeaderRetainsKeyboardParityAndSelection(t *testing.T) {
	m := repositoryTestModel()
	m.GroupingMode = model.GroupingModeOrganization
	org := findMouseTarget(t, m, "acme 3")
	got, _ := m.handleMouseMsg(tea.MouseMsg{X: org.X, Y: org.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = got.(Model)
	if !strings.Contains(stripRepositoryANSI(m.renderOrgHeader(m.Groups[0])), m.Symbols.Selected) {
		t.Fatal("selected organization gutter missing")
	}
	got, _ = m.handleKeyMsg(tea.KeyMsg{Type: tea.KeyEnter})
	m = got.(Model)
	if m.Groups[0].Collapsed {
		t.Fatal("enter did not expand clicked org")
	}
	got, _ = m.toggleCurrentOrg()
	m = got.(Model)
	if !m.Groups[0].Collapsed {
		t.Fatal("o did not collapse clicked org")
	}
}

func TestMouseOrganizationProjectionCollapsedHeaderStillToggles(t *testing.T) {
	m := repositoryTestModel()
	m.GroupingMode = model.GroupingModeOrganization
	m.Groups[0].Collapsed = true
	if keys := m.mouseItemKeys(); len(keys) != 1 || keys[0] != organizationFocusKey("acme") {
		t.Fatalf("keys=%v", keys)
	}
	org := findMouseTarget(t, m, "acme 3")
	got, cmd := m.handleMouseMsg(tea.MouseMsg{X: org.X, Y: org.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	updated := got.(Model)
	if cmd != nil || updated.Groups[0].Collapsed || updated.SelectedKey != organizationFocusKey("acme") {
		t.Fatal("collapsed header did not expand")
	}
}

func TestMouseNodeClicksToggleExactNodes(t *testing.T) {
	m := repositoryTestModel()
	org := findMouseTarget(t, m, "acme 3")
	got, cmd := m.handleMouseMsg(tea.MouseMsg{X: org.X, Y: org.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = got.(Model)
	if cmd != nil || m.SelectedKey != organizationFocusKey("acme") || !m.TreeOrganizationCollapsed[organizationFocusKey("acme")] {
		t.Fatal("org click")
	}
	m.TreeOrganizationCollapsed[organizationFocusKey("acme")] = false
	repo := findMouseTarget(t, m, "api 2")
	got, cmd = m.handleMouseMsg(tea.MouseMsg{X: repo.X, Y: repo.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	m = got.(Model)
	if cmd != nil || m.SelectedKey != repositoryFocusKey("acme", "api") || !m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] {
		t.Fatal("repo click")
	}
}

func TestMouseNodeClickInitializesCollapseMaps(t *testing.T) {
	m := repositoryTestModel()
	m.TreeOrganizationCollapsed = nil
	org := findMouseTarget(t, m, "acme 3")
	got, _ := m.handleMouseMsg(tea.MouseMsg{X: org.X, Y: org.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if updated := got.(Model); updated.TreeOrganizationCollapsed == nil || !updated.TreeOrganizationCollapsed[organizationFocusKey("acme")] {
		t.Fatal("organization map not initialized")
	}

	m = repositoryTestModel()
	m.RepositoryCollapsed = nil
	repo := findMouseTarget(t, m, "api 2")
	got, _ = m.handleMouseMsg(tea.MouseMsg{X: repo.X, Y: repo.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if updated := got.(Model); updated.RepositoryCollapsed == nil || !updated.RepositoryCollapsed[repositoryFocusKey("acme", "api")] {
		t.Fatal("repository map not initialized")
	}
}

func TestMouseIgnoresNonActivatingAndInertCoordinates(t *testing.T) {
	m := repositoryTestModel()
	before := m.SelectedKey
	target := findMouseTarget(t, m, "Add retry budget")
	statusY := len(strings.Split(m.View(), "\n")) - 1
	for _, msg := range []tea.MouseMsg{
		{X: 0, Y: 0, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},           // header
		{X: 0, Y: statusY - 1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}, // blank
		{X: 0, Y: statusY, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},     // status
		{X: target.X, Y: target.Y, Action: tea.MouseActionRelease, Button: tea.MouseButtonLeft},
		{X: target.X, Y: target.Y, Action: tea.MouseActionMotion, Button: tea.MouseButtonLeft},
		{X: target.X, Y: target.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonRight},
		{X: target.X, Y: target.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonWheelDown},
		{X: -1, Y: target.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
		{X: m.Width, Y: target.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
		{X: target.X, Y: -1, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
		{X: target.X, Y: 999, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft},
	} {
		got, cmd := m.handleMouseMsg(msg)
		if got.(Model).SelectedKey != before || cmd != nil {
			t.Fatalf("activated %#v", msg)
		}
	}

	short := findMouseTarget(t, m, "acme 3")
	if got, cmd := m.handleMouseMsg(tea.MouseMsg{X: m.mouseTargets()[0].Width, Y: short.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}); got.(Model).SelectedKey != before || cmd != nil {
		t.Fatal("click beyond rendered row activated")
	}
}

func TestMouseTargetsAreInertOutsideNormalListState(t *testing.T) {
	base := repositoryTestModel()
	target := findMouseTarget(t, base, "Add retry budget")
	click := tea.MouseMsg{X: target.X, Y: target.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft}

	tests := []struct {
		name  string
		alter func(*Model)
	}{
		{"modal", func(m *Model) { m.Modal = ModalState{Type: ModalHelp} }},
		{"loading", func(m *Model) { m.IsLoading = true }},
		{"error", func(m *Model) { m.Error = errors.New("failed") }},
		{"empty", func(m *Model) { m.Groups = nil }},
		{"all drafts hidden", func(m *Model) {
			for i := range m.Groups[0].PRs {
				m.Groups[0].PRs[i].IsDraft = true
			}
			m.ShowDrafts = false
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := base
			tc.alter(&m)
			if targets := m.mouseTargets(); len(targets) != 0 {
				t.Fatalf("targets=%#v", targets)
			}
			got, cmd := m.handleMouseMsg(click)
			if got.(Model).SelectedKey != base.SelectedKey || cmd != nil {
				t.Fatal("inert state activated")
			}
		})
	}
}

func TestMouseUnknownTargetIsInert(t *testing.T) {
	m := repositoryTestModel()
	before := m.SelectedKey
	got, cmd := m.activateMouseTarget(mouseTarget{Key: "missing/pr#999"})
	if got.(Model).SelectedKey != before || cmd != nil {
		t.Fatal("unknown target activated")
	}
}

func TestMouseSuppressesTooNarrowSentinelTargets(t *testing.T) {
	m := repositoryTestModel()
	m.Width = 23
	if targets := m.mouseTargets(); len(targets) != 0 {
		t.Fatalf("targets=%#v", targets)
	}
}

func TestMouseTargetsTrackCollapseDraftsAndRendererClipping(t *testing.T) {
	m := repositoryTestModel()
	m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] = true
	if target := findOptionalMouseTarget(m, "Add retry budget"); target != nil {
		t.Fatal("collapsed PR target exists")
	}
	m.RepositoryCollapsed[repositoryFocusKey("acme", "api")] = false
	m.Groups[0].PRs[2].IsDraft = true
	m.ShowDrafts = false
	if target := findOptionalMouseTarget(m, "Correct callback validation"); target != nil {
		t.Fatal("hidden draft target exists")
	}
	m.ShowDrafts = true
	m.Height = 4
	targets := m.mouseTargets()
	if len(targets) == 0 {
		t.Fatal("expected displayed suffix target")
	}
	for _, target := range targets {
		if target.Y < 0 || target.Y >= 4 {
			t.Fatalf("unclipped target=%#v", target)
		}
	}
	last := targets[len(targets)-1]
	var opened string
	m.OpenPR = func(org, repo string, number int) error {
		opened = org + "/" + repo
		return nil
	}
	got, cmd := m.handleMouseMsg(tea.MouseMsg{X: 0, Y: last.Y, Action: tea.MouseActionPress, Button: tea.MouseButtonLeft})
	if got.(Model).SelectedKey != last.Key || cmd == nil {
		t.Fatalf("clipped click selected=%q target=%q", got.(Model).SelectedKey, last.Key)
	}
	_ = cmd()
	if opened != "acme/web" {
		t.Fatalf("opened=%q", opened)
	}

	m.Height = 1 // only the final status line survives Bubble Tea's suffix clipping
	if targets := m.mouseTargets(); len(targets) != 0 {
		t.Fatalf("height-one targets=%#v", targets)
	}
}

type targetCoordinate struct{ X, Y int }

func findMouseTarget(t *testing.T, m Model, needle string) targetCoordinate {
	t.Helper()
	target := findOptionalMouseTarget(m, needle)
	if target == nil {
		t.Fatalf("target %q missing", needle)
	}
	return *target
}
func findOptionalMouseTarget(m Model, needle string) *targetCoordinate {
	for _, target := range m.mouseTargets() {
		if strings.Contains(target.Text, needle) {
			return &targetCoordinate{X: min(2, max(0, target.Width-1)), Y: target.Y}
		}
	}
	return nil
}
