package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewKeyMap(t *testing.T) {
	km := NewKeyMap()

	if km == nil {
		t.Fatal("NewKeyMap returned nil")
	}
}

func TestKeyMapBindings(t *testing.T) {
	km := NewKeyMap()

	tests := []struct {
		name    string
		binding key.Binding
		keys    []string
	}{
		{"Up", km.Up, []string{"k", "up"}},
		{"Down", km.Down, []string{"j", "down"}},
		{"Top", km.Top, []string{"g"}},
		{"Bottom", km.Bottom, []string{"G"}},
		{"ToggleOrg", km.ToggleOrg, []string{"o"}},
		{"ToggleAllOrgs", km.ToggleAllOrgs, []string{"O"}},
		{"ToggleDrafts", km.ToggleDrafts, []string{"d"}},
		{"CycleDisplayMode", km.CycleDisplayMode, []string{"c"}},
		{"ToggleWatch", km.ToggleWatch, []string{"w"}},
		{"UpdateBranch", km.UpdateBranch, []string{"u"}},
		{"Refresh", km.Refresh, []string{"r"}},
		{"OpenBrowser", km.OpenBrowser, []string{"enter"}},
		{"Help", km.Help, []string{"?"}},
		{"Quit", km.Quit, []string{"q", "esc"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, k := range tc.keys {
				keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
				// Handle special keys
				switch k {
				case "up":
					keyMsg = tea.KeyMsg{Type: tea.KeyUp}
				case "down":
					keyMsg = tea.KeyMsg{Type: tea.KeyDown}
				case "enter":
					keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
				case "esc":
					keyMsg = tea.KeyMsg{Type: tea.KeyEscape}
				}
				if !key.Matches(keyMsg, tc.binding) {
					t.Errorf("key %q should match binding %s", k, tc.name)
				}
			}
		})
	}
}

func TestKeyMapShortHelp(t *testing.T) {
	km := NewKeyMap()
	help := km.ShortHelp()

	if len(help) == 0 {
		t.Error("ShortHelp should return bindings")
	}

	// Verify essential bindings are included
	helpKeys := make(map[string]bool)
	for _, b := range help {
		for _, k := range b.Keys() {
			helpKeys[k] = true
		}
	}

	essentials := []string{"?", "q"}
	for _, k := range essentials {
		if !helpKeys[k] {
			t.Errorf("ShortHelp should include %q binding", k)
		}
	}
}

func TestKeyMapFullHelp(t *testing.T) {
	km := NewKeyMap()
	help := km.FullHelp()

	if len(help) == 0 {
		t.Error("FullHelp should return binding groups")
	}

	// Count total bindings
	totalBindings := 0
	for _, group := range help {
		totalBindings += len(group)
	}

	if totalBindings < 10 {
		t.Error("FullHelp should include all major bindings")
	}
}
