package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/jgordijn/pr-dashboard/internal/config"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

// newTestModelWithStyles creates a test model with styles initialized.
func newTestModelWithStyles() Model {
	cfg := &config.Config{
		General: config.GeneralConfig{
			Username:        "testuser",
			RefreshInterval: 30,
		},
		Display: config.DisplayConfig{
			ShowDrafts:  true,
			InitialMode: "full",
		},
		Organizations: []config.OrganizationConfig{
			{Login: "org1"},
		},
	}
	m := NewModel(cfg, nil)
	m.Styles = NewStyles()
	m.Keys = NewKeyMap()
	return m
}

// TestViewNoModal tests the View method when no modal is shown.
func TestViewNoModal(t *testing.T) {
	m := newTestModelWithStyles()
	m.Groups = []model.PRGroup{
		{
			Organization: "testorg",
			PRs: []model.PullRequest{
				{
					Key:         "testorg/repo#1",
					Title:       "Test PR",
					Author:      "testuser",
					Number:      1,
					DaysOpen:    5,
					CheckStatus: model.CheckStatusPassing,
					MergeStatus: model.MergeStatusReady,
				},
			},
		},
	}
	m.SelectedKey = "testorg/repo#1"

	view := m.View()

	// Verify header is present
	if !strings.Contains(view, "PR Dashboard") {
		t.Error("expected header to contain 'PR Dashboard'")
	}

	// Verify status bar has help hint
	if !strings.Contains(view, "Press ? for help") {
		t.Error("expected status bar to contain help hint")
	}
}

// TestViewWithModal tests the View method when a modal is shown.
func TestViewWithModal(t *testing.T) {
	m := newTestModelWithStyles()
	m.Modal = ModalState{
		Type:    ModalSuccess,
		Title:   "Success",
		Message: "Branch updated successfully",
	}

	view := m.View()

	// Verify modal content is present
	if !strings.Contains(view, "Success") {
		t.Error("expected modal title in view")
	}
	if !strings.Contains(view, "Branch updated successfully") {
		t.Error("expected modal message in view")
	}
}

// TestRenderHeader tests the renderHeader method.
func TestRenderHeader(t *testing.T) {
	m := newTestModelWithStyles()

	// Normal header (not loading)
	m.IsLoading = false
	header := m.renderHeader()
	if !strings.Contains(header, "PR Dashboard") {
		t.Error("expected header to contain 'PR Dashboard'")
	}
	// When not loading, header should not contain spinner
	spinnerView := m.Spinner.View()
	if strings.Contains(header, spinnerView) && spinnerView != "" {
		t.Error("expected header to not contain spinner when not loading")
	}

	// Loading header should contain spinner
	m.IsLoading = true
	header = m.renderHeader()
	if !strings.Contains(header, "PR Dashboard") {
		t.Error("expected loading header to still contain 'PR Dashboard'")
	}
	// Header should contain spinner output when loading
	spinnerView = m.Spinner.View()
	if spinnerView != "" && !strings.Contains(header, spinnerView) {
		t.Error("expected header to show spinner when loading")
	}
}

// TestRenderLoading tests the renderLoading method.
func TestRenderLoading(t *testing.T) {
	m := newTestModelWithStyles()
	loading := m.renderLoading()

	if !strings.Contains(loading, "Loading pull requests") {
		t.Error("expected loading message")
	}
}

// TestRenderError tests the renderError method.
func TestRenderError(t *testing.T) {
	m := newTestModelWithStyles()
	m.Error = &testViewError{msg: "API rate limit exceeded"}

	errorView := m.renderError()

	if !strings.Contains(errorView, "Error:") {
		t.Error("expected 'Error:' prefix")
	}
	if !strings.Contains(errorView, "API rate limit exceeded") {
		t.Error("expected error message in view")
	}
}

type testViewError struct {
	msg string
}

func (e *testViewError) Error() string {
	return e.msg
}

// TestRenderEmpty tests the renderEmpty method per pr-display/spec.md.
func TestRenderEmpty(t *testing.T) {
	m := newTestModelWithStyles()
	empty := m.renderEmpty()

	if !strings.Contains(empty, "No open PRs") {
		t.Error("expected empty state message")
	}
	// Check for the emoji as per spec
	if !strings.Contains(empty, "🎉") {
		t.Error("expected celebration emoji in empty state")
	}
}

