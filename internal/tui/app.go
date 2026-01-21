package tui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/jgordijn/pr-dashboard/internal/config"
	"github.com/jgordijn/pr-dashboard/internal/github"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

// Model is the main application model for the TUI.
// It implements tea.Model interface for Bubble Tea.
type Model struct {
	// Configuration
	Config *config.Config
	// GitHub client for API calls
	Client *github.Client
	// Key bindings
	Keys *KeyMap
	// Styles
	Styles *Styles

	// PR Data
	Groups     []model.PRGroup
	TotalCount int

	// Selection - track by stable key per spec
	SelectedKey string
	// selectedIdx is derived from SelectedKey for rendering
	selectedIdx int

	// Display settings
	DisplayMode model.DisplayMode
	ShowDrafts  bool
	WatchMode   bool

	// Modal state
	Modal ModalState

	// Status
	LastRefresh time.Time
	RateLimit   RateLimitInfo
	IsLoading   bool
	Error       error

	// Action state per branch-actions/spec.md
	ActionInProgress bool
	RefreshQueued    bool

	// Change tracking per watch-mode/spec.md
	ChangedKeys map[string]time.Time

	// Terminal dimensions
	Width  int
	Height int

	// Internal state for gg detection
	gPressed bool

	// Loading spinner
	Spinner spinner.Model
}

// NewModel creates a new TUI model with the given configuration and client.
func NewModel(cfg *config.Config, client *github.Client) Model {
	displayMode := model.DisplayModeFull
	switch cfg.Display.InitialMode {
	case "compact":
		displayMode = model.DisplayModeCompact
	case "minimal":
		displayMode = model.DisplayModeMinimal
	}

	// Initialize spinner with dots style
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = NewStyles().SpinnerStyle

	return Model{
		Config:      cfg,
		Client:      client,
		Keys:        NewKeyMap(),
		Styles:      NewStyles(),
		DisplayMode: displayMode,
		ShowDrafts:  cfg.Display.ShowDrafts,
		WatchMode:   false,
		Modal:       ModalState{Type: ModalNone},
		ChangedKeys: make(map[string]time.Time),
		IsLoading:   true, // Start in loading state
		Spinner:     s,
	}
}

// Init implements tea.Model. It returns the initial command to fetch PRs.
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.fetchPRsCmd(),
		tea.WindowSize(),
		m.Spinner.Tick, // Start spinner animation
	)
}

// fetchPRsCmd returns a command that fetches PRs from GitHub.
func (m Model) fetchPRsCmd() tea.Cmd {
	return func() tea.Msg {
		if m.Client == nil {
			return PRsErrorMsg{Err: github.ErrEmptyUsername}
		}

		// Collect organization names
		var orgNames []string
		for _, org := range m.Config.Organizations {
			orgNames = append(orgNames, org.Login)
		}

		// Fetch PRs with context
		result, err := m.Client.FetchPRs(context.Background(), m.Config.General.Username, orgNames)
		if err != nil {
			return PRsErrorMsg{Err: err}
		}

		prs := model.TransformPRs(result.PullRequests)
		groups := model.GroupByOrganization(prs)

		// Extract rate limit info
		rateLimitInfo := RateLimitInfo{
			Remaining: 5000,
			ResetAt:   time.Now().Add(time.Hour),
		}
		if result.RateLimit != nil {
			rateLimitInfo.Remaining = result.RateLimit.Remaining
			rateLimitInfo.ResetAt = result.RateLimit.ResetAt
		}

		return PRsLoadedMsg{
			Groups:    groups,
			RateLimit: rateLimitInfo,
		}
	}
}

// watchTickCmd returns a command that sends a tick after the refresh interval.
func (m Model) watchTickCmd() tea.Cmd {
	interval := time.Duration(m.Config.General.RefreshInterval) * time.Second
	return tea.Tick(interval, func(t time.Time) tea.Msg {
		return RefreshTickMsg{Time: t}
	})
}

// clearHighlightCmd returns a command to clear highlight after 2 seconds.
func clearHighlightCmd(key string) tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return ClearHighlightMsg{Key: key}
	})
}

// SelectedPR returns the currently selected PR, or nil if none selected.
func (m *Model) SelectedPR() *model.PullRequest {
	if m.SelectedKey == "" {
		return nil
	}
	for i := range m.Groups {
		for j := range m.Groups[i].PRs {
			if m.Groups[i].PRs[j].Key == m.SelectedKey {
				return &m.Groups[i].PRs[j]
			}
		}
	}
	return nil
}

// visiblePRs returns all visible PRs considering collapsed orgs and draft filter.
func (m *Model) visiblePRs() []model.PullRequest {
	var visible []model.PullRequest
	for _, group := range m.Groups {
		if group.Collapsed {
			continue
		}
		for _, pr := range group.PRs {
			if !m.ShowDrafts && pr.IsDraft {
				continue
			}
			visible = append(visible, pr)
		}
	}
	return visible
}

// findNearestVisibleKey finds the nearest visible PR key after selection changes.
func (m *Model) findNearestVisibleKey() string {
	visible := m.visiblePRs()
	if len(visible) == 0 {
		return ""
	}
	// If current selection is still visible, keep it
	for _, pr := range visible {
		if pr.Key == m.SelectedKey {
			return m.SelectedKey
		}
	}
	// Otherwise, select first visible
	return visible[0].Key
}

// countVisiblePRs returns the total count of visible PRs.
func (m *Model) countVisiblePRs() int {
	return len(m.visiblePRs())
}
