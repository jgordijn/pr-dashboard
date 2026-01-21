package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jgordijn/pr-dashboard/internal/github"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

// Update implements tea.Model. It handles all messages and returns updated model and commands.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case PRsLoadedMsg:
		return m.handlePRsLoaded(msg)

	case PRsErrorMsg:
		m.IsLoading = false
		m.Error = msg.Err
		return m, nil

	case ActionStartMsg:
		m.ActionInProgress = true
		m.IsLoading = true
		return m, nil

	case ActionResultMsg:
		return m.handleActionResult(msg)

	case RefreshTickMsg:
		if m.WatchMode && !m.ActionInProgress {
			m.IsLoading = true
			return m, tea.Batch(m.fetchPRsCmd(), m.Spinner.Tick)
		}
		if m.WatchMode && m.ActionInProgress {
			m.RefreshQueued = true
		}
		return m, nil

	case ClearHighlightMsg:
		delete(m.ChangedKeys, msg.Key)
		return m, nil

	case spinner.TickMsg:
		// Update spinner animation while loading
		if m.IsLoading {
			var cmd tea.Cmd
			m.Spinner, cmd = m.Spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	return m, nil
}

// handleKeyMsg processes keyboard input.
func (m Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If modal is showing, only handle dismiss keys
	if m.Modal.Type != ModalNone {
		if key.Matches(msg, m.Keys.Quit) || key.Matches(msg, m.Keys.OpenBrowser) {
			m.Modal = ModalState{Type: ModalNone}
			return m, nil
		}
		return m, nil
	}

	// Handle gg (go to top) - vim style double-g
	if msg.String() == "g" {
		if m.gPressed {
			// gg pressed - go to top
			m.gPressed = false
			return m.goToTop()
		}
		m.gPressed = true
		return m, nil
	}
	m.gPressed = false

	switch {
	case key.Matches(msg, m.Keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, m.Keys.Help):
		m.Modal = ModalState{Type: ModalHelp, Title: "Help", Message: ""}
		return m, nil

	case key.Matches(msg, m.Keys.Up):
		return m.moveUp()

	case key.Matches(msg, m.Keys.Down):
		return m.moveDown()

	case key.Matches(msg, m.Keys.Bottom):
		return m.goToBottom()

	case key.Matches(msg, m.Keys.ToggleOrg):
		return m.toggleCurrentOrg()

	case key.Matches(msg, m.Keys.ToggleAllOrgs):
		return m.toggleAllOrgs()

	case key.Matches(msg, m.Keys.ToggleDrafts):
		return m.toggleDrafts()

	case key.Matches(msg, m.Keys.CycleDisplayMode):
		return m.cycleDisplayMode()

	case key.Matches(msg, m.Keys.ToggleWatch):
		return m.toggleWatch()

	case key.Matches(msg, m.Keys.Refresh):
		if m.ActionInProgress {
			m.RefreshQueued = true
			return m, nil
		}
		m.IsLoading = true
		return m, tea.Batch(m.fetchPRsCmd(), m.Spinner.Tick)

	case key.Matches(msg, m.Keys.UpdateBranch):
		return m.handleUpdateBranch()

	case key.Matches(msg, m.Keys.OpenBrowser):
		return m.openInBrowser()
	}

	return m, nil
}

// handlePRsLoaded processes successfully loaded PR data.
func (m Model) handlePRsLoaded(msg PRsLoadedMsg) (tea.Model, tea.Cmd) {
	// Detect changes for highlighting
	oldPRs := m.getAllPRs()
	newPRs := getAllPRsFromGroups(msg.Groups)
	changes := model.DetectChanges(oldPRs, newPRs)

	var cmds []tea.Cmd
	for _, change := range changes {
		m.ChangedKeys[change.Key] = time.Now()
		cmds = append(cmds, clearHighlightCmd(change.Key))
	}

	// Update groups
	m.Groups = msg.Groups
	m.TotalCount = countPRsInGroups(msg.Groups)
	m.RateLimit = msg.RateLimit
	m.LastRefresh = time.Now()
	m.IsLoading = false
	m.Error = nil

	// Preserve selection by stable key
	m.SelectedKey = m.findNearestVisibleKey()

	// Start watch tick if enabled
	if m.WatchMode {
		cmds = append(cmds, m.watchTickCmd())
	}

	return m, tea.Batch(cmds...)
}

