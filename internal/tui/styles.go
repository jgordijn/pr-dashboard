// Package tui provides the terminal user interface for the PR dashboard.
package tui

import "github.com/charmbracelet/lipgloss"

// Color constants per pr-display/spec.md
const (
	// ColorGreen for PASSING, APPROVED, READY, CLEAN states
	ColorGreen = lipgloss.Color("#10B981")
	// ColorYellow for PENDING, BEHIND, BLOCKED, UNSTABLE, HAS_HOOKS states
	ColorYellow = lipgloss.Color("#F59E0B")
	// ColorRed for FAILING, CHANGES_REQUESTED, CONFLICTS, DIRTY states
	ColorRed = lipgloss.Color("#EF4444")
	// ColorGray for NONE, UNKNOWN, DRAFT states
	ColorGray = lipgloss.Color("#6B7280")
	// ColorHighlight for selection and changes
	ColorHighlight = lipgloss.Color("#3B82F6")
	// ColorWhite for normal text
	ColorWhite = lipgloss.Color("#FFFFFF")
	// ColorDim for dimmed/secondary text
	ColorDim = lipgloss.Color("#9CA3AF")
)

// Styles contains all lipgloss styles for the TUI.
type Styles struct {
	// Header styles
	HeaderStyle lipgloss.Style
	// Selection style (bold; the gutter carries selection identity)
	SelectedStyle lipgloss.Style
	// Draft PR style (dimmed)
	DraftStyle lipgloss.Style
	// Changed PR style (highlight foreground; the gutter remains distinct)
	ChangedStyle lipgloss.Style
	// Status styles
	StatusPassingStyle          lipgloss.Style
	StatusFailingStyle          lipgloss.Style
	StatusPendingStyle          lipgloss.Style
	StatusNoneStyle             lipgloss.Style
	StatusChangesRequestedStyle lipgloss.Style
	// Merge status styles
	MergeReadyStyle    lipgloss.Style
	MergeBehindStyle   lipgloss.Style
	MergeBlockedStyle  lipgloss.Style
	MergeConflictStyle lipgloss.Style
	MergeUnknownStyle  lipgloss.Style
	// Modal styles
	ModalStyle        lipgloss.Style
	ModalTitleStyle   lipgloss.Style
	ModalSuccessStyle lipgloss.Style
	ModalErrorStyle   lipgloss.Style
	// Status bar style
	StatusBarStyle lipgloss.Style
	// Normal text style
	NormalStyle lipgloss.Style
	// Dim text style
	DimStyle lipgloss.Style
	// Spinner style for loading indicator
	SpinnerStyle lipgloss.Style
	// Organization header collapsed indicator
	CollapsedIndicator string
	// Organization header expanded indicator
	ExpandedIndicator string
}

// NewStyles creates a new Styles instance with all styles configured.
func NewStyles() *Styles {
	return &Styles{
		HeaderStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite),

		SelectedStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite),

		DraftStyle: lipgloss.NewStyle().
			Foreground(ColorDim),

		ChangedStyle: lipgloss.NewStyle().
			Foreground(ColorHighlight),

		StatusPassingStyle: lipgloss.NewStyle().
			Foreground(ColorGreen),

		StatusFailingStyle: lipgloss.NewStyle().
			Foreground(ColorRed),

		StatusPendingStyle: lipgloss.NewStyle().
			Foreground(ColorYellow),

		StatusNoneStyle: lipgloss.NewStyle().
			Foreground(ColorGray),

		StatusChangesRequestedStyle: lipgloss.NewStyle().
			Foreground(ColorRed),

		MergeReadyStyle: lipgloss.NewStyle().
			Foreground(ColorGreen),

		MergeBehindStyle: lipgloss.NewStyle().
			Foreground(ColorYellow),

		MergeBlockedStyle: lipgloss.NewStyle().
			Foreground(ColorYellow),

		MergeConflictStyle: lipgloss.NewStyle().
			Foreground(ColorRed),

		MergeUnknownStyle: lipgloss.NewStyle().
			Foreground(ColorGray),

		ModalStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorHighlight).
			Padding(1, 2),

		ModalTitleStyle: lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite),

		ModalSuccessStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorGreen).
			Padding(1, 2),

		ModalErrorStyle: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorRed).
			Padding(1, 2),

		StatusBarStyle: lipgloss.NewStyle().
			Background(lipgloss.Color("#1F2937")).
			Foreground(ColorWhite).
			Padding(0, 1),

		NormalStyle: lipgloss.NewStyle().
			Foreground(ColorWhite),

		DimStyle: lipgloss.NewStyle().
			Foreground(ColorDim),

		SpinnerStyle: lipgloss.NewStyle().
			Foreground(ColorHighlight),

		CollapsedIndicator: "▸",
		ExpandedIndicator:  "▾",
	}
}
