package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

func designPR() model.PullRequest {
	return model.PullRequest{Key: "org/pr-dashboard#351", Organization: "org", Repository: "pr-dashboard", Number: 351, Title: "Add project status glyph rendering", Author: "jeroen", DaysOpen: 2, UnresolvedThreads: "2", CheckStatus: model.CheckStatusPassing, ReviewStatus: model.ReviewStatusApproved, MergeStatus: model.MergeStatusReady}
}

func TestSymbolSetsAreSingleCellAndASCIISetIsASCII(t *testing.T) {
	for name, set := range map[string]SymbolSet{"unicode": UnicodeSymbols, "ascii": ASCIISymbols} {
		for symbolName, glyph := range set.all() {
			if got := lipgloss.Width(glyph); got != 1 {
				t.Errorf("%s %s width = %d", name, symbolName, got)
			}
			if name == "ascii" && !regexp.MustCompile(`^[\x00-\x7f]$`).MatchString(glyph) {
				t.Errorf("ASCII %s = %q", symbolName, glyph)
			}
		}
	}
}

func TestStatusGlyphMappings(t *testing.T) {
	m := newTestModelWithStyles()
	m.Symbols = UnicodeSymbols
	tests := []struct {
		name string
		pr   model.PullRequest
		want string
	}{
		{"passing approved ready", model.PullRequest{CheckStatus: model.CheckStatusPassing, ReviewStatus: model.ReviewStatusApproved, MergeStatus: model.MergeStatusReady}, "✓ ✓ ✓"},
		{"failing changes behind", model.PullRequest{CheckStatus: model.CheckStatusFailing, ReviewStatus: model.ReviewStatusChangesRequested, MergeStatus: model.MergeStatusBehind}, "✗ ! ↓"},
		{"pending required blocked", model.PullRequest{CheckStatus: model.CheckStatusPending, ReviewStatus: model.ReviewStatusReviewRequired, MergeStatus: model.MergeStatusBlocked}, "◐ ? ⊘"},
		{"none none conflict", model.PullRequest{MergeStatus: model.MergeStatusConflicts}, "· · ≠"},
		{"dirty is conflict", model.PullRequest{MergeStatus: model.MergeStatusDirty}, "· · ≠"},
		{"unstable", model.PullRequest{MergeStatus: model.MergeStatusUnstable}, "· · ~"},
		{"hooks are unstable", model.PullRequest{MergeStatus: model.MergeStatusHasHooks}, "· · ~"},
		{"draft", model.PullRequest{IsDraft: true, MergeStatus: model.MergeStatusDraft}, "· · ○"},
		{"draft conflict", model.PullRequest{IsDraft: true, MergeStatus: model.MergeStatusConflicts}, "· · ≠"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripANSI(m.renderStatusTriad(tc.pr)); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestRowsAlignTriadAcrossFrame(t *testing.T) {
	m := newTestModelWithStyles()
	m.Width = 80
	m.DisplayMode = model.DisplayModeFull
	a := designPR()
	a.Repository = "a"
	a.Number = 1
	a.Title = "Short"
	a.Author = "al"
	b := designPR()
	b.Key = "org/long#12345"
	b.Repository = "long-repo-name"
	b.Number = 12345
	b.Title = "A much longer title"
	b.Author = "alexander"
	b.DaysOpen = 22
	m.Groups = []model.PRGroup{{Organization: "org", PRs: []model.PullRequest{a, b}}}
	rows := []string{stripANSI(m.renderPRRow(a)), stripANSI(m.renderPRRow(b))}
	for _, glyph := range []string{"✓ ✓ ✓"} {
		_ = glyph
	}
	first := strings.Index(rows[0], "✓ ✓ ✓")
	second := strings.Index(rows[1], "✓ ✓ ✓")
	if first != second {
		t.Fatalf("triads not aligned: %d != %d\n%s\n%s", first, second, rows[0], rows[1])
	}
	if strings.Index(rows[0], "al") > first {
		t.Fatalf("author must precede triad: %q", rows[0])
	}
}

func TestProjectFirstRowsAndResponsiveWidth(t *testing.T) {
	for _, width := range []int{100, 80, 60, 40, 36, 24} {
		m := newTestModelWithStyles()
		m.Width = width
		m.Symbols = UnicodeSymbols
		pr := designPR()
		row := m.renderPRRow(pr)
		plain := stripANSI(row)
		if !strings.Contains(plain, "#351") {
			t.Fatalf("width %d lost number: %q", width, plain)
		}
		if width >= 40 && !strings.Contains(plain, "pr-dashboard#351") {
			t.Fatalf("width %d lost project: %q", width, plain)
		}
		if !strings.Contains(plain, "✓ ✓ ✓") {
			t.Fatalf("width %d lost triad: %q", width, plain)
		}
		if got := lipgloss.Width(row); got > width {
			t.Fatalf("width %d rendered %d: %q", width, got, plain)
		}
	}
}

func TestGutterStatesAndTooNarrow(t *testing.T) {
	pr := designPR()
	for _, tc := range []struct {
		name, selected string
		changed        bool
		prefix         string
	}{
		{"neither", "", false, "   "}, {"selected", pr.Key, false, "▶  "},
		{"changed", "", true, " ● "}, {"both", pr.Key, true, "▶● "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModelWithStyles()
			m.Symbols = UnicodeSymbols
			m.SelectedKey = tc.selected
			if tc.changed {
				m.ChangedKeys[pr.Key] = m.LastRefresh
			}
			if got := stripANSI(m.renderPRRow(pr)); !strings.HasPrefix(got, tc.prefix) {
				t.Fatalf("gutter = %q", got)
			}
		})
	}
	m := newTestModelWithStyles()
	m.Width = 12
	if got := m.renderPRRow(pr); lipgloss.Width(got) > 12 || !strings.Contains(got, "…") {
		t.Fatalf("narrow row = %q", got)
	}
}

func TestTruncateIdentityAndLeftCells(t *testing.T) {
	if got := truncateIdentity("repo", 1, 20); got != "repo#1" {
		t.Fatal(got)
	}
	if got := truncateIdentity("", 1, 20); got != "#1" {
		t.Fatal(got)
	}
	for _, tc := range []struct {
		in    string
		width int
		want  string
	}{
		{"repo", 9, "repo"}, {"repo", 0, ""}, {"repo", 1, "…"}, {"longrepo", 5, "…repo"}, {"界ab", 3, "…ab"},
	} {
		if got := truncateCellsLeft(tc.in, tc.width); got != tc.want {
			t.Errorf("left(%q,%d)=%q", tc.in, tc.width, got)
		}
	}
}

func TestTruncateCellsPreservesGraphemes(t *testing.T) {
	for _, tc := range []struct {
		in    string
		width int
		want  string
	}{
		{"abcdef", 4, "abc…"},
		{"a界bc", 4, "a界…"},
		{"👩‍💻 coding", 5, "👩‍💻 c…"},
		{"abc", 0, ""},
		{"abc", 1, "…"},
	} {
		if got := truncateCells(tc.in, tc.width); got != tc.want {
			t.Errorf("truncateCells(%q,%d)=%q want %q", tc.in, tc.width, got, tc.want)
		}
		if lipgloss.Width(truncateCells(tc.in, tc.width)) > tc.width {
			t.Fatal("overflow")
		}
	}
}

func TestAgeStrings(t *testing.T) {
	now := time.Now()
	for _, tc := range []struct {
		name    string
		created time.Time
		days    int
		want    string
	}{
		{"fallback", time.Time{}, 4, "4d"}, {"future", now.Add(time.Hour), 0, "0m"},
		{"minutes", now.Add(-30 * time.Minute), 0, "30m"}, {"hours", now.Add(-17 * time.Hour), 0, "17h"},
		{"days", now.Add(-3 * 24 * time.Hour), 0, "3d"}, {"weeks", now.Add(-21 * 24 * time.Hour), 0, "3w"},
		{"months", now.Add(-90 * 24 * time.Hour), 0, "3mo"}, {"years", now.Add(-730 * 24 * time.Hour), 0, "2y"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := ageString(model.PullRequest{CreatedAt: tc.created, DaysOpen: tc.days}); got != tc.want {
				t.Fatalf("got %q want %q", got, tc.want)
			}
		})
	}
}

func TestThreadLabels(t *testing.T) {
	m := newTestModelWithStyles()
	if got := m.threadLabel("100+"); got != "◈99+" {
		t.Fatal(got)
	}
	if got := m.threadLabel("many"); got != "◈many" {
		t.Fatal(got)
	}
}

func TestSelectedDecoderAndCollapsedRollup(t *testing.T) {
	m := newTestModelWithStyles()
	m.Width = 100
	m.Symbols = UnicodeSymbols
	pr := designPR()
	pr.CheckStatus = model.CheckStatusFailing
	pr.MergeStatus = model.MergeStatusBehind
	pr.Mergeable = "MERGEABLE"
	pr.MergeStateStatus = "BEHIND"
	pr.UnresolvedCount = 2
	pr.Reviewers = []string{"alice", "team:core", "bob"}
	m.Groups = []model.PRGroup{{Organization: "org", Collapsed: true, PRs: []model.PullRequest{pr}}}
	m.SelectedKey = pr.Key
	decoder := stripANSI(m.renderSelectedDecoder())
	for _, want := range []string{"pr-dashboard#351", "✗failing", "✓approved", "↓behind", "press u", "◈2", "@alice, /core +1"} {
		if !strings.Contains(decoder, want) {
			t.Errorf("decoder missing %q: %q", want, decoder)
		}
	}
	header := stripANSI(m.renderOrgHeader(m.Groups[0]))
	if !strings.Contains(header, "✗1") || !strings.Contains(header, "↓1") || !strings.Contains(header, "◈2") {
		t.Fatalf("rollup = %q", header)
	}
	m.Width = 8
	if got := stripANSI(m.renderOrgHeader(m.Groups[0])); lipgloss.Width(got) > 8 {
		t.Fatalf("header overflow: %q", got)
	}
	m.Width = 100
	m.Groups[0].Collapsed = false
	if got := stripANSI(m.renderOrgHeader(m.Groups[0])); strings.Contains(got, "↓1") {
		t.Fatalf("expanded rollup = %q", got)
	}
}

func TestCompactDropsAuthorAndAgeFormatting(t *testing.T) {
	m := newTestModelWithStyles()
	m.Width = 100
	m.DisplayMode = model.DisplayModeCompact
	pr := designPR()
	pr.CreatedAt = time.Now().Add(-17 * time.Hour)
	m.Groups = []model.PRGroup{{Organization: "org", PRs: []model.PullRequest{pr}}}
	row := stripANSI(m.renderPRRow(pr))
	if strings.Contains(row, pr.Author) {
		t.Fatalf("compact author: %q", row)
	}
	if !strings.Contains(row, "17h") {
		t.Fatalf("age not formatted: %q", row)
	}
}

func TestDecoderDraftAndStatusBarDegradation(t *testing.T) {
	m := newTestModelWithStyles()
	m.Width = 100
	if got := m.renderSelectedDecoder(); got != "" {
		t.Fatal(got)
	}
	pr := designPR()
	pr.IsDraft = true
	pr.MergeStatus = model.MergeStatusConflicts
	m.Groups = []model.PRGroup{{Organization: "org", PRs: []model.PullRequest{pr}}}
	m.SelectedKey = pr.Key
	if got := stripANSI(m.renderSelectedDecoder()); !strings.Contains(got, "draft") || !strings.Contains(got, "≠conflicts") {
		t.Fatal(got)
	}
	if got := stripANSI(m.renderStatusBar()); !strings.Contains(got, "conflicts") {
		t.Fatal(got)
	}
	m.Width = 24
	m.Config.General.Username = "a-very-long-username"
	if got := stripANSI(m.renderStatusBar()); lipgloss.Width(got) > 24 || !strings.Contains(got, "?help") {
		t.Fatalf("bar must preserve help: %q", got)
	}
	m.SelectedKey = ""
	m.Width = 20
	m.Config.General.Username = "testuser"
	if got := stripANSI(m.renderStatusBar()); !strings.Contains(got, "@testuser · ?help") {
		t.Fatalf("bar drop ladder: %q", got)
	}
}

func TestEffectiveModeLabelsAndASCIIFill(t *testing.T) {
	m := newTestModelWithStyles()
	m.DisplayMode = model.DisplayModeFull
	for _, tc := range []struct {
		width int
		ascii bool
		want  string
	}{{80, false, "full"}, {70, false, "full→compact"}, {50, false, "full→compact"}, {30, false, "full→minimal"}, {30, true, "full>minimal"}} {
		m.Width = tc.width
		m.Config.Display.ASCII = tc.ascii
		if got := m.effectiveModeLabel(); got != tc.want {
			t.Errorf("got %q want %q", got, tc.want)
		}
	}
	m.Width = 80
	m.Config.Display.ASCII = true
	m.Symbols = ASCIISymbols
	pr := designPR()
	pr.CheckStatus = model.CheckStatusFailing
	header := stripANSI(m.renderOrgHeader(model.PRGroup{Organization: "org", Collapsed: true, PRs: []model.PullRequest{pr}}))
	if !strings.Contains(header, "---") {
		t.Fatalf("ASCII fill: %q", header)
	}
}

func TestHugeNumberStillBoundsRow(t *testing.T) {
	m := newTestModelWithStyles()
	m.Width = 24
	pr := designPR()
	pr.Number = 999999999999999999
	if got := m.renderPRRow(pr); lipgloss.Width(got) > 24 {
		t.Fatalf("overflow %d: %q", lipgloss.Width(got), got)
	}
}

func TestASCIIRowIsPureASCII(t *testing.T) {
	m := newTestModelWithStyles()
	m.Width = 80
	m.Symbols = ASCIISymbols
	pr := designPR()
	pr.CheckStatus = model.CheckStatusFailing
	pr.ReviewStatus = model.ReviewStatusChangesRequested
	pr.MergeStatus = model.MergeStatusConflicts
	m.SelectedKey = pr.Key
	plain := stripANSI(m.renderPRRow(pr))
	if !strings.Contains(plain, "x ! X") || !strings.HasPrefix(plain, "> ") {
		t.Fatalf("ASCII row = %q", plain)
	}
	for _, r := range plain {
		if r > 127 {
			t.Fatalf("non-ASCII %q", r)
		}
	}
}

func TestHeaderIncludesTotal(t *testing.T) {
	m := newTestModelWithStyles()
	m.TotalCount = 3
	m.IsLoading = false
	if got := stripANSI(m.renderHeader()); got != "PR Dashboard · 3" {
		t.Fatal(got)
	}
}

func TestHelpContainsCompactLegend(t *testing.T) {
	m := newTestModelWithStyles()
	m.Symbols = UnicodeSymbols
	help := stripANSI(m.renderHelpModal())
	for _, want := range []string{"Symbols", "✓ passing", "! changes", "≠ conflicts", "▶ selected", "◈n threads", "45m/17h/3d", "Merge = v # X", "≠ wins"} {
		if !strings.Contains(help, want) {
			t.Errorf("help missing %q", want)
		}
	}
	if strings.Count(help, "\n")+1 > 24 {
		t.Fatalf("help has too many lines: %d", strings.Count(help, "\n")+1)
	}
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
}
