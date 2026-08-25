package tui

import (
	"testing"
	"time"

	"github.com/jgordijn/pr-dashboard/internal/config"
	"github.com/jgordijn/pr-dashboard/internal/model"
)

func TestNewModel(t *testing.T) {
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

	if m.Config != cfg {
		t.Error("Config not set correctly")
	}

	if m.DisplayMode != model.DisplayModeFull {
		t.Errorf("expected DisplayModeFull, got %v", m.DisplayMode)
	}

	if !m.ShowDrafts {
		t.Error("ShowDrafts should be true")
	}

	if m.WatchMode {
		t.Error("WatchMode should be false initially")
	}

	if m.Keys == nil {
		t.Error("Keys should be initialized")
	}

	if m.Styles == nil {
		t.Error("Styles should be initialized")
	}

	if m.ChangedKeys == nil {
		t.Error("ChangedKeys map should be initialized")
	}
}

func TestNewModelSelectsASCIICharacters(t *testing.T) {
	cfg := &config.Config{Display: config.DisplayConfig{InitialMode: "full", ASCII: true}}
	m := NewModel(cfg, nil)
	if m.Symbols.MergeConflicts != "X" || m.Symbols.Selected != ">" {
		t.Fatalf("unexpected ASCII symbols: %+v", m.Symbols)
	}
}

func TestNewModelDisplayModes(t *testing.T) {
	tests := []struct {
		mode     string
		expected model.DisplayMode
	}{
		{"full", model.DisplayModeFull},
		{"compact", model.DisplayModeCompact},
		{"minimal", model.DisplayModeMinimal},
		{"invalid", model.DisplayModeFull}, // defaults to full
	}

	for _, tc := range tests {
		t.Run(tc.mode, func(t *testing.T) {
			cfg := &config.Config{
				Display: config.DisplayConfig{
					InitialMode: tc.mode,
				},
			}
			m := NewModel(cfg, nil)
			if m.DisplayMode != tc.expected {
				t.Errorf("expected %v, got %v", tc.expected, m.DisplayMode)
			}
		})
	}
}

func TestModelVisiblePRs(t *testing.T) {
	m := Model{
		Groups: []model.PRGroup{
			{
				Organization: "org1",
				Collapsed:    false,
				PRs: []model.PullRequest{
					{Key: "org1/repo#1", IsDraft: false},
					{Key: "org1/repo#2", IsDraft: true},
				},
			},
			{
				Organization: "org2",
				Collapsed:    true,
				PRs: []model.PullRequest{
					{Key: "org2/repo#3", IsDraft: false},
				},
			},
		},
		ShowDrafts: true,
	}

	visible := m.visiblePRs()
	if len(visible) != 2 {
		t.Errorf("expected 2 visible PRs (org2 collapsed), got %d", len(visible))
	}

	// Hide drafts
	m.ShowDrafts = false
	visible = m.visiblePRs()
	if len(visible) != 1 {
		t.Errorf("expected 1 visible PR (drafts hidden, org2 collapsed), got %d", len(visible))
	}
}

func TestModelFindNearestVisibleKey(t *testing.T) {
	m := Model{
		Groups: []model.PRGroup{
			{
				Organization: "org1",
				PRs: []model.PullRequest{
					{Key: "org1/repo#1"},
					{Key: "org1/repo#2"},
				},
			},
		},
		ShowDrafts:  true,
		SelectedKey: "org1/repo#1",
	}

	// Current selection is visible
	key := m.findNearestVisibleKey()
	if key != "org1/repo#1" {
		t.Errorf("expected org1/repo#1, got %s", key)
	}

	// Selection no longer exists
	m.SelectedKey = "nonexistent"
	key = m.findNearestVisibleKey()
	if key != "org1/repo#1" {
		t.Errorf("expected first visible PR org1/repo#1, got %s", key)
	}

	// No visible PRs
	m.Groups = nil
	key = m.findNearestVisibleKey()
	if key != "" {
		t.Errorf("expected empty key when no visible PRs, got %s", key)
	}
}

func TestModelSelectedPR(t *testing.T) {
	pr := model.PullRequest{Key: "org/repo#1", Title: "Test PR"}
	m := Model{
		Groups: []model.PRGroup{
			{
				Organization: "org",
				PRs:          []model.PullRequest{pr},
			},
		},
		SelectedKey: "org/repo#1",
	}

	selected := m.SelectedPR()
	if selected == nil {
		t.Fatal("expected selected PR, got nil")
	}
	if selected.Key != "org/repo#1" {
		t.Errorf("expected org/repo#1, got %s", selected.Key)
	}

	// No selection
	m.SelectedKey = ""
	if m.SelectedPR() != nil {
		t.Error("expected nil when no selection")
	}

	// Invalid selection
	m.SelectedKey = "nonexistent"
	if m.SelectedPR() != nil {
		t.Error("expected nil for nonexistent selection")
	}
}

func TestModelCountVisiblePRs(t *testing.T) {
	m := Model{
		Groups: []model.PRGroup{
			{
				Organization: "org1",
				Collapsed:    false,
				PRs: []model.PullRequest{
					{Key: "org1/repo#1", IsDraft: false},
					{Key: "org1/repo#2", IsDraft: true},
				},
			},
		},
		ShowDrafts: true,
	}

	count := m.countVisiblePRs()
	if count != 2 {
		t.Errorf("expected 2, got %d", count)
	}

	m.ShowDrafts = false
	count = m.countVisiblePRs()
	if count != 1 {
		t.Errorf("expected 1 with drafts hidden, got %d", count)
	}
}

func TestRateLimitInfo(t *testing.T) {
	resetTime := time.Now().Add(time.Hour)
	info := RateLimitInfo{
		Remaining: 100,
		ResetAt:   resetTime,
	}

	if info.Remaining != 100 {
		t.Errorf("expected 100 remaining, got %d", info.Remaining)
	}

	if info.ResetAt != resetTime {
		t.Errorf("expected reset time %v, got %v", resetTime, info.ResetAt)
	}
}
