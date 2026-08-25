package tui

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jgordijn/pr-dashboard/internal/config"
	"github.com/jgordijn/pr-dashboard/internal/github"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

// Update implements tea.Model. It handles all messages and returns updated model and commands.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.MouseMsg:
		return m.handleMouseMsg(msg)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		return m, nil

	case PRsLoadedMsg:
		return m.handlePRsLoaded(msg)

	case PRsErrorMsg:
		if msg.Account != "" && msg.Account != m.Config.General.Username {
			return m, nil
		}
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

	case AccountsLoadedMsg:
		return m.handleAccountsLoaded(msg)

	case AccountSwitchedMsg:
		return m.handleAccountSwitched(msg)

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
	if m.ViewMode == ViewDashboard && m.FlashMessage != "" && msg.String() != "H" && msg.String() != "z" && msg.String() != "M" {
		m.FlashMessage = ""
	}
	// If modal is showing, it captures keys over both dashboard and manager views.
	if m.Modal.Type != ModalNone {
		// Account picker: handle number key to select an account
		if m.Modal.Type == ModalAccountPicker {
			if key.Matches(msg, m.Keys.Quit) || key.Matches(msg, m.Keys.OpenBrowser) {
				m.Modal = ModalState{Type: ModalNone}
				return m, nil
			}
			return m.handleAccountPickerKey(msg)
		}
		if key.Matches(msg, m.Keys.Quit) || key.Matches(msg, m.Keys.OpenBrowser) {
			m.Modal = ModalState{Type: ModalNone}
			return m, nil
		}
		return m, nil
	}
	if m.ViewMode == ViewHiddenItems {
		return m.handleHiddenManagerKey(msg)
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

	case key.Matches(msg, m.Keys.Left):
		return m.treeLeft()

	case key.Matches(msg, m.Keys.Right):
		return m.treeRight()

	case key.Matches(msg, m.Keys.ToggleOrg):
		return m.toggleCurrentOrg()

	case key.Matches(msg, m.Keys.ToggleAllOrgs):
		return m.toggleAllOrgs()

	case key.Matches(msg, m.Keys.ToggleDrafts):
		return m.toggleDrafts()

	case key.Matches(msg, m.Keys.CycleDisplayMode):
		return m.cycleDisplayMode()

	case key.Matches(msg, m.Keys.ToggleGrouping):
		return m.toggleGrouping()

	case key.Matches(msg, m.Keys.CycleSort):
		m = m.cycleSortField()
		return m, nil

	case key.Matches(msg, m.Keys.ToggleSort):
		m = m.toggleSortDirection()
		return m, nil

	case key.Matches(msg, m.Keys.HideItem):
		return m.hideFocusedItem()

	case key.Matches(msg, m.Keys.UndoHide):
		return m.undoLastHide()

	case key.Matches(msg, m.Keys.ManageHidden):
		return m.openHiddenManager()

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
		if m.GroupingMode == model.GroupingModeOrganization {
			if _, ok := parseOrganizationFocusKey(m.SelectedKey); ok {
				m = m.toggleOrganizationViewNode(m.SelectedKey)
				return m, nil
			}
		}
		if m.GroupingMode == model.GroupingModeRepository {
			if _, ok := parseOrganizationFocusKey(m.SelectedKey); ok {
				m = m.toggleTreeOrganization(m.SelectedKey)
				return m, nil
			}
			if _, _, ok := parseRepositoryFocusKey(m.SelectedKey); ok {
				m = m.toggleRepository(m.SelectedKey)
				return m, nil
			}
		}
		return m.openInBrowser()

	case key.Matches(msg, m.Keys.SwitchAccount):
		if m.ActionInProgress {
			return m, nil
		}
		return m, m.listAccountsCmd()
	}

	return m, nil
}