// TestRenderPRList tests the renderPRList method.
func TestRenderPRList(t *testing.T) {
	m := newTestModelWithStyles()
	m.Groups = []model.PRGroup{
		{
			Organization: "org1",
			Collapsed:    false,
			PRs: []model.PullRequest{
				{Key: "org1/repo#1", Title: "First PR", Number: 1, Author: "user1"},
				{Key: "org1/repo#2", Title: "Second PR", Number: 2, Author: "user2"},
			},
		},
		{
			Organization: "org2",
			Collapsed:    true,
			PRs: []model.PullRequest{
				{Key: "org2/repo#3", Title: "Third PR", Number: 3, Author: "user3"},
			},
		},
	}

	list := m.renderPRList()

	// Verify expanded org shows PRs
	if !strings.Contains(list, "org1") {
		t.Error("expected org1 in list")
	}
	if !strings.Contains(list, "#1") {
		t.Error("expected PR #1 in list")
	}
	if !strings.Contains(list, "#2") {
		t.Error("expected PR #2 in list")
	}

	// Verify collapsed org shows header but not PRs
	if !strings.Contains(list, "org2") {
		t.Error("expected org2 header in list")
	}
	if strings.Contains(list, "#3") {
		t.Error("PR #3 should be hidden because org2 is collapsed")
	}
}

// TestRenderPRListDraftFiltering tests draft PR filtering.
func TestRenderPRListDraftFiltering(t *testing.T) {
	m := newTestModelWithStyles()
	m.Groups = []model.PRGroup{
		{
			Organization: "org1",
			PRs: []model.PullRequest{
				{Key: "org1/repo#1", Title: "Regular PR", Number: 1, IsDraft: false},
				{Key: "org1/repo#2", Title: "Draft PR", Number: 2, IsDraft: true},
			},
		},
	}

	// With drafts shown
	m.ShowDrafts = true
	list := m.renderPRList()
	if !strings.Contains(list, "#1") {
		t.Error("expected regular PR in list")
	}
	if !strings.Contains(list, "#2") {
		t.Error("expected draft PR in list when drafts shown")
	}

	// With drafts hidden
	m.ShowDrafts = false
	list = m.renderPRList()
	if !strings.Contains(list, "#1") {
		t.Error("expected regular PR in list")
	}
	if strings.Contains(list, "#2") {
		t.Error("draft PR should be hidden")
	}
}

// TestRenderOrgHeader tests organization header rendering.
func TestRenderOrgHeader(t *testing.T) {
	m := newTestModelWithStyles()
	m.ShowDrafts = true

	tests := []struct {
		name      string
		group     model.PRGroup
		expectExp bool
	}{
		{
			name: "expanded org",
			group: model.PRGroup{
				Organization: "myorg",
				Collapsed:    false,
				PRs: []model.PullRequest{
					{Key: "myorg/repo#1"},
					{Key: "myorg/repo#2"},
				},
			},
			expectExp: true,
		},
		{
			name: "collapsed org",
			group: model.PRGroup{
				Organization: "otherorg",
				Collapsed:    true,
				PRs: []model.PullRequest{
					{Key: "otherorg/repo#1"},
				},
			},
			expectExp: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			header := m.renderOrgHeader(tc.group)

			// Verify org name
			if !strings.Contains(header, tc.group.Organization) {
				t.Errorf("expected org name %s in header", tc.group.Organization)
			}

			// Verify PR count
			if !strings.Contains(header, "PRs") {
				t.Error("expected PR count in header")
			}

			// Verify correct indicator
			if tc.expectExp {
				if !strings.Contains(header, m.Styles.ExpandedIndicator) {
					t.Error("expected expanded indicator")
				}
			} else {
				if !strings.Contains(header, m.Styles.CollapsedIndicator) {
					t.Error("expected collapsed indicator")
				}
			}
		})
	}
}