// handleActionResult processes action completion.
func (m Model) handleActionResult(msg ActionResultMsg) (tea.Model, tea.Cmd) {
	m.ActionInProgress = false
	m.IsLoading = false

	if msg.Success {
		m.Modal = ModalState{
			Type:    ModalSuccess,
			Title:   "Success",
			Message: msg.Message,
		}
	} else {
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Error",
			Message: msg.Message,
		}
	}

	var cmds []tea.Cmd

	// Execute queued refresh
	if m.RefreshQueued {
		m.RefreshQueued = false
		m.IsLoading = true
		cmds = append(cmds, m.fetchPRsCmd(), m.Spinner.Tick)
	}

	// Restart watch mode
	if m.WatchMode {
		cmds = append(cmds, m.watchTickCmd())
	}

	return m, tea.Batch(cmds...)
}

// moveUp moves selection to previous visible PR.
func (m Model) moveUp() (tea.Model, tea.Cmd) {
	visible := m.visiblePRs()
	if len(visible) == 0 {
		return m, nil
	}

	currentIdx := -1
	for i, pr := range visible {
		if pr.Key == m.SelectedKey {
			currentIdx = i
			break
		}
	}

	if currentIdx > 0 {
		m.SelectedKey = visible[currentIdx-1].Key
	}
	return m, nil
}

// moveDown moves selection to next visible PR.
func (m Model) moveDown() (tea.Model, tea.Cmd) {
	visible := m.visiblePRs()
	if len(visible) == 0 {
		return m, nil
	}

	currentIdx := -1
	for i, pr := range visible {
		if pr.Key == m.SelectedKey {
			currentIdx = i
			break
		}
	}

	if currentIdx < len(visible)-1 {
		m.SelectedKey = visible[currentIdx+1].Key
	}
	return m, nil
}

// goToTop moves selection to first visible PR.
func (m Model) goToTop() (tea.Model, tea.Cmd) {
	visible := m.visiblePRs()
	if len(visible) > 0 {
		m.SelectedKey = visible[0].Key
	}
	return m, nil
}

// goToBottom moves selection to last visible PR.
func (m Model) goToBottom() (tea.Model, tea.Cmd) {
	visible := m.visiblePRs()
	if len(visible) > 0 {
		m.SelectedKey = visible[len(visible)-1].Key
	}
	return m, nil
}

// toggleCurrentOrg toggles collapse state of the org containing selected PR.
func (m Model) toggleCurrentOrg() (tea.Model, tea.Cmd) {
	if m.SelectedKey == "" {
		return m, nil
	}

	pr := m.SelectedPR()
	if pr == nil {
		return m, nil
	}

	for i := range m.Groups {
		if m.Groups[i].Organization == pr.Organization {
			m.Groups[i].Collapsed = !m.Groups[i].Collapsed
			break
		}
	}

	// Adjust selection if now in collapsed group
	m.SelectedKey = m.findNearestVisibleKey()
	return m, nil
}

// toggleAllOrgs toggles collapse state of all organizations.
func (m Model) toggleAllOrgs() (tea.Model, tea.Cmd) {
	// Determine if we should expand or collapse all
	// If any is collapsed, expand all; otherwise collapse all
	anyCollapsed := false
	for _, group := range m.Groups {
		if group.Collapsed {
			anyCollapsed = true
			break
		}
	}

	for i := range m.Groups {
		m.Groups[i].Collapsed = !anyCollapsed
	}

	m.SelectedKey = m.findNearestVisibleKey()
	return m, nil
}

// toggleDrafts toggles draft visibility.
func (m Model) toggleDrafts() (tea.Model, tea.Cmd) {
	m.ShowDrafts = !m.ShowDrafts
	m.SelectedKey = m.findNearestVisibleKey()
	return m, nil
}