// handlePRsLoaded processes successfully loaded PR data.
func (m Model) handlePRsLoaded(msg PRsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Account != "" && msg.Account != m.Config.General.Username {
		return m, nil
	}
	// Detect changes for highlighting
	oldPRs := m.getAllPRs()
	newPRs := getAllPRsFromGroups(msg.Groups)
	changes := model.DetectChanges(oldPRs, newPRs)

	var cmds []tea.Cmd
	for _, change := range changes {
		m.ChangedKeys[change.Key] = time.Now()
		cmds = append(cmds, clearHighlightCmd(change.Key))
	}

	// Capture legacy/in-memory collapse flags, then restore them onto fresh data.
	if m.OrganizationCollapsed == nil {
		m.OrganizationCollapsed = make(map[string]bool)
	}
	for _, group := range m.Groups {
		if group.Collapsed {
			m.OrganizationCollapsed[organizationFocusKey(group.Organization)] = true
		}
	}
	for i := range msg.Groups {
		msg.Groups[i].Collapsed = m.OrganizationCollapsed[organizationFocusKey(msg.Groups[i].Organization)]
	}

	// Update groups
	m.Groups = msg.Groups
	m.TotalCount = countPRsInGroups(msg.Groups)
	m.RateLimit = msg.RateLimit
	m.LastRefresh = time.Now()
	m.IsLoading = false
	m.Error = nil
	m.FlashMessage = ""

	// Preserve selection by stable key
	m.SelectedKey = m.findNearestVisibleKey()

	// Start watch tick if enabled
	if m.WatchMode {
		cmds = append(cmds, m.watchTickCmd())
	}
	m = m.persistViewState()
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
	keys := m.visibleItemKeys()
	if len(keys) == 0 {
		return m, nil
	}
	currentIdx := -1
	for i, key := range keys {
		if key == m.SelectedKey {
			currentIdx = i
			break
		}
	}
	if currentIdx > 0 {
		m.SelectedKey = keys[currentIdx-1]
	}
	return m.persistViewState(), nil
}

// moveDown moves selection to next visible PR.
func (m Model) moveDown() (tea.Model, tea.Cmd) {
	keys := m.visibleItemKeys()
	if len(keys) == 0 {
		return m, nil
	}
	currentIdx := -1
	for i, key := range keys {
		if key == m.SelectedKey {
			currentIdx = i
			break
		}
	}
	if currentIdx < len(keys)-1 {
		m.SelectedKey = keys[currentIdx+1]
	}
	return m.persistViewState(), nil
}

// goToTop moves selection to first visible PR.
func (m Model) goToTop() (tea.Model, tea.Cmd) {
	keys := m.visibleItemKeys()
	if len(keys) > 0 {
		m.SelectedKey = keys[0]
	}
	return m.persistViewState(), nil
}

// goToBottom moves selection to last visible PR.
func (m Model) goToBottom() (tea.Model, tea.Cmd) {
	keys := m.visibleItemKeys()
	if len(keys) > 0 {
		m.SelectedKey = keys[len(keys)-1]
	}
	return m.persistViewState(), nil
}

// toggleCurrentOrg toggles collapse state of the org containing selected PR.
func (m Model) toggleCurrentOrg() (tea.Model, tea.Cmd) {
	if m.SelectedKey == "" {
		return m, nil
	}
	if m.GroupingMode == model.GroupingModeRepository {
		m = m.toggleFocusedTreeNode()
		return m.persistViewState(), nil
	}
	if _, ok := parseOrganizationFocusKey(m.SelectedKey); ok {
		m = m.toggleOrganizationViewNode(m.SelectedKey)
		return m.persistViewState(), nil
	}

	pr := m.SelectedPR()
	if pr == nil {
		return m, nil
	}

	for i := range m.Groups {
		if m.Groups[i].Organization == pr.Organization {
			m.Groups[i].Collapsed = !m.Groups[i].Collapsed
			if m.OrganizationCollapsed == nil {
				m.OrganizationCollapsed = make(map[string]bool)
			}
			m.OrganizationCollapsed[organizationFocusKey(pr.Organization)] = m.Groups[i].Collapsed
			break
		}
	}

	// Adjust selection if now in collapsed group
	m.SelectedKey = m.findNearestVisibleKey()
	return m.persistViewState(), nil
}

// toggleAllOrgs toggles collapse state of all organizations.
func (m Model) toggleAllOrgs() (tea.Model, tea.Cmd) {
	if m.GroupingMode == model.GroupingModeRepository {
		m.ensureTreeCollapseMaps()
		organizations := m.visibleTreeOrganizations()
		anyCollapsed := false
		for _, organization := range organizations {
			if m.TreeOrganizationCollapsed[organizationFocusKey(organization.Organization)] {
				anyCollapsed = true
				break
			}
			for _, repository := range organization.Repositories {
				if m.RepositoryCollapsed[repositoryFocusKey(repository.Organization, repository.Repository)] {
					anyCollapsed = true
					break
				}
			}
			if anyCollapsed {
				break
			}
		}
		for _, organization := range organizations {
			m.TreeOrganizationCollapsed[organizationFocusKey(organization.Organization)] = !anyCollapsed
			for _, repository := range organization.Repositories {
				m.RepositoryCollapsed[repositoryFocusKey(repository.Organization, repository.Repository)] = !anyCollapsed
			}
		}
		m.SelectedKey = m.findNearestVisibleKey()
		return m.persistViewState(), nil
	}
	// Determine if we should expand or collapse all
	// If any is collapsed, expand all; otherwise collapse all
	anyCollapsed := false
	for _, group := range m.Groups {
		if group.Collapsed {
			anyCollapsed = true
			break
		}
	}

	if m.OrganizationCollapsed == nil {
		m.OrganizationCollapsed = make(map[string]bool)
	}
	for i := range m.Groups {
		m.Groups[i].Collapsed = !anyCollapsed
		m.OrganizationCollapsed[organizationFocusKey(m.Groups[i].Organization)] = !anyCollapsed
	}

	m.SelectedKey = m.findNearestVisibleKey()
	return m.persistViewState(), nil
}

