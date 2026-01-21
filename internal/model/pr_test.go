package model

import "testing"

func TestReviewStatus_String(t *testing.T) {
	tests := []struct {
		status ReviewStatus
		want   string
	}{
		{ReviewStatusNone, "None"},
		{ReviewStatusReviewRequired, "Review Required"},
		{ReviewStatusChangesRequested, "Changes Requested"},
		{ReviewStatusApproved, "Approved"},
		{ReviewStatus(99), "Unknown"}, // Invalid value
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("ReviewStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckStatus_String(t *testing.T) {
	tests := []struct {
		status CheckStatus
		want   string
	}{
		{CheckStatusNone, "None"},
		{CheckStatusPending, "Pending"},
		{CheckStatusPassing, "Passing"},
		{CheckStatusFailing, "Failing"},
		{CheckStatus(99), "Unknown"}, // Invalid value
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("CheckStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMergeStatus_String(t *testing.T) {
	tests := []struct {
		status MergeStatus
		want   string
	}{
		{MergeStatusUnknown, "Unknown"},
		{MergeStatusReady, "Clean"},
		{MergeStatusBehind, "Behind"},
		{MergeStatusBlocked, "Blocked"},
		{MergeStatusConflicts, "Conflicts"},
		{MergeStatusDirty, "Dirty"},
		{MergeStatusUnstable, "Unstable"},
		{MergeStatusHasHooks, "Has Hooks"},
		{MergeStatusDraft, "Draft"},
		{MergeStatus(99), "Unknown"}, // Invalid value
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.status.String(); got != tt.want {
				t.Errorf("MergeStatus.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDisplayMode_String(t *testing.T) {
	tests := []struct {
		mode DisplayMode
		want string
	}{
		{DisplayModeFull, "full"},
		{DisplayModeCompact, "compact"},
		{DisplayModeMinimal, "minimal"},
		{DisplayMode(99), "full"}, // Invalid defaults to full
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.mode.String(); got != tt.want {
				t.Errorf("DisplayMode.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseDisplayMode(t *testing.T) {
	tests := []struct {
		input string
		want  DisplayMode
	}{
		{"full", DisplayModeFull},
		{"compact", DisplayModeCompact},
		{"minimal", DisplayModeMinimal},
		{"invalid", DisplayModeFull}, // Invalid defaults to full
		{"", DisplayModeFull},        // Empty defaults to full
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := ParseDisplayMode(tt.input); got != tt.want {
				t.Errorf("ParseDisplayMode(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestPullRequest_CanUpdateBranch(t *testing.T) {
	tests := []struct {
		name             string
		mergeable        string
		mergeStateStatus string
		want             bool
	}{
		{"behind and mergeable", "MERGEABLE", "BEHIND", true},
		{"clean and mergeable", "MERGEABLE", "CLEAN", false},
		{"conflicts", "CONFLICTING", "BEHIND", false},
		{"unknown", "UNKNOWN", "BEHIND", false},
		{"empty mergeable", "", "BEHIND", false},
		{"empty state status", "MERGEABLE", "", false},
		{"blocked", "MERGEABLE", "BLOCKED", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := PullRequest{
				Mergeable:        tt.mergeable,
				MergeStateStatus: tt.mergeStateStatus,
			}
			if got := pr.CanUpdateBranch(); got != tt.want {
				t.Errorf("CanUpdateBranch() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPullRequest_UpdateBranchBlockedReason(t *testing.T) {
	tests := []struct {
		name             string
		isDraft          bool
		mergeable        string
		mergeStateStatus string
		wantEmpty        bool
		wantContains     string
	}{
		{
			name:             "can update",
			mergeable:        "MERGEABLE",
			mergeStateStatus: "BEHIND",
			wantEmpty:        true,
		},
		{
			name:         "draft PR",
			isDraft:      true,
			wantContains: "draft",
		},
		{
			name:         "conflicts",
			mergeable:    "CONFLICTING",
			wantContains: "conflicts",
		},
		{
			name:         "unknown mergeable",
			mergeable:    "UNKNOWN",
			wantContains: "unknown",
		},
		{
			name:         "empty mergeable",
			mergeable:    "",
			wantContains: "unknown",
		},
		{
			name:             "already clean",
			mergeable:        "MERGEABLE",
			mergeStateStatus: "CLEAN",
			wantContains:     "already up to date",
		},
		{
			name:             "blocked",
			mergeable:        "MERGEABLE",
			mergeStateStatus: "BLOCKED",
			wantContains:     "blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := PullRequest{
				IsDraft:          tt.isDraft,
				Mergeable:        tt.mergeable,
				MergeStateStatus: tt.mergeStateStatus,
			}
			got := pr.UpdateBranchBlockedReason()

			if tt.wantEmpty {
				if got != "" {
					t.Errorf("UpdateBranchBlockedReason() = %q, want empty", got)
				}
				return
			}

			if got == "" {
				t.Errorf("UpdateBranchBlockedReason() = empty, want to contain %q", tt.wantContains)
				return
			}

			// Check contains (case-insensitive not needed here as we control the strings)
			found := false
			for i := 0; i <= len(got)-len(tt.wantContains); i++ {
				if got[i:i+len(tt.wantContains)] == tt.wantContains {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("UpdateBranchBlockedReason() = %q, want to contain %q", got, tt.wantContains)
			}
		})
	}
}