// TestRenderOrgHeaderDraftCount tests that draft count is correct when filtered.
func TestRenderOrgHeaderDraftCount(t *testing.T) {
	m := newTestModelWithStyles()

	group := model.PRGroup{
		Organization: "org",
		PRs: []model.PullRequest{
			{Key: "org/repo#1", IsDraft: false},
			{Key: "org/repo#2", IsDraft: true},
			{Key: "org/repo#3", IsDraft: false},
		},
	}

	// With drafts shown - should show 3 PRs
	m.ShowDrafts = true
	header := m.renderOrgHeader(group)
	if !strings.Contains(header, "3 PRs") {
		t.Errorf("expected '3 PRs' when drafts shown, got: %s", header)
	}

	// With drafts hidden - should show 2 PRs
	m.ShowDrafts = false
	header = m.renderOrgHeader(group)
	if !strings.Contains(header, "2 PRs") {
		t.Errorf("expected '2 PRs' when drafts hidden, got: %s", header)
	}
}

// TestRenderPRRow tests PR row rendering with different display modes.
func TestRenderPRRow(t *testing.T) {
	m := newTestModelWithStyles()
	pr := model.PullRequest{
		Key:         "org/repo#42",
		Title:       "Fix important bug",
		Author:      "developer",
		Number:      42,
		DaysOpen:    3,
		CheckStatus: model.CheckStatusPassing,
		MergeStatus: model.MergeStatusReady,
	}

	tests := []struct {
		mode     model.DisplayMode
		contains []string
	}{
		{
			mode:     model.DisplayModeFull,
			contains: []string{"#42", "Fix important bug", "developer", "3d"},
		},
		{
			mode:     model.DisplayModeCompact,
			contains: []string{"#42", "Fix important bug", "developer"},
		},
		{
			mode:     model.DisplayModeMinimal,
			contains: []string{"#42", "Fix important bug", "developer"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.mode.String(), func(t *testing.T) {
			m.DisplayMode = tc.mode
			row := m.renderPRRow(pr)

			for _, expected := range tc.contains {
				if !strings.Contains(row, expected) {
					t.Errorf("expected %q in row for %s mode", expected, tc.mode)
				}
			}
		})
	}
}

// TestRenderPRRowSelected tests selection highlighting.
func TestRenderPRRowSelected(t *testing.T) {
	m := newTestModelWithStyles()
	pr := model.PullRequest{
		Key:    "org/repo#1",
		Title:  "Test PR",
		Number: 1,
	}

	// Selected PR
	m.SelectedKey = "org/repo#1"
	row := m.renderPRRow(pr)
	// The row should have selection styling applied
	// We can't easily test for ANSI codes, but we can verify the content is there
	if !strings.Contains(row, "#1") {
		t.Error("selected row should contain PR number")
	}
}

// TestRenderPRRowChanged tests change highlighting.
func TestRenderPRRowChanged(t *testing.T) {
	m := newTestModelWithStyles()
	pr := model.PullRequest{
		Key:    "org/repo#1",
		Title:  "Test PR",
		Number: 1,
	}

	// Mark as changed
	m.ChangedKeys = map[string]time.Time{
		"org/repo#1": time.Now(),
	}

	row := m.renderPRRow(pr)
	// Verify content is present (styling is applied)
	if !strings.Contains(row, "#1") {
		t.Error("changed row should contain PR number")
	}
}

// TestRenderPRRowDraft tests draft PR styling.
func TestRenderPRRowDraft(t *testing.T) {
	m := newTestModelWithStyles()
	pr := model.PullRequest{
		Key:     "org/repo#1",
		Title:   "Draft PR",
		Number:  1,
		IsDraft: true,
	}

	row := m.renderPRRow(pr)
	// Draft PRs should have dimmed styling
	if !strings.Contains(row, "#1") {
		t.Error("draft row should contain PR number")
	}
}

// TestRenderPRRowFull tests full display mode details.
func TestRenderPRRowFull(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		name     string
		pr       model.PullRequest
		contains []string
		excludes []string
	}{
		{
			name: "with draft badge",
			pr: model.PullRequest{
				Key:     "org/repo#1",
				Title:   "Draft feature",
				Number:  1,
				Author:  "dev",
				IsDraft: true,
			},
			contains: []string{"#1", "Draft feature", "dev", "[DRAFT]"},
		},
		{
			name: "without draft badge",
			pr: model.PullRequest{
				Key:     "org/repo#2",
				Title:   "Ready feature",
				Number:  2,
				Author:  "dev",
				IsDraft: false,
			},
			contains: []string{"#2", "Ready feature", "dev"},
			excludes: []string{"[DRAFT]"},
		},
		{
			name: "with unresolved threads",
			pr: model.PullRequest{
				Key:               "org/repo#3",
				Title:             "Needs review",
				Number:            3,
				Author:            "dev",
				UnresolvedThreads: "5",
			},
			contains: []string{"#3", "threads:5"},
		},
		{
			name: "without threads when zero",
			pr: model.PullRequest{
				Key:               "org/repo#4",
				Title:             "Clean PR",
				Number:            4,
				Author:            "dev",
				UnresolvedThreads: "0",
			},
			excludes: []string{"threads:"},
		},
		{
			name: "with truncated threads count",
			pr: model.PullRequest{
				Key:               "org/repo#5",
				Title:             "Many threads",
				Number:            5,
				Author:            "dev",
				UnresolvedThreads: "100+",
			},
			contains: []string{"threads:100+"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			row := m.renderPRRowFull(tc.pr)

			for _, expected := range tc.contains {
				if !strings.Contains(row, expected) {
					t.Errorf("expected %q in row", expected)
				}
			}

			for _, excluded := range tc.excludes {
				if strings.Contains(row, excluded) {
					t.Errorf("unexpected %q in row", excluded)
				}
			}
		})
	}
}

