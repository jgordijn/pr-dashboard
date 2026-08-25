package tui

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/rivo/uniseg"

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
	if m.TotalCount > 0 {
		title += fmt.Sprintf(" · %d", m.TotalCount)
	}
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

// renderPRList renders the active organization or repository projection.
func (m Model) renderPRList() string {
	if m.GroupingMode == model.GroupingModeRepository {
		return m.renderRepositoryTree()
	}
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

func (m Model) renderRepositoryTree() string {
	var b strings.Builder
	for _, organization := range m.visibleTreeOrganizations() {
		organizationKey := organizationFocusKey(organization.Organization)
		b.WriteString(m.renderTreeOrganizationHeader(organization))
		b.WriteByte('\n')
		if m.TreeOrganizationCollapsed[organizationKey] {
			continue
		}
		for _, group := range organization.Repositories {
			key := repositoryFocusKey(group.Organization, group.Repository)
			b.WriteString(m.renderRepositoryHeader(group))
			b.WriteByte('\n')
			if m.RepositoryCollapsed[key] {
				continue
			}
			for i, pr := range group.PRs {
				b.WriteString(m.renderRepositoryPRRow(pr, i == len(group.PRs)-1))
				b.WriteByte('\n')
			}
		}
	}
	return b.String()
}

func (m Model) renderTreeOrganizationHeader(organization repositoryTreeOrganization) string {
	key := organizationFocusKey(organization.Organization)
	indicator := m.Symbols.Expanded
	if m.TreeOrganizationCollapsed[key] {
		indicator = m.Symbols.Collapsed
	}
	gutter := "  "
	if m.SelectedKey == key {
		gutter = m.Symbols.Selected + " "
	}
	prs := organization.PRs()
	header := gutter + " " + indicator + " " + organization.Organization + fmt.Sprintf(" %d", len(prs))
	if m.TreeOrganizationCollapsed[key] {
		risk := m.repositoryRisk(prs)
		if risk != "" {
			fill := "─"
			if m.Config.Display.ASCII {
				fill = "-"
			}
			fillWidth := m.availableWidth() - lipgloss.Width(header) - lipgloss.Width(risk) - 2
			if fillWidth > 0 {
				header += " " + strings.Repeat(fill, fillWidth) + " " + risk
			} else {
				header += " " + risk
			}
		}
	}
	if lipgloss.Width(header) > m.availableWidth() {
		header = m.truncateForMode(header, m.availableWidth())
	}
	if m.SelectedKey == key {
		return m.Styles.SelectedStyle.Render(header)
	}
	return m.Styles.HeaderStyle.Render(header)
}

func (m Model) renderRepositoryHeader(group model.RepositoryGroup) string {
	key := repositoryFocusKey(group.Organization, group.Repository)
	indicator := m.Symbols.Expanded
	if m.RepositoryCollapsed[key] {
		indicator = m.Symbols.Collapsed
	}
	gutter := "  "
	if m.SelectedKey == key {
		gutter = m.Symbols.Selected + " "
	}
	header := "  " + gutter + " " + indicator + " " + group.Repository + fmt.Sprintf(" %d", len(group.PRs))
	if m.RepositoryCollapsed[key] {
		risk := m.repositoryRisk(group.PRs)
		if risk != "" {
			fill := "─"
			if m.Config.Display.ASCII {
				fill = "-"
			}
			fillWidth := m.availableWidth() - lipgloss.Width(header) - lipgloss.Width(risk) - 2
			if fillWidth > 0 {
				header += " " + strings.Repeat(fill, fillWidth) + " " + risk
			} else {
				header += " " + risk
			}
		}
	}
	if lipgloss.Width(header) > m.availableWidth() {
		header = m.truncateForMode(header, m.availableWidth())
	}
	if m.SelectedKey == key {
		return m.Styles.SelectedStyle.Render(header)
	}
	return m.Styles.HeaderStyle.Render(header)
}

func (m Model) repositoryRisk(prs []model.PullRequest) string {
	failing, behind, threads := 0, 0, 0
	for _, pr := range prs {
		if pr.CheckStatus == model.CheckStatusFailing {
			failing++
		}
		if pr.MergeStatus == model.MergeStatusBehind {
			behind++
		}
		threads += pr.UnresolvedCount
		if pr.UnresolvedCount == 0 {
			n, _ := strconv.Atoi(strings.TrimSuffix(pr.UnresolvedThreads, "+"))
			threads += n
		}
	}
	var risk []string
	if failing > 0 {
		risk = append(risk, fmt.Sprintf("%s%d", m.Symbols.CIFailing, failing))
	}
	if behind > 0 {
		risk = append(risk, fmt.Sprintf("%s%d", m.Symbols.MergeBehind, behind))
	}
	if threads > 0 {
		risk = append(risk, fmt.Sprintf("%s%d", m.Symbols.Thread, threads))
	}
	return strings.Join(risk, " ")
}

type repositoryRowLayout struct {
	number, title, repository, author, age int
	showAuthor, showAge, showThreads       bool
}

func (m Model) repositoryLayoutFor(pr model.PullRequest) repositoryRowLayout {
	width := m.availableWidth()
	prs := m.visiblePRs()
	if len(prs) == 0 {
		prs = []model.PullRequest{pr}
	}
	layout := repositoryRowLayout{showAuthor: m.DisplayMode == model.DisplayModeFull && width >= 80, showAge: m.DisplayMode != model.DisplayModeMinimal && width >= 60, showThreads: m.DisplayMode != model.DisplayModeMinimal && width >= 60}
	repoCap := 20
	if width < 40 {
		repoCap = 8
	}
	for _, item := range prs {
		n := lipgloss.Width(fmt.Sprintf("#%d", item.Number))
		if n > layout.number {
			layout.number = n
		}
		r := lipgloss.Width(m.truncateForMode(item.Repository, repoCap))
		if r > layout.repository {
			layout.repository = r
		}
		if layout.showAuthor {
			a := lipgloss.Width(m.truncateForMode(item.Author, 12))
			if a > layout.author {
				layout.author = a
			}
		}
		if layout.showAge {
			a := lipgloss.Width(ageString(item))
			if a > layout.age {
				layout.age = a
			}
		}
	}
	if layout.age == 0 {
		layout.age = 2
	}
	fixed := 4 + 2 + 2 + layout.number + layout.repository + 5
	separators := 6
	if layout.showAuthor {
		fixed += layout.author
		separators++
	}
	if layout.showAge {
		fixed += layout.age
		separators++
	}
	if layout.showThreads {
		fixed += 4
		separators++
	}
	layout.title = max(1, width-fixed-separators)
	return layout
}

func (m Model) renderRepositoryPRRow(pr model.PullRequest, last bool) string {
	if m.availableWidth() < 24 {
		return m.truncateForMode("Terminal too narrow (minimum 24 columns)", m.availableWidth())
	}
	selected := pr.Key == m.SelectedKey
	_, changed := m.ChangedKeys[pr.Key]
	gutter := "  "
	if selected {
		gutter = m.Symbols.Selected + " "
	}
	if changed {
		gutter = " " + m.Symbols.Changed
	}
	if selected && changed {
		gutter = m.Symbols.Selected + m.Symbols.Changed
	}
	connector := "├─"
	if last {
		connector = "└─"
	}
	if m.Config.Display.ASCII {
		connector = "|-"
		if last {
			connector = "`-"
		}
	}
	layout := m.repositoryLayoutFor(pr)
	parts := []string{"    ", gutter, connector, padRight(fmt.Sprintf("#%d", pr.Number), layout.number), padRight(m.truncateForMode(pr.Title, layout.title), layout.title), padRight(m.truncateForMode(pr.Repository, layout.repository), layout.repository)}
	if layout.showAuthor {
		parts = append(parts, padRight(m.truncateForMode(pr.Author, layout.author), layout.author))
	}
	if layout.showAge {
		parts = append(parts, padLeft(ageString(pr), layout.age))
	}
	parts = append(parts, m.renderStatusTriad(pr))
	if layout.showThreads && pr.UnresolvedThreads != "" && pr.UnresolvedThreads != "0" {
		parts = append(parts, padLeft(m.threadLabel(pr.UnresolvedThreads), 4))
	}
	row := strings.Join(parts, " ")
	if lipgloss.Width(row) > m.availableWidth() {
		return m.truncateForMode("Terminal too narrow for repository row", m.availableWidth())
	}
	if selected {
		row = m.Styles.SelectedStyle.Render(row)
	} else if changed {
		row = m.Styles.ChangedStyle.Render(row)
	} else if pr.IsDraft {
		row = m.Styles.DraftStyle.Render(row)
	}
	return row
}

func (m Model) truncateForMode(value string, width int) string {
	if !m.Config.Display.ASCII {
		return truncateCells(value, width)
	}
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(value) <= width {
		return value
	}
	if width <= 3 {
		return strings.Repeat(".", width)
	}
	g := uniseg.NewGraphemes(value)
	result := ""
	for g.Next() {
		next := result + g.Str()
		if lipgloss.Width(next) > width-3 {
			break
		}
		result = next
	}
	return result + "..."
}

// renderOrgHeader renders an organization header and, when collapsed, a risk rollup.
func (m Model) renderOrgHeader(group model.PRGroup) string {
	indicator := m.Symbols.Expanded
	if group.Collapsed {
		indicator = m.Symbols.Collapsed
	}
	visible, failing, behind, threads := 0, 0, 0, 0
	for _, pr := range group.PRs {
		if !m.ShowDrafts && pr.IsDraft {
			continue
		}
		visible++
		if pr.CheckStatus == model.CheckStatusFailing {
			failing++
		}
		if pr.MergeStatus == model.MergeStatusBehind {
			behind++
		}
		threads += pr.UnresolvedCount
		if pr.UnresolvedCount == 0 {
			n, _ := strconv.Atoi(strings.TrimSuffix(pr.UnresolvedThreads, "+"))
			threads += n
		}
	}
	gutter := "  "
	selected := m.SelectedKey == organizationFocusKey(group.Organization)
	if selected {
		gutter = m.Symbols.Selected + " "
	}
	header := fmt.Sprintf("%s %s %s %d", gutter, indicator, group.Organization, visible)
	width := m.availableWidth()
	if group.Collapsed {
		var risk []string
		if failing > 0 {
			risk = append(risk, fmt.Sprintf("%s%d", m.Symbols.CIFailing, failing))
		}
		if behind > 0 {
			risk = append(risk, fmt.Sprintf("%s%d", m.Symbols.MergeBehind, behind))
		}
		if threads > 0 {
			risk = append(risk, fmt.Sprintf("%s%d", m.Symbols.Thread, threads))
		}
		if len(risk) > 0 {
			right := strings.Join(risk, " ")
			fill := "─"
			if m.Config.Display.ASCII {
				fill = "-"
			}
			fillWidth := width - lipgloss.Width(header) - lipgloss.Width(right) - 2
			if fillWidth > 0 {
				header += " " + strings.Repeat(fill, fillWidth) + " " + right
			} else {
				header += " " + right
			}
		}
	}
	if lipgloss.Width(header) > width {
		header = truncateCells(header, width)
	}
	if selected {
		return m.Styles.SelectedStyle.Render(header)
	}
	return m.Styles.HeaderStyle.Render(header)
}

type rowLayout struct {
	id, title, author, age           int
	showAuthor, showAge, showThreads bool
}

func (m Model) layoutFor(pr model.PullRequest) rowLayout {
	width := m.availableWidth()
	capID := 24
	if width < 40 {
		capID = 12
	}
	prs := m.visiblePRs()
	if len(prs) == 0 {
		prs = []model.PullRequest{pr}
	}
	layout := rowLayout{showAuthor: m.DisplayMode == model.DisplayModeFull && width >= 80, showAge: m.DisplayMode != model.DisplayModeMinimal && width >= 60, showThreads: m.DisplayMode != model.DisplayModeMinimal && width >= 60}
	for _, item := range prs {
		idw := lipgloss.Width(truncateIdentity(item.Repository, item.Number, capID))
		if idw > layout.id {
			layout.id = idw
		}
		if layout.showAuthor {
			w := lipgloss.Width(truncateCells(item.Author, 12))
			if w > layout.author {
				layout.author = w
			}
		}
		if layout.showAge {
			w := lipgloss.Width(ageString(item))
			if w > layout.age {
				layout.age = w
			}
		}
	}
	if layout.id > capID {
		layout.id = capID
	}
	if layout.age == 0 {
		layout.age = 2
	}
	fixed := 2 + layout.id + 5
	fields := 3 // gutter, id, title, triad => three separators
	if layout.showAuthor {
		fixed += layout.author
		fields++
	}
	if layout.showAge {
		fixed += layout.age
		fields++
	}
	if layout.showThreads {
		fixed += 4
		fields++
	}
	layout.title = max(1, width-fixed-fields)
	return layout
}

// renderPRRow renders a single, project-first, one-line PR row.
func (m Model) renderPRRow(pr model.PullRequest) string {
	width := m.availableWidth()
	if width < 24 {
		return truncateCells("Terminal too narrow (minimum 24 columns)", width)
	}
	selected := pr.Key == m.SelectedKey
	_, changed := m.ChangedKeys[pr.Key]
	gutter := "  "
	if selected {
		gutter = m.Symbols.Selected + " "
	}
	if changed {
		gutter = " " + m.Symbols.Changed
	}
	if selected && changed {
		gutter = m.Symbols.Selected + m.Symbols.Changed
	}
	layout := m.layoutFor(pr)
	rawID := truncateIdentity(pr.Repository, pr.Number, layout.id)
	if lipgloss.Width(rawID) > layout.id {
		return truncateCells("Terminal too narrow for PR number", width)
	}
	id := padRight(rawID, layout.id)
	title := padRight(truncateCells(pr.Title, layout.title), layout.title)
	parts := []string{gutter, id, title}
	if layout.showAuthor {
		parts = append(parts, padRight(truncateCells(pr.Author, layout.author), layout.author))
	}
	if layout.showAge {
		parts = append(parts, padLeft(ageString(pr), layout.age))
	}
	parts = append(parts, m.renderStatusTriad(pr))
	if layout.showThreads && pr.UnresolvedThreads != "" && pr.UnresolvedThreads != "0" {
		parts = append(parts, padLeft(m.threadLabel(pr.UnresolvedThreads), 4))
	}
	row := strings.Join(parts, " ")
	if selected {
		row = m.Styles.SelectedStyle.Render(row)
	} else if changed {
		row = m.Styles.ChangedStyle.Render(row)
	} else if pr.IsDraft {
		row = m.Styles.DraftStyle.Render(row)
	}
	return row
}

func (m Model) renderPRRowFull(pr model.PullRequest) string    { return m.renderPRRow(pr) }
func (m Model) renderPRRowCompact(pr model.PullRequest) string { return m.renderPRRow(pr) }
func (m Model) renderPRRowMinimal(pr model.PullRequest) string { return m.renderPRRow(pr) }

func (m Model) availableWidth() int {
	if m.Width > 0 {
		return m.Width
	}
	return 80
}

func padRight(s string, width int) string {
	return s + strings.Repeat(" ", max(0, width-lipgloss.Width(s)))
}
func padLeft(s string, width int) string {
	return strings.Repeat(" ", max(0, width-lipgloss.Width(s))) + s
}

func ageString(pr model.PullRequest) string {
	if pr.CreatedAt.IsZero() {
		return fmt.Sprintf("%dd", pr.DaysOpen)
	}
	d := time.Since(pr.CreatedAt)
	if d < 0 {
		d = 0
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 48*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	days := int(d.Hours() / 24)
	if days < 14 {
		return fmt.Sprintf("%dd", days)
	}
	if days < 60 {
		return fmt.Sprintf("%dw", days/7)
	}
	if days < 365 {
		return fmt.Sprintf("%dmo", days/30)
	}
	return fmt.Sprintf("%dy", days/365)
}

func truncateIdentity(repository string, number, max int) string {
	suffix := fmt.Sprintf("#%d", number)
	if repository == "" {
		return suffix
	}
	full := repository + suffix
	if lipgloss.Width(full) <= max {
		return full
	}
	return truncateCellsLeft(repository, max-lipgloss.Width(suffix)) + suffix
}

func truncateCellsLeft(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	g := uniseg.NewGraphemes(s)
	var clusters []string
	for g.Next() {
		clusters = append(clusters, g.Str())
	}
	result := ""
	for i := len(clusters) - 1; i >= 0; i-- {
		if lipgloss.Width(clusters[i]+result) > width-1 {
			break
		}
		result = clusters[i] + result
	}
	return "…" + result
}

func truncateCells(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}
	if width == 1 {
		return "…"
	}
	g := uniseg.NewGraphemes(s)
	result := ""
	for g.Next() {
		next := result + g.Str()
		if lipgloss.Width(next) > width-1 {
			break
		}
		result = next
	}
	return result + "…"
}

func (m Model) truncateTitle(title string, maxLen int) string { return truncateCells(title, maxLen) }

func (m Model) renderStatusTriad(pr model.PullRequest) string {
	ci, cis, _ := m.checkSymbol(pr.CheckStatus)
	review, rs, _ := m.reviewSymbol(pr.ReviewStatus)
	merge, ms, _ := m.mergeSymbol(pr)
	return strings.Join([]string{cis.Render(ci), rs.Render(review), ms.Render(merge)}, " ")
}

func (m Model) threadLabel(value string) string {
	n, err := strconv.Atoi(strings.TrimSuffix(value, "+"))
	if err == nil && n > 99 {
		value = "99+"
	}
	return m.Symbols.Thread + value
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

// renderSelectedDecoder explains the positional symbols for the selected PR.
func (m Model) renderSelectedDecoder() string { return m.renderSelectedDecoderLevel(0) }

func (m Model) renderSelectedDecoderLevel(level int) string {
	pr := m.SelectedPR()
	if pr == nil {
		return ""
	}
	ci, cis, ciWord := m.checkSymbol(pr.CheckStatus)
	review, rs, reviewWord := m.reviewSymbol(pr.ReviewStatus)
	merge, ms, mergeWord := m.mergeSymbol(*pr)
	status := ms.Render(merge) + mergeWord
	if level < 2 {
		status = strings.Join([]string{cis.Render(ci) + ciWord, rs.Render(review) + reviewWord, status}, " ")
	}
	parts := []string{fmt.Sprintf("%s#%d", pr.Repository, pr.Number), status}
	if pr.IsDraft && mergeWord != "draft" {
		parts = append(parts, "draft")
	}
	if pr.CanUpdateBranch() && !pr.IsDraft {
		parts = append(parts, "— press u to update")
	}
	if level == 0 && len(pr.Reviewers) > 0 {
		reviewers := make([]string, 0, min(2, len(pr.Reviewers)))
		for _, reviewer := range pr.Reviewers[:min(2, len(pr.Reviewers))] {
			if strings.HasPrefix(reviewer, "team:") {
				reviewers = append(reviewers, "/"+strings.TrimPrefix(reviewer, "team:"))
			} else {
				reviewers = append(reviewers, "@"+reviewer)
			}
		}
		label := strings.Join(reviewers, ", ")
		if len(pr.Reviewers) > 2 {
			label += fmt.Sprintf(" +%d", len(pr.Reviewers)-2)
		}
		parts = append(parts, label)
	}
	if level < 3 && pr.UnresolvedThreads != "" && pr.UnresolvedThreads != "0" {
		parts = append(parts, m.threadLabel(pr.UnresolvedThreads))
	}
	return strings.Join(parts, " · ")
}

func (m Model) effectiveModeLabel() string {
	label := m.DisplayMode.String()
	effective := label
	if m.availableWidth() < 40 {
		effective = "minimal"
	} else if m.availableWidth() < 80 && m.DisplayMode == model.DisplayModeFull {
		effective = "compact"
	}
	if effective == label {
		return label
	}
	arrow := "→"
	if m.Config.Display.ASCII {
		arrow = ">"
	}
	return label + arrow + effective
}

// renderStatusBar renders compact chrome and reserves the discoverability hint.
func (m Model) renderStatusBar() string {
	help := "?help"
	limit := m.availableWidth() - 2
	username := "@" + m.Config.General.Username
	watch := ""
	if m.WatchMode {
		watch = fmt.Sprintf("↻%ds", m.Config.General.RefreshInterval)
	}
	mode := m.effectiveModeLabel()
	grouping := "org"
	if m.GroupingMode == model.GroupingModeRepository {
		grouping = "repo"
	}
	clock := ""
	if !m.LastRefresh.IsZero() {
		clock = m.LastRefresh.Format("15:04")
	}
	rate := ""
	if m.RateLimit.Remaining > 0 && m.RateLimit.Remaining <= 100 {
		rate = fmt.Sprintf("⚠rl%d", m.RateLimit.Remaining)
	}
	joinLeft := func(parts ...string) string {
		var kept []string
		for _, p := range parts {
			if p != "" {
				kept = append(kept, p)
			}
		}
		return strings.Join(kept, " ")
	}
	left := joinLeft(username, watch, grouping, mode, clock, rate)
	for level := 0; level <= 3; level++ {
		decoder := m.renderSelectedDecoderLevel(level)
		if decoder == "" {
			break
		}
		candidate := left + " │ " + decoder + " │ " + help
		if lipgloss.Width(candidate) <= limit {
			return m.Styles.StatusBarStyle.Render(candidate)
		}
	}
	if baseline := left + " · " + help; lipgloss.Width(baseline) <= limit {
		return m.Styles.StatusBarStyle.Render(baseline)
	}
	for _, candidateLeft := range []string{joinLeft(username, watch, grouping, mode, rate), joinLeft(username, grouping, mode, rate), joinLeft(username, grouping, rate), joinLeft(username, rate), username} {
		candidate := candidateLeft + " · " + help
		if lipgloss.Width(candidate) <= limit {
			return m.Styles.StatusBarStyle.Render(candidate)
		}
	}
	userWidth := max(1, limit-lipgloss.Width(" · "+help))
	left = truncateCells(username, userWidth)
	return m.Styles.StatusBarStyle.Render(left + " · " + help)
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
	case ModalAccountPicker:
		return m.renderAccountPickerModal()
	default:
		return ""
	}
}

// renderHelpModal renders keybindings and the status language in an 80x24-friendly layout.
func (m Model) renderHelpModal() string {
	lines := []string{
		m.Styles.ModalTitleStyle.Render("Keys & Symbols"),
		"j/k/↑/↓ move   gg/G top/bottom   o/O collapse",
		"h/l tree       v view (organization/repository)",
		"u update       r refresh          Enter open/toggle",
		"mouse click PR open / node toggle",
		"s account      c mode   d drafts  w watch",
		"? help         q/Esc quit",
		"",
		"Symbols",
		fmt.Sprintf("CI      %s passing  %s failing  %s pending  %s none", m.Symbols.CIPassing, m.Symbols.CIFailing, m.Symbols.CIPending, m.Symbols.CINone),
		fmt.Sprintf("Review  %s approved %s changes  %s required %s none", m.Symbols.ReviewApproved, m.Symbols.ReviewChanges, m.Symbols.ReviewRequired, m.Symbols.ReviewNone),
		fmt.Sprintf("Merge   %s ready    %s behind   %s blocked", m.Symbols.MergeReady, m.Symbols.MergeBehind, m.Symbols.MergeBlocked),
		fmt.Sprintf("        %s conflicts %s unstable %s unknown %s draft", m.Symbols.MergeConflicts, m.Symbols.MergeUnstable, m.Symbols.MergeUnknown, m.Symbols.MergeDraft),
		fmt.Sprintf("Gutter  %s selected  %s changed", m.Symbols.Selected, m.Symbols.Changed),
		fmt.Sprintf("Other   %sn threads  age 45m/17h/3d", m.Symbols.Thread),
		"ASCII   CI + x ~ -  Review + ! ? -",
		"        Merge = v # X ~ ? o",
		"Draft   row dimmed; ≠ wins merge slot on conflict",
		"",
		m.Styles.DimStyle.Render("Enter, q, or Esc dismisses"),
	}
	return m.Styles.ModalStyle.Render(strings.Join(lines, "\n"))
}

// renderAccountPickerModal renders the account picker modal.
func (m Model) renderAccountPickerModal() string {
	var b strings.Builder

	b.WriteString(m.Styles.ModalTitleStyle.Render("Switch Account"))
	b.WriteString("\n\n")

	for i, account := range m.Accounts {
		label := fmt.Sprintf("  %d. %s", i+1, account.Login)
		if account.Active {
			label += " (active)"
		}
		b.WriteString(label)
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.Styles.DimStyle.Render("Press number to select, q or Esc to cancel"))

	return m.Styles.ModalStyle.Render(b.String())
}