// cycleDisplayMode cycles through display modes: full -> compact -> minimal -> full.
func (m Model) cycleDisplayMode() (tea.Model, tea.Cmd) {
	switch m.DisplayMode {
	case model.DisplayModeFull:
		m.DisplayMode = model.DisplayModeCompact
	case model.DisplayModeCompact:
		m.DisplayMode = model.DisplayModeMinimal
	case model.DisplayModeMinimal:
		m.DisplayMode = model.DisplayModeFull
	}
	return m, nil
}

// toggleWatch toggles watch mode.
func (m Model) toggleWatch() (tea.Model, tea.Cmd) {
	m.WatchMode = !m.WatchMode
	if m.WatchMode {
		return m, m.watchTickCmd()
	}
	return m, nil
}

// handleUpdateBranch initiates branch update for selected PR.
func (m Model) handleUpdateBranch() (tea.Model, tea.Cmd) {
	if m.ActionInProgress {
		return m, nil
	}

	pr := m.SelectedPR()
	if pr == nil {
		return m, nil
	}

	// Check if PR is a draft - draft PRs cannot be updated
	if pr.IsDraft {
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Cannot Update",
			Message: "Cannot update: PR is a draft",
		}
		return m, nil
	}

	// Validate preconditions per branch-actions/spec.md
	switch pr.MergeStatus {
	case model.MergeStatusConflicts:
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Cannot Update",
			Message: "Cannot update: PR has merge conflicts",
		}
		return m, nil
	case model.MergeStatusUnknown:
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Cannot Update",
			Message: "Cannot update: merge status unknown (GitHub is still computing)",
		}
		return m, nil
	case model.MergeStatusReady:
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Already Current",
			Message: "Branch is already up to date",
		}
		return m, nil
	case model.MergeStatusBlocked:
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Cannot Update",
			Message: "Cannot update: merge is blocked",
		}
		return m, nil
	case model.MergeStatusBehind:
		// This is the only valid state for update
	default:
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Cannot Update",
			Message: "Branch cannot be updated in current state",
		}
		return m, nil
	}

	m.ActionInProgress = true
	m.IsLoading = true

	return m, tea.Batch(m.updateBranchCmd(pr), m.Spinner.Tick)
}

// updateBranchCmd returns a command to update the branch.
func (m Model) updateBranchCmd(pr *model.PullRequest) tea.Cmd {
	return func() tea.Msg {
		err := github.UpdateBranch(context.Background(), pr.Organization, pr.Repository, pr.Number)
		if err != nil {
			return ActionResultMsg{
				PRKey:   pr.Key,
				Action:  "update",
				Success: false,
				Message: err.Error(),
				Err:     err,
			}
		}
		return ActionResultMsg{
			PRKey:   pr.Key,
			Action:  "update",
			Success: true,
			Message: "Branch updated successfully",
		}
	}
}

// openInBrowser opens the selected PR in the default browser.
func (m Model) openInBrowser() (tea.Model, tea.Cmd) {
	pr := m.SelectedPR()
	if pr == nil {
		return m, nil
	}

	return m, m.openBrowserCmd(pr)
}

// openBrowserCmd returns a command to open PR in browser via gh CLI.
func (m Model) openBrowserCmd(pr *model.PullRequest) tea.Cmd {
	return func() tea.Msg {
		// gh pr view --web <number> --repo <owner/name>
		err := github.OpenPRInBrowser(pr.Organization, pr.Repository, pr.Number)
		if err != nil {
			return ActionResultMsg{
				PRKey:   pr.Key,
				Action:  "open",
				Success: false,
				Message: "Failed to open in browser: " + err.Error(),
				Err:     err,
			}
		}
		return nil
	}
}

// getAllPRs returns all PRs from current groups.
func (m *Model) getAllPRs() []model.PullRequest {
	return getAllPRsFromGroups(m.Groups)
}

// getAllPRsFromGroups extracts all PRs from groups.
func getAllPRsFromGroups(groups []model.PRGroup) []model.PullRequest {
	var prs []model.PullRequest
	for _, g := range groups {
		prs = append(prs, g.PRs...)
	}
	return prs
}

// countPRsInGroups counts total PRs across all groups.
func countPRsInGroups(groups []model.PRGroup) int {
	count := 0
	for _, g := range groups {
		count += len(g.PRs)
	}
	return count
}