// TestRenderPRRowCompact tests compact display mode.
func TestRenderPRRowCompact(t *testing.T) {
	m := newTestModelWithStyles()

	pr := model.PullRequest{
		Key:         "org/repo#1",
		Title:       "Test feature",
		Number:      1,
		Author:      "dev",
		CheckStatus: model.CheckStatusPassing,
		MergeStatus: model.MergeStatusReady,
	}

	row := m.renderPRRowCompact(pr)

	// Should have basic info
	if !strings.Contains(row, "#1") {
		t.Error("expected PR number")
	}
	if !strings.Contains(row, "dev") {
		t.Error("expected author")
	}

	// Should have separator
	if !strings.Contains(row, "|") {
		t.Error("expected separator")
	}
}

// TestRenderPRRowMinimal tests minimal display mode.
func TestRenderPRRowMinimal(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		name     string
		pr       model.PullRequest
		contains string
	}{
		{
			name: "ready PR",
			pr: model.PullRequest{
				Key:      "org/repo#1",
				Title:    "Ready",
				Number:   1,
				Author:   "dev",
				DaysOpen: 2,
				IsDraft:  false,
			},
			contains: "[Ready]",
		},
		{
			name: "draft PR",
			pr: model.PullRequest{
				Key:      "org/repo#2",
				Title:    "WIP",
				Number:   2,
				Author:   "dev",
				DaysOpen: 1,
				IsDraft:  true,
			},
			contains: "[Draft]",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			row := m.renderPRRowMinimal(tc.pr)

			if !strings.Contains(row, tc.contains) {
				t.Errorf("expected %q in row, got: %s", tc.contains, row)
			}
			if !strings.Contains(row, tc.pr.Author) {
				t.Error("expected author in minimal row")
			}
		})
	}
}

// TestTruncateTitle tests title truncation with ellipsis.
func TestTruncateTitle(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		name     string
		title    string
		maxLen   int
		expected string
	}{
		{
			name:     "short title unchanged",
			title:    "Short",
			maxLen:   10,
			expected: "Short",
		},
		{
			name:     "exact length unchanged",
			title:    "Exactly10!",
			maxLen:   10,
			expected: "Exactly10!",
		},
		{
			name:     "long title truncated",
			title:    "This is a very long title that needs truncation",
			maxLen:   20,
			expected: "This is a very lo...",
		},
		{
			name:     "very short maxLen",
			title:    "Hello World",
			maxLen:   5,
			expected: "He...",
		},
		{
			name:     "empty title",
			title:    "",
			maxLen:   10,
			expected: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := m.truncateTitle(tc.title, tc.maxLen)
			if result != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, result)
			}
			// Verify truncated title doesn't exceed maxLen
			if len(result) > tc.maxLen {
				t.Errorf("result length %d exceeds maxLen %d", len(result), tc.maxLen)
			}
		})
	}
}

