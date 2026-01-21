package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/jgordijn/pr-dashboard/internal/model"
)

// View implements tea.Model. It renders the current state as a string.
func (m Model) View() string {
	// Handle modal overlay
	if m.Modal.Type != ModalNone {
		return m.renderWithModal()
	}

	var b strings.Builder

	// Header with title
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Main content area
	if m.IsLoading && len(m.Groups) == 0 {
		b.WriteString(m.renderLoading())
	} else if m.Error != nil && len(m.Groups) == 0 {
		b.WriteString(m.renderError())
	} else if len(m.Groups) == 0 || m.countVisiblePRs() == 0 {
		b.WriteString(m.renderEmpty())
	} else {
		b.WriteString(m.renderPRList())
	}

	// Status bar
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// renderHeader renders the application header.
func (m Model) renderHeader() string {
	title := "PR Dashboard"
	if m.IsLoading {
		title += " " + m.Spinner.View()
	}
	return m.Styles.HeaderStyle.Render(title)
}

// renderLoading renders the loading state with spinner.
func (m Model) renderLoading() string {
	return m.Spinner.View() + " " + m.Styles.DimStyle.Render("Loading pull requests...")
}

// renderError renders the error state.
func (m Model) renderError() string {
	return m.Styles.StatusFailingStyle.Render(fmt.Sprintf("Error: %v", m.Error))
}

// renderEmpty renders the empty state per pr-display/spec.md.
func (m Model) renderEmpty() string {
	return m.Styles.NormalStyle.Render("No open PRs - nice work! 🎉")
}

// renderPRList renders the PR list grouped by organization.
func (m Model) renderPRList() string {
	var b strings.Builder

	for _, group := range m.Groups {
		// Render organization header
		b.WriteString(m.renderOrgHeader(group))
		b.WriteString("\n")

		// Skip PRs if collapsed
		if group.Collapsed {
			continue
		}

		// Render PRs in group
		for _, pr := range group.PRs {
			// Skip drafts if hidden
			if !m.ShowDrafts && pr.IsDraft {
				continue
			}
			b.WriteString(m.renderPRRow(pr))
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderOrgHeader renders an organization header with collapse indicator.
func (m Model) renderOrgHeader(group model.PRGroup) string {
	indicator := m.Styles.ExpandedIndicator
	if group.Collapsed {
		indicator = m.Styles.CollapsedIndicator
	}

	// Count visible PRs in this group
	visibleCount := 0
	for _, pr := range group.PRs {
		if m.ShowDrafts || !pr.IsDraft {
			visibleCount++
		}
	}

	header := fmt.Sprintf("%s %s (%d PRs)", indicator, group.Organization, visibleCount)
	return m.Styles.HeaderStyle.Render(header)
}

// renderPRRow renders a single PR row based on display mode.
func (m Model) renderPRRow(pr model.PullRequest) string {
	isSelected := pr.Key == m.SelectedKey
	isChanged := false
	if _, ok := m.ChangedKeys[pr.Key]; ok {
		isChanged = true
	}

	var row string
	switch m.DisplayMode {
	case model.DisplayModeFull:
		row = m.renderPRRowFull(pr)
	case model.DisplayModeCompact:
		row = m.renderPRRowCompact(pr)
	case model.DisplayModeMinimal:
		row = m.renderPRRowMinimal(pr)
	default:
		row = m.renderPRRowFull(pr)
	}

	// Apply styling
	if isSelected {
		row = m.Styles.SelectedStyle.Render(row)
	} else if isChanged {
		row = m.Styles.ChangedStyle.Render(row)
	} else if pr.IsDraft {
		row = m.Styles.DraftStyle.Render(row)
	}

	return row
}

// renderPRRowFull renders a PR row in full display mode.
// Format: #123 Title [Draft] Author 3d | Checks | Reviews | Merge | Threads
func (m Model) renderPRRowFull(pr model.PullRequest) string {
	var parts []string

	// PR number and title
	title := m.truncateTitle(pr.Title, 40)
	prInfo := fmt.Sprintf("#%d %s", pr.Number, title)
	parts = append(parts, prInfo)

	// Draft badge
	if pr.IsDraft {
		parts = append(parts, "[DRAFT]")
	}

	// Author and age
	parts = append(parts, pr.Author)
	parts = append(parts, fmt.Sprintf("%dd", pr.DaysOpen))

	// Separator
	parts = append(parts, "|")

	// Check status
	parts = append(parts, m.renderCheckStatus(pr.CheckStatus))

	// Review status with reviewers
	parts = append(parts, m.renderReviewStatus(pr.ReviewStatus, pr.Reviewers))

	// Merge status
	parts = append(parts, m.renderMergeStatus(pr.MergeStatus))

	// Unresolved threads
	if pr.UnresolvedThreads != "0" {
		parts = append(parts, fmt.Sprintf("threads:%s", pr.UnresolvedThreads))
	}

	return strings.Join(parts, " ")
}

// renderPRRowCompact renders a PR row in compact display mode.
// Format: #123 Title Author | Icons only with counts
func (m Model) renderPRRowCompact(pr model.PullRequest) string {
	var parts []string

	// PR number and title
	title := m.truncateTitle(pr.Title, 30)
	parts = append(parts, fmt.Sprintf("#%d %s", pr.Number, title))

	// Author
	parts = append(parts, pr.Author)

	// Separator
	parts = append(parts, "|")

	// Status icons
	parts = append(parts, m.getCheckIcon(pr.CheckStatus))
	parts = append(parts, m.getReviewIcon(pr.ReviewStatus))
	parts = append(parts, m.getMergeIcon(pr.MergeStatus))

	return strings.Join(parts, " ")
}

// renderPRRowMinimal renders a PR row in minimal display mode.
// Format: #123 Title Author
func (m Model) renderPRRowMinimal(pr model.PullRequest) string {
	title := m.truncateTitle(pr.Title, 50)
	status := "Ready"
	if pr.IsDraft {
		status = "Draft"
	}
	return fmt.Sprintf("#%d %s [%s] %s %dd", pr.Number, title, status, pr.Author, pr.DaysOpen)
}

// truncateTitle truncates a title with ellipsis if too long.
func (m Model) truncateTitle(title string, maxLen int) string {
	if len(title) <= maxLen {
		return title
	}
	return title[:maxLen-3] + "..."
}

// renderCheckStatus renders the check status with color.
func (m Model) renderCheckStatus(status model.CheckStatus) string {
	var style lipgloss.Style
	var text string

	switch status {
	case model.CheckStatusPassing:
		style = m.Styles.StatusPassingStyle
		text = "checks:passing"
	case model.CheckStatusFailing:
		style = m.Styles.StatusFailingStyle
		text = "checks:failing"
	case model.CheckStatusPending:
		style = m.Styles.StatusPendingStyle
		text = "checks:pending"
	default:
		style = m.Styles.StatusNoneStyle
		text = "checks:none"
	}

	return style.Render(text)
}

// renderReviewStatus renders the review status with reviewers.
func (m Model) renderReviewStatus(status model.ReviewStatus, reviewers []string) string {
	var style lipgloss.Style
	var text string

	switch status {
	case model.ReviewStatusApproved:
		style = m.Styles.StatusPassingStyle
		text = "approved"
	case model.ReviewStatusChangesRequested:
		style = m.Styles.StatusChangesRequestedStyle
		text = "changes"
	case model.ReviewStatusReviewRequired:
		style = m.Styles.StatusPendingStyle
		text = "review"
	default:
		style = m.Styles.StatusNoneStyle
		text = "no-review"
	}

	if len(reviewers) > 0 {
		text += ":" + strings.Join(reviewers, ",")
	}

	return style.Render(text)
}

// renderMergeStatus renders the merge status with appropriate color.
func (m Model) renderMergeStatus(status model.MergeStatus) string {
	var style lipgloss.Style
	var text string

	switch status {
	case model.MergeStatusReady:
		style = m.Styles.MergeReadyStyle
		text = "ready"
	case model.MergeStatusBehind:
		style = m.Styles.MergeBehindStyle
		text = "behind"
	case model.MergeStatusBlocked:
		style = m.Styles.MergeBlockedStyle
		text = "blocked"
	case model.MergeStatusConflicts:
		style = m.Styles.MergeConflictStyle
		text = "conflicts"
	case model.MergeStatusDirty:
		style = m.Styles.MergeConflictStyle
		text = "dirty"
	case model.MergeStatusUnstable:
		style = m.Styles.MergeBehindStyle
		text = "unstable"
	case model.MergeStatusHasHooks:
		style = m.Styles.MergeBehindStyle
		text = "has-hooks"
	case model.MergeStatusDraft:
		style = m.Styles.MergeUnknownStyle
		text = "draft"
	default:
		style = m.Styles.MergeUnknownStyle
		text = "unknown"
	}

	return style.Render(text)
}

// getCheckIcon returns icon for check status.
func (m Model) getCheckIcon(status model.CheckStatus) string {
	switch status {
	case model.CheckStatusPassing:
		return m.Styles.StatusPassingStyle.Render("✓")
	case model.CheckStatusFailing:
		return m.Styles.StatusFailingStyle.Render("✗")
	case model.CheckStatusPending:
		return m.Styles.StatusPendingStyle.Render("⏳")
	default:
		return m.Styles.StatusNoneStyle.Render("-")
	}
}

// getReviewIcon returns icon for review status.
func (m Model) getReviewIcon(status model.ReviewStatus) string {
	switch status {
	case model.ReviewStatusApproved:
		return m.Styles.StatusPassingStyle.Render("✓")
	case model.ReviewStatusChangesRequested:
		return m.Styles.StatusChangesRequestedStyle.Render("!")
	case model.ReviewStatusReviewRequired:
		return m.Styles.StatusPendingStyle.Render("?")
	default:
		return m.Styles.StatusNoneStyle.Render("-")
	}
}

// getMergeIcon returns icon for merge status.
func (m Model) getMergeIcon(status model.MergeStatus) string {
	switch status {
	case model.MergeStatusReady:
		return m.Styles.MergeReadyStyle.Render("✓")
	case model.MergeStatusBehind:
		return m.Styles.MergeBehindStyle.Render("↓")
	case model.MergeStatusBlocked:
		return m.Styles.MergeBlockedStyle.Render("⊘")
	case model.MergeStatusConflicts, model.MergeStatusDirty:
		return m.Styles.MergeConflictStyle.Render("✗")
	default:
		return m.Styles.MergeUnknownStyle.Render("?")
	}
}

// renderStatusBar renders the bottom status bar.
func (m Model) renderStatusBar() string {
	var parts []string

	// Watch mode indicator
	if m.WatchMode {
		parts = append(parts, fmt.Sprintf("Watch: %ds", m.Config.General.RefreshInterval))
	}

	// Display mode
	parts = append(parts, fmt.Sprintf("Mode: %s", m.DisplayMode))

	// Last refresh time
	if !m.LastRefresh.IsZero() {
		parts = append(parts, fmt.Sprintf("Last refresh: %s", m.LastRefresh.Format("15:04")))
	}

	// Rate limit warning
	if m.RateLimit.Remaining > 0 && m.RateLimit.Remaining <= 100 {
		parts = append(parts, fmt.Sprintf("Rate limit: %d (resets %s)",
			m.RateLimit.Remaining, m.RateLimit.ResetAt.Format("15:04")))
	}

	// Help hint
	parts = append(parts, "Press ? for help")

	return m.Styles.StatusBarStyle.Render(strings.Join(parts, " | "))
}

// renderWithModal renders the main view with a modal overlay.
func (m Model) renderWithModal() string {
	var b strings.Builder

	// Render background (dimmed main view)
	b.WriteString(m.Styles.DimStyle.Render(m.renderMainView()))
	b.WriteString("\n\n")

	// Render modal
	b.WriteString(m.renderModal())

	return b.String()
}

// renderMainView renders the main content without modal logic.
func (m Model) renderMainView() string {
	var b strings.Builder
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	if m.IsLoading && len(m.Groups) == 0 {
		b.WriteString(m.renderLoading())
	} else if m.Error != nil && len(m.Groups) == 0 {
		b.WriteString(m.renderError())
	} else if len(m.Groups) == 0 || m.countVisiblePRs() == 0 {
		b.WriteString(m.renderEmpty())
	} else {
		b.WriteString(m.renderPRList())
	}

	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// renderModal renders the current modal.
func (m Model) renderModal() string {
	switch m.Modal.Type {
	case ModalHelp:
		return m.renderHelpModal()
	case ModalSuccess:
		return m.Styles.ModalSuccessStyle.Render(
			m.Styles.ModalTitleStyle.Render(m.Modal.Title) + "\n\n" +
				m.Modal.Message + "\n\n" +
				m.Styles.DimStyle.Render("Press Enter, q, or Esc to dismiss"))
	case ModalError:
		return m.Styles.ModalErrorStyle.Render(
			m.Styles.ModalTitleStyle.Render(m.Modal.Title) + "\n\n" +
				m.Modal.Message + "\n\n" +
				m.Styles.DimStyle.Render("Press Enter, q, or Esc to dismiss"))
	default:
		return ""
	}
}

// renderHelpModal renders the help modal with key bindings.
func (m Model) renderHelpModal() string {
	var b strings.Builder

	b.WriteString(m.Styles.ModalTitleStyle.Render("Keyboard Shortcuts"))
	b.WriteString("\n\n")

	// Navigation
	b.WriteString("Navigation:\n")
	b.WriteString("  j/↓     Move down\n")
	b.WriteString("  k/↑     Move up\n")
	b.WriteString("  gg      Go to top\n")
	b.WriteString("  G       Go to bottom\n")
	b.WriteString("  o       Toggle org collapse\n")
	b.WriteString("  O       Toggle all orgs\n")
	b.WriteString("\n")

	// Actions
	b.WriteString("Actions:\n")
	b.WriteString("  u       Update branch\n")
	b.WriteString("  r       Refresh\n")
	b.WriteString("  Enter   Open in browser\n")
	b.WriteString("\n")

	// Display
	b.WriteString("Display:\n")
	b.WriteString("  c       Cycle display mode\n")
	b.WriteString("  d       Toggle drafts\n")
	b.WriteString("  w       Toggle watch mode\n")
	b.WriteString("\n")

	// Other
	b.WriteString("Other:\n")
	b.WriteString("  ?       Help\n")
	b.WriteString("  q/Esc   Quit\n")
	b.WriteString("\n")

	b.WriteString(m.Styles.DimStyle.Render("Press Enter, q, or Esc to dismiss"))

	return m.Styles.ModalStyle.Render(b.String())
}
