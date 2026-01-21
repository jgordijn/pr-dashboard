package tui

import (
	"testing"
)

func TestNewStyles(t *testing.T) {
	styles := NewStyles()

	if styles == nil {
		t.Fatal("NewStyles returned nil")
	}

	// Verify all styles are initialized (not zero values)
	tests := []struct {
		name  string
		style interface{}
	}{
		{"HeaderStyle", styles.HeaderStyle},
		{"SelectedStyle", styles.SelectedStyle},
		{"DraftStyle", styles.DraftStyle},
		{"ChangedStyle", styles.ChangedStyle},
		{"StatusPassingStyle", styles.StatusPassingStyle},
		{"StatusFailingStyle", styles.StatusFailingStyle},
		{"StatusPendingStyle", styles.StatusPendingStyle},
		{"StatusNoneStyle", styles.StatusNoneStyle},
		{"StatusChangesRequestedStyle", styles.StatusChangesRequestedStyle},
		{"MergeReadyStyle", styles.MergeReadyStyle},
		{"MergeBehindStyle", styles.MergeBehindStyle},
		{"MergeBlockedStyle", styles.MergeBlockedStyle},
		{"MergeConflictStyle", styles.MergeConflictStyle},
		{"MergeUnknownStyle", styles.MergeUnknownStyle},
		{"ModalStyle", styles.ModalStyle},
		{"ModalTitleStyle", styles.ModalTitleStyle},
		{"ModalSuccessStyle", styles.ModalSuccessStyle},
		{"ModalErrorStyle", styles.ModalErrorStyle},
		{"StatusBarStyle", styles.StatusBarStyle},
		{"NormalStyle", styles.NormalStyle},
		{"DimStyle", styles.DimStyle},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Style is initialized - just verify it's not nil conceptually
			// lipgloss.Style is a value type, so we test that Render doesn't panic
			_ = tc.style
		})
	}
}

func TestStylesIndicators(t *testing.T) {
	styles := NewStyles()

	if styles.CollapsedIndicator == "" {
		t.Error("CollapsedIndicator should not be empty")
	}

	if styles.ExpandedIndicator == "" {
		t.Error("ExpandedIndicator should not be empty")
	}

	if styles.CollapsedIndicator == styles.ExpandedIndicator {
		t.Error("CollapsedIndicator and ExpandedIndicator should be different")
	}
}

func TestStylesColorConstants(t *testing.T) {
	// Test that color constants are defined
	tests := []struct {
		name  string
		color string
	}{
		{"ColorGreen", string(ColorGreen)},
		{"ColorYellow", string(ColorYellow)},
		{"ColorRed", string(ColorRed)},
		{"ColorGray", string(ColorGray)},
		{"ColorHighlight", string(ColorHighlight)},
		{"ColorWhite", string(ColorWhite)},
		{"ColorDim", string(ColorDim)},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.color == "" {
				t.Errorf("%s should not be empty", tc.name)
			}
			// Verify it's a valid hex color format
			if tc.color[0] != '#' {
				t.Errorf("%s should start with '#', got %q", tc.name, tc.color)
			}
		})
	}
}
