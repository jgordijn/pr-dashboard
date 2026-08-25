package tui

import (
	"time"

	"github.com/jgordijn/pr-dashboard/internal/config"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

// PRsLoadedMsg is sent when PR data has been fetched successfully.
type PRsLoadedMsg struct {
	Account   string
	Groups    []model.PRGroup
	RateLimit RateLimitInfo
}

// PRsErrorMsg is sent when an error occurs fetching PRs.
type PRsErrorMsg struct {
	Account string
	Err     error
}

// ActionStartMsg is sent when an action (like update branch) starts.
type ActionStartMsg struct {
	PRKey  string
	Action string
}

// ActionResultMsg is sent when an action completes.
type ActionResultMsg struct {
	PRKey   string
	Action  string
	Success bool
	Message string
	Err     error
}

// RefreshTickMsg is sent by the watch mode timer.
type RefreshTickMsg struct {
	Time time.Time
}

// ClearHighlightMsg is sent to clear change highlighting for a specific PR.
type ClearHighlightMsg struct {
	Key string
}

// RateLimitInfo contains rate limit information from GitHub API.
type RateLimitInfo struct {
	Remaining int
	ResetAt   time.Time
}

// ModalType identifies the type of modal being displayed.
type ModalType int

const (
	// ModalNone indicates no modal is shown.
	ModalNone ModalType = iota
	// ModalHelp indicates the help modal is shown.
	ModalHelp
	// ModalSuccess indicates a success modal is shown.
	ModalSuccess
	// ModalError indicates an error modal is shown.
	ModalError
	// ModalAccountPicker indicates the account picker modal is shown.
	ModalAccountPicker
)

// ModalState represents the current modal state.
type ModalState struct {
	Type    ModalType
	Title   string
	Message string
}

// AccountsLoadedMsg is sent when the list of gh CLI accounts has been fetched.
type AccountsLoadedMsg struct {
	Accounts []config.GHAccount
	Err      error
}

// AccountSwitchedMsg is sent when an account switch operation completes.
type AccountSwitchedMsg struct {
	Login string // The account that was switched to
	Token string // The auth token for the new account
	Err   error
}
