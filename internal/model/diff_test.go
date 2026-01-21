package model

import (
	"testing"
)

func TestChangeType_String(t *testing.T) {
	tests := []struct {
		changeType ChangeType
		want       string
	}{
		{ChangeTypeNew, "New"},
		{ChangeTypeRemoved, "Removed"},
		{ChangeTypeReviewStatus, "Review Status"},
		{ChangeTypeCheckStatus, "Check Status"},
		{ChangeTypeMergeStatus, "Merge Status"},
		{ChangeTypeThreadCount, "Thread Count"},
		{ChangeType(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.changeType.String(); got != tt.want {
				t.Errorf("ChangeType.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDetectChanges_NewPR(t *testing.T) {
	old := []PullRequest{}
	new := []PullRequest{
		{Key: "org/repo#1", Title: "New PR"},
	}

	changes := DetectChanges(old, new)

	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}

	c := changes[0]
	if c.Key != "org/repo#1" {
		t.Errorf("Key = %s, want org/repo#1", c.Key)
	}
	if c.Type != ChangeTypeNew {
		t.Errorf("Type = %v, want New", c.Type)
	}
	if c.NewValue != "New PR" {
		t.Errorf("NewValue = %s, want New PR", c.NewValue)
	}
}

func TestDetectChanges_RemovedPR(t *testing.T) {
	old := []PullRequest{
		{Key: "org/repo#1", Title: "Old PR"},
	}
	new := []PullRequest{}

	changes := DetectChanges(old, new)

	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}

	c := changes[0]
	if c.Key != "org/repo#1" {
		t.Errorf("Key = %s, want org/repo#1", c.Key)
	}
	if c.Type != ChangeTypeRemoved {
		t.Errorf("Type = %v, want Removed", c.Type)
	}
	if c.OldValue != "Old PR" {
		t.Errorf("OldValue = %s, want Old PR", c.OldValue)
	}
}

func TestDetectChanges_ReviewStatusChanged(t *testing.T) {
	old := []PullRequest{
		{Key: "org/repo#1", ReviewStatus: ReviewStatusReviewRequired},
	}
	new := []PullRequest{
		{Key: "org/repo#1", ReviewStatus: ReviewStatusApproved},
	}

	changes := DetectChanges(old, new)

	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}

	c := changes[0]
	if c.Type != ChangeTypeReviewStatus {
		t.Errorf("Type = %v, want ReviewStatus", c.Type)
	}
	if c.OldValue != "Review Required" {
		t.Errorf("OldValue = %s, want Review Required", c.OldValue)
	}
	if c.NewValue != "Approved" {
		t.Errorf("NewValue = %s, want Approved", c.NewValue)
	}
}

func TestDetectChanges_CheckStatusChanged(t *testing.T) {
	old := []PullRequest{
		{Key: "org/repo#1", CheckStatus: CheckStatusPending},
	}
	new := []PullRequest{
		{Key: "org/repo#1", CheckStatus: CheckStatusPassing},
	}

	changes := DetectChanges(old, new)

	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}

	c := changes[0]
	if c.Type != ChangeTypeCheckStatus {
		t.Errorf("Type = %v, want CheckStatus", c.Type)
	}
}

func TestDetectChanges_MergeStatusChanged(t *testing.T) {
	old := []PullRequest{
		{Key: "org/repo#1", MergeStatus: MergeStatusBehind},
	}
	new := []PullRequest{
		{Key: "org/repo#1", MergeStatus: MergeStatusReady},
	}

	changes := DetectChanges(old, new)

	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}

	c := changes[0]
	if c.Type != ChangeTypeMergeStatus {
		t.Errorf("Type = %v, want MergeStatus", c.Type)
	}
}

func TestDetectChanges_ThreadCountChanged(t *testing.T) {
	old := []PullRequest{
		{Key: "org/repo#1", UnresolvedCount: 5, UnresolvedThreads: "5"},
	}
	new := []PullRequest{
		{Key: "org/repo#1", UnresolvedCount: 3, UnresolvedThreads: "3"},
	}

	changes := DetectChanges(old, new)

	if len(changes) != 1 {
		t.Fatalf("len(changes) = %d, want 1", len(changes))
	}

	c := changes[0]
	if c.Type != ChangeTypeThreadCount {
		t.Errorf("Type = %v, want ThreadCount", c.Type)
	}
	if c.OldValue != "5" {
		t.Errorf("OldValue = %s, want 5", c.OldValue)
	}
	if c.NewValue != "3" {
		t.Errorf("NewValue = %s, want 3", c.NewValue)
	}
}

func TestDetectChanges_NoChanges(t *testing.T) {
	pr := PullRequest{
		Key:               "org/repo#1",
		ReviewStatus:      ReviewStatusApproved,
		CheckStatus:       CheckStatusPassing,
		MergeStatus:       MergeStatusReady,
		UnresolvedCount:   0,
		UnresolvedThreads: "0",
		DaysOpen:          5,
	}

	old := []PullRequest{pr}
	new := []PullRequest{pr}

	changes := DetectChanges(old, new)

	if len(changes) != 0 {
		t.Errorf("len(changes) = %d, want 0", len(changes))
	}
}