// TestRenderCheckStatus tests check status rendering.
func TestRenderCheckStatus(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		status   model.CheckStatus
		contains string
	}{
		{model.CheckStatusPassing, "passing"},
		{model.CheckStatusFailing, "failing"},
		{model.CheckStatusPending, "pending"},
		{model.CheckStatusNone, "none"},
	}

	for _, tc := range tests {
		t.Run(tc.status.String(), func(t *testing.T) {
			result := m.renderCheckStatus(tc.status)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected %q in result, got: %s", tc.contains, result)
			}
			if !strings.Contains(result, "checks:") {
				t.Error("expected 'checks:' prefix")
			}
		})
	}
}

// TestRenderReviewStatus tests review status rendering.
func TestRenderReviewStatus(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		name      string
		status    model.ReviewStatus
		reviewers []string
		contains  []string
	}{
		{
			name:     "approved",
			status:   model.ReviewStatusApproved,
			contains: []string{"approved"},
		},
		{
			name:     "changes requested",
			status:   model.ReviewStatusChangesRequested,
			contains: []string{"changes"},
		},
		{
			name:     "review required",
			status:   model.ReviewStatusReviewRequired,
			contains: []string{"review"},
		},
		{
			name:     "none",
			status:   model.ReviewStatusNone,
			contains: []string{"no-review"},
		},
		{
			name:      "with reviewers",
			status:    model.ReviewStatusApproved,
			reviewers: []string{"alice", "bob"},
			contains:  []string{"approved", "alice", "bob"},
		},
		{
			name:      "with team reviewer",
			status:    model.ReviewStatusApproved,
			reviewers: []string{"team:core-team"},
			contains:  []string{"approved", "team:core-team"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := m.renderReviewStatus(tc.status, tc.reviewers)

			for _, expected := range tc.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected %q in result, got: %s", expected, result)
				}
			}
		})
	}
}

// TestRenderMergeStatus tests merge status rendering with all states.
func TestRenderMergeStatus(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		status   model.MergeStatus
		contains string
	}{
		{model.MergeStatusReady, "ready"},
		{model.MergeStatusBehind, "behind"},
		{model.MergeStatusBlocked, "blocked"},
		{model.MergeStatusConflicts, "conflicts"},
		{model.MergeStatusDirty, "dirty"},
		{model.MergeStatusUnstable, "unstable"},
		{model.MergeStatusHasHooks, "has-hooks"},
		{model.MergeStatusDraft, "draft"},
		{model.MergeStatusUnknown, "unknown"},
	}

	for _, tc := range tests {
		t.Run(tc.status.String(), func(t *testing.T) {
			result := m.renderMergeStatus(tc.status)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected %q in result, got: %s", tc.contains, result)
			}
		})
	}
}

// TestGetCheckIcon tests check status icon rendering.
func TestGetCheckIcon(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		status   model.CheckStatus
		contains string
	}{
		{model.CheckStatusPassing, "✓"},
		{model.CheckStatusFailing, "✗"},
		{model.CheckStatusPending, "⏳"},
		{model.CheckStatusNone, "-"},
	}

	for _, tc := range tests {
		t.Run(tc.status.String(), func(t *testing.T) {
			result := m.getCheckIcon(tc.status)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected icon %q, got: %s", tc.contains, result)
			}
		})
	}
}

// TestGetReviewIcon tests review status icon rendering.
func TestGetReviewIcon(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		status   model.ReviewStatus
		contains string
	}{
		{model.ReviewStatusApproved, "✓"},
		{model.ReviewStatusChangesRequested, "!"},
		{model.ReviewStatusReviewRequired, "?"},
		{model.ReviewStatusNone, "-"},
	}

	for _, tc := range tests {
		t.Run(tc.status.String(), func(t *testing.T) {
			result := m.getReviewIcon(tc.status)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected icon %q, got: %s", tc.contains, result)
			}
		})
	}
}

