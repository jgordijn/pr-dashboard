package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keyboard bindings for the TUI.
// Per tui-navigation/spec.md, supports vim-style and arrow key navigation.
type KeyMap struct {
	// Navigation
	Up     key.Binding
	Down   key.Binding
	Top    key.Binding
	Bottom key.Binding

	// Organization controls
	ToggleOrg     key.Binding
	ToggleAllOrgs key.Binding

	// Display controls
	ToggleDrafts    key.Binding
	CycleDisplayMode key.Binding

	// Watch mode
	ToggleWatch key.Binding

	// Actions
	UpdateBranch  key.Binding
	Refresh       key.Binding
	OpenBrowser   key.Binding
	SwitchAccount key.Binding

	// UI
	Help key.Binding
	Quit key.Binding
}

// NewKeyMap creates a KeyMap with default bindings per spec.
func NewKeyMap() *KeyMap {
	return &KeyMap{
		Up: key.NewBinding(
			key.WithKeys("k", "up"),
			key.WithHelp("k/up", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down"),
			key.WithHelp("j/down", "move down"),
		),
		Top: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("gg", "go to top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "go to bottom"),
		),
		ToggleOrg: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "toggle org collapse"),
		),
		ToggleAllOrgs: key.NewBinding(
			key.WithKeys("O"),
			key.WithHelp("O", "toggle all orgs"),
		),
		ToggleDrafts: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "toggle drafts"),
		),
		CycleDisplayMode: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "cycle display mode"),
		),
		ToggleWatch: key.NewBinding(
			key.WithKeys("w"),
			key.WithHelp("w", "toggle watch mode"),
		),
		UpdateBranch: key.NewBinding(
			key.WithKeys("u"),
			key.WithHelp("u", "update branch"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "refresh"),
		),
		OpenBrowser: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "open in browser"),
		),
		SwitchAccount: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "switch account"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc"),
			key.WithHelp("q/esc", "quit"),
		),
	}
}

// ShortHelp returns a condensed list of key bindings for the help bar.
func (k *KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.OpenBrowser, k.Refresh, k.Help, k.Quit}
}

// FullHelp returns the full list of key bindings organized by category.
func (k *KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Navigation
		{k.Up, k.Down, k.Top, k.Bottom},
		// Organization
		{k.ToggleOrg, k.ToggleAllOrgs},
		// Display
		{k.CycleDisplayMode, k.ToggleDrafts, k.ToggleWatch},
		// Actions
		{k.UpdateBranch, k.Refresh, k.OpenBrowser, k.SwitchAccount},
		// Other
		{k.Help, k.Quit},
	}
}