// toggleDrafts toggles draft visibility.
func (m Model) toggleDrafts() (tea.Model, tea.Cmd) {
	m.ShowDrafts = !m.ShowDrafts
	m.SelectedKey = m.findNearestVisibleKey()
	return m.persistViewState(), nil
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
	return m.persistViewState(), nil
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
		opener := m.OpenPR
		if opener == nil {
			return ActionResultMsg{PRKey: pr.Key, Action: "open", Success: false, Message: "Failed to open in browser: browser opener unavailable", Err: errors.New("browser opener unavailable")}
		}
		err := opener(pr.Organization, pr.Repository, pr.Number)
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

// listAccountsCmd returns a command that fetches the list of gh CLI accounts.
func (m Model) listAccountsCmd() tea.Cmd {
	return func() tea.Msg {
		accounts, err := config.ListGHAccounts()
		return AccountsLoadedMsg{Accounts: accounts, Err: err}
	}
}

// handleAccountsLoaded processes the list of fetched accounts.
func (m Model) handleAccountsLoaded(msg AccountsLoadedMsg) (tea.Model, tea.Cmd) {
	if msg.Err != nil {
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Account Error",
			Message: msg.Err.Error(),
		}
		return m, nil
	}

	// Override Active flags to reflect the dashboard's current account,
	// not the gh CLI's active account (they can differ since we use explicit tokens).
	for i := range msg.Accounts {
		msg.Accounts[i].Active = msg.Accounts[i].Login == m.Config.General.Username
	}
	m.Accounts = msg.Accounts

	if len(msg.Accounts) <= 1 {
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Switch Account",
			Message: "Only one account available",
		}
		return m, nil
	}

	m.Modal = ModalState{
		Type:  ModalAccountPicker,
		Title: "Switch Account",
	}
	return m, nil
}

// handleAccountPickerKey handles key presses while the account picker modal is open.
func (m Model) handleAccountPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Parse number key (1-based index)
	idx, err := strconv.Atoi(msg.String())
	if err != nil || idx < 1 || idx > len(m.Accounts) {
		return m, nil
	}

	selected := m.Accounts[idx-1]

	// If already active, just dismiss
	if selected.Active {
		m.Modal = ModalState{Type: ModalNone}
		return m, nil
	}

	m.Modal = ModalState{Type: ModalNone}
	m.IsLoading = true
	return m, m.switchAccountCmd(selected.Login)
}

// switchAccountCmd returns a command that switches the active gh account
// and fetches the new account's auth token.
func (m Model) switchAccountCmd(login string) tea.Cmd {
	return func() tea.Msg {
		if err := config.SwitchGHAccount(login); err != nil {
			return AccountSwitchedMsg{Login: login, Err: err}
		}

		// Fetch the token explicitly — go-gh caches auth config in-process,
		// so NewClient() would still use the old token.
		token, err := config.GHAuthToken(login)
		if err != nil {
			return AccountSwitchedMsg{Login: login, Err: err}
		}

		return AccountSwitchedMsg{Login: login, Token: token}
	}
}

// handleAccountSwitched processes an account switch result.
func (m Model) handleAccountSwitched(msg AccountSwitchedMsg) (tea.Model, tea.Cmd) {
	m.IsLoading = false

	if msg.Err != nil {
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Switch Failed",
			Message: msg.Err.Error(),
		}
		return m, nil
	}

	// Create a new client with the explicit token — go-gh caches auth config
	// in-process, so NewClient() would still use the old account's token.
	newClient, err := github.NewClientWithToken(msg.Token)
	if err != nil {
		m.Modal = ModalState{
			Type:    ModalError,
			Title:   "Client Error",
			Message: "Switched account but failed to create client: " + err.Error(),
		}
		return m, nil
	}

	m.Client = newClient
	m.Config.General.Username = msg.Login
	m.Groups = nil
	m.TotalCount = 0
	m.ChangedKeys = make(map[string]time.Time)
	m.restoreAccountView(msg.Login)
	m.ViewMode = ViewDashboard
	m.HiddenManager = HiddenManagerState{}
	m.LastHidden = nil
	m.FlashMessage = ""
	m.IsLoading = true

	return m, tea.Batch(m.fetchPRsCmd(), m.Spinner.Tick)
}