// TestGetMergeIcon tests merge status icon rendering.
func TestGetMergeIcon(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		status   model.MergeStatus
		contains string
	}{
		{model.MergeStatusReady, "✓"},
		{model.MergeStatusBehind, "↓"},
		{model.MergeStatusBlocked, "⊘"},
		{model.MergeStatusConflicts, "✗"},
		{model.MergeStatusDirty, "✗"},
		{model.MergeStatusUnknown, "?"},
	}

	for _, tc := range tests {
		t.Run(tc.status.String(), func(t *testing.T) {
			result := m.getMergeIcon(tc.status)
			if !strings.Contains(result, tc.contains) {
				t.Errorf("expected icon %q, got: %s", tc.contains, result)
			}
		})
	}
}

// TestRenderStatusBar tests status bar rendering.
func TestRenderStatusBar(t *testing.T) {
	m := newTestModelWithStyles()

	// Basic status bar
	bar := m.renderStatusBar()

	// Should contain display mode
	if !strings.Contains(bar, "Mode:") {
		t.Error("expected display mode in status bar")
	}

	// Should contain help hint
	if !strings.Contains(bar, "Press ? for help") {
		t.Error("expected help hint in status bar")
	}
}

// TestRenderStatusBarWatchMode tests watch mode indicator.
func TestRenderStatusBarWatchMode(t *testing.T) {
	m := newTestModelWithStyles()
	m.WatchMode = true

	bar := m.renderStatusBar()

	if !strings.Contains(bar, "Watch:") {
		t.Error("expected watch mode indicator")
	}
	if !strings.Contains(bar, "30s") {
		t.Error("expected refresh interval")
	}
}

// TestRenderStatusBarLastRefresh tests last refresh time display.
func TestRenderStatusBarLastRefresh(t *testing.T) {
	m := newTestModelWithStyles()
	m.LastRefresh = time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)

	bar := m.renderStatusBar()

	if !strings.Contains(bar, "Last refresh:") {
		t.Error("expected last refresh time")
	}
	if !strings.Contains(bar, "14:30") {
		t.Error("expected formatted time")
	}
}

// TestRenderStatusBarRateLimit tests rate limit warning display.
func TestRenderStatusBarRateLimit(t *testing.T) {
	m := newTestModelWithStyles()
	m.RateLimit = RateLimitInfo{
		Remaining: 50,
		ResetAt:   time.Date(2024, 1, 15, 15, 0, 0, 0, time.UTC),
	}

	bar := m.renderStatusBar()

	if !strings.Contains(bar, "Rate limit:") {
		t.Error("expected rate limit warning")
	}
	if !strings.Contains(bar, "50") {
		t.Error("expected remaining count")
	}
}

// TestRenderStatusBarNoRateLimitWarning tests no warning when limit is fine.
func TestRenderStatusBarNoRateLimitWarning(t *testing.T) {
	m := newTestModelWithStyles()
	m.RateLimit = RateLimitInfo{
		Remaining: 5000,
		ResetAt:   time.Now().Add(time.Hour),
	}

	bar := m.renderStatusBar()

	if strings.Contains(bar, "Rate limit:") {
		t.Error("should not show rate limit when remaining > 100")
	}
}

// TestRenderWithModal tests modal overlay rendering.
func TestRenderWithModal(t *testing.T) {
	m := newTestModelWithStyles()
	m.Modal = ModalState{
		Type:    ModalSuccess,
		Title:   "Success",
		Message: "Operation completed",
	}

	view := m.renderWithModal()

	// Should contain modal content
	if !strings.Contains(view, "Success") {
		t.Error("expected modal title")
	}
	if !strings.Contains(view, "Operation completed") {
		t.Error("expected modal message")
	}
}

// TestRenderMainView tests main view rendering without modal.
func TestRenderMainView(t *testing.T) {
	m := newTestModelWithStyles()
	m.Groups = []model.PRGroup{
		{
			Organization: "org",
			PRs: []model.PullRequest{
				{Key: "org/repo#1", Title: "Test", Number: 1},
			},
		},
	}

	view := m.renderMainView()

	// Should contain header
	if !strings.Contains(view, "PR Dashboard") {
		t.Error("expected header")
	}

	// Should contain status bar
	if !strings.Contains(view, "Press ? for help") {
		t.Error("expected status bar")
	}
}