func TestDetectChanges_DaysOpenIgnored(t *testing.T) {
	old := []PullRequest{
		{Key: "org/repo#1", DaysOpen: 5},
	}
	new := []PullRequest{
		{Key: "org/repo#1", DaysOpen: 6},
	}

	changes := DetectChanges(old, new)

	// DaysOpen changes should be ignored per spec
	if len(changes) != 0 {
		t.Errorf("len(changes) = %d, want 0 (DaysOpen should be ignored)", len(changes))
	}
}

func TestDetectChanges_MultipleChanges(t *testing.T) {
	old := []PullRequest{
		{
			Key:          "org/repo#1",
			ReviewStatus: ReviewStatusReviewRequired,
			CheckStatus:  CheckStatusPending,
		},
		{Key: "org/repo#2", MergeStatus: MergeStatusBehind},
	}
	new := []PullRequest{
		{
			Key:          "org/repo#1",
			ReviewStatus: ReviewStatusApproved,
			CheckStatus:  CheckStatusPassing,
		},
		{Key: "org/repo#3", Title: "Brand New"},
	}

	changes := DetectChanges(old, new)

	// Should detect:
	// 1. ReviewStatus change for #1
	// 2. CheckStatus change for #1
	// 3. Removal of #2
	// 4. New PR #3
	if len(changes) != 4 {
		t.Fatalf("len(changes) = %d, want 4", len(changes))
	}

	// Verify types are present
	types := make(map[ChangeType]int)
	for _, c := range changes {
		types[c.Type]++
	}

	if types[ChangeTypeReviewStatus] != 1 {
		t.Error("Expected 1 ReviewStatus change")
	}
	if types[ChangeTypeCheckStatus] != 1 {
		t.Error("Expected 1 CheckStatus change")
	}
	if types[ChangeTypeRemoved] != 1 {
		t.Error("Expected 1 Removed change")
	}
	if types[ChangeTypeNew] != 1 {
		t.Error("Expected 1 New change")
	}
}

func TestDetectChanges_Empty(t *testing.T) {
	changes := DetectChanges(nil, nil)
	if len(changes) != 0 {
		t.Errorf("DetectChanges(nil, nil) = %d changes, want 0", len(changes))
	}

	changes = DetectChanges([]PullRequest{}, []PullRequest{})
	if len(changes) != 0 {
		t.Errorf("DetectChanges([], []) = %d changes, want 0", len(changes))
	}
}

func TestGetChangedKeys(t *testing.T) {
	changes := []Change{
		{Key: "org/repo#1", Type: ChangeTypeNew},
		{Key: "org/repo#2", Type: ChangeTypeReviewStatus},
		{Key: "org/repo#3", Type: ChangeTypeRemoved}, // Should be excluded
		{Key: "org/repo#1", Type: ChangeTypeCheckStatus}, // Duplicate key
	}

	keys := GetChangedKeys(changes)

	if len(keys) != 2 {
		t.Errorf("len(keys) = %d, want 2", len(keys))
	}

	if !keys["org/repo#1"] {
		t.Error("Expected key org/repo#1")
	}
	if !keys["org/repo#2"] {
		t.Error("Expected key org/repo#2")
	}
	if keys["org/repo#3"] {
		t.Error("Removed key should not be included")
	}
}

func TestGetChangedKeys_Empty(t *testing.T) {
	keys := GetChangedKeys(nil)
	if len(keys) != 0 {
		t.Errorf("GetChangedKeys(nil) = %d keys, want 0", len(keys))
	}
}

func TestHasChanges(t *testing.T) {
	tests := []struct {
		name    string
		changes []Change
		want    bool
	}{
		{"empty", nil, false},
		{"only removed", []Change{{Type: ChangeTypeRemoved}}, false},
		{"has new", []Change{{Type: ChangeTypeNew}}, true},
		{"has status change", []Change{{Type: ChangeTypeCheckStatus}}, true},
		{"mixed with removed", []Change{{Type: ChangeTypeRemoved}, {Type: ChangeTypeNew}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasChanges(tt.changes); got != tt.want {
				t.Errorf("HasChanges() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterVisibleChanges(t *testing.T) {
	newPRs := []PullRequest{
		{Key: "org/repo#1"},
		{Key: "org/repo#3"},
	}

	changes := []Change{
		{Key: "org/repo#1", Type: ChangeTypeNew},
		{Key: "org/repo#2", Type: ChangeTypeRemoved},       // Removed are kept
		{Key: "org/repo#3", Type: ChangeTypeCheckStatus},
		{Key: "org/repo#4", Type: ChangeTypeReviewStatus}, // Not in new list, filtered
	}

	visible := FilterVisibleChanges(changes, newPRs)

	if len(visible) != 3 {
		t.Fatalf("len(visible) = %d, want 3", len(visible))
	}

	// Verify which changes are kept
	keys := make(map[string]bool)
	for _, c := range visible {
		keys[c.Key] = true
	}

	if !keys["org/repo#1"] {
		t.Error("Expected org/repo#1 to be visible")
	}
	if !keys["org/repo#2"] {
		t.Error("Expected removed org/repo#2 to be visible")
	}
	if !keys["org/repo#3"] {
		t.Error("Expected org/repo#3 to be visible")
	}
	if keys["org/repo#4"] {
		t.Error("org/repo#4 should not be visible")
	}
}

func TestFilterVisibleChanges_Empty(t *testing.T) {
	visible := FilterVisibleChanges(nil, nil)
	if visible != nil {
		t.Errorf("FilterVisibleChanges(nil, nil) = %v, want nil", visible)
	}
}