// TestRenderModal tests modal rendering for different types.
func TestRenderModal(t *testing.T) {
	m := newTestModelWithStyles()

	tests := []struct {
		name      string
		modalType ModalType
		title     string
		message   string
		contains  []string
	}{
		{
			name:      "help modal",
			modalType: ModalHelp,
			contains:  []string{"Keyboard Shortcuts", "j/↓", "k/↑", "q/Esc"},
		},
		{
			name:      "success modal",
			modalType: ModalSuccess,
			title:     "Success",
			message:   "Branch updated",
			contains:  []string{"Success", "Branch updated", "dismiss"},
		},
		{
			name:      "error modal",
			modalType: ModalError,
			title:     "Error",
			message:   "Update failed",
			contains:  []string{"Error", "Update failed", "dismiss"},
		},
		{
			name:      "no modal",
			modalType: ModalNone,
			contains:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m.Modal = ModalState{
				Type:    tc.modalType,
				Title:   tc.title,
				Message: tc.message,
			}

			result := m.renderModal()

			for _, expected := range tc.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected %q in modal, got: %s", expected, result)
				}
			}
		})
	}
}

// TestRenderHelpModal tests help modal content.
func TestRenderHelpModal(t *testing.T) {
	m := newTestModelWithStyles()

	help := m.renderHelpModal()

	// Check all navigation keys
	navKeys := []string{"j/↓", "k/↑", "gg", "G", "o", "O"}
	for _, key := range navKeys {
		if !strings.Contains(help, key) {
			t.Errorf("expected navigation key %q in help", key)
		}
	}

	// Check action keys
	actionKeys := []string{"u", "r", "Enter"}
	for _, key := range actionKeys {
		if !strings.Contains(help, key) {
			t.Errorf("expected action key %q in help", key)
		}
	}

	// Check display keys
	displayKeys := []string{"c", "d", "w"}
	for _, key := range displayKeys {
		if !strings.Contains(help, key) {
			t.Errorf("expected display key %q in help", key)
		}
	}

	// Check other keys
	otherKeys := []string{"?", "q/Esc"}
	for _, key := range otherKeys {
		if !strings.Contains(help, key) {
			t.Errorf("expected other key %q in help", key)
		}
	}

	// Check section headers
	sections := []string{"Navigation:", "Actions:", "Display:", "Other:"}
	for _, section := range sections {
		if !strings.Contains(help, section) {
			t.Errorf("expected section %q in help", section)
		}
	}

	// Check dismiss hint
	if !strings.Contains(help, "dismiss") {
		t.Error("expected dismiss hint in help")
	}
}

// TestViewStateTransitions tests View output for different states.
func TestViewStateTransitions(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*Model)
		contains string
	}{
		{
			name: "loading with no data",
			setup: func(m *Model) {
				m.IsLoading = true
				m.Groups = nil
			},
			contains: "Loading",
		},
		{
			name: "error with no data",
			setup: func(m *Model) {
				m.IsLoading = false // Must clear loading state to show error
				m.Error = &testViewError{msg: "network error"}
				m.Groups = nil
			},
			contains: "Error",
		},
		{
			name: "empty after load",
			setup: func(m *Model) {
				m.IsLoading = false // Must clear loading state to show empty
				m.Groups = []model.PRGroup{}
			},
			contains: "No open PRs",
		},
		{
			name: "with data",
			setup: func(m *Model) {
				m.IsLoading = false // Must clear loading state to show data
				m.Groups = []model.PRGroup{
					{
						Organization: "org",
						PRs: []model.PullRequest{
							{Key: "org/repo#1", Title: "Test", Number: 1},
						},
					},
				}
			},
			contains: "#1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModelWithStyles()
			tc.setup(&m)

			view := m.View()

			if !strings.Contains(view, tc.contains) {
				t.Errorf("expected %q in view for state %s", tc.contains, tc.name)
			}
		})
	}
}
