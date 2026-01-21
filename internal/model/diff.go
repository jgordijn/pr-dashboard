package model

// ChangeType represents the type of change detected in a PR.
type ChangeType int

const (
	// ChangeTypeNew indicates a PR appeared in the list.
	ChangeTypeNew ChangeType = iota
	// ChangeTypeRemoved indicates a PR is no longer in the list.
	ChangeTypeRemoved
	// ChangeTypeReviewStatus indicates review status changed.
	ChangeTypeReviewStatus
	// ChangeTypeCheckStatus indicates check status changed.
	ChangeTypeCheckStatus
	// ChangeTypeMergeStatus indicates merge status changed.
	ChangeTypeMergeStatus
	// ChangeTypeThreadCount indicates unresolved thread count changed.
	ChangeTypeThreadCount
)

// String returns the display string for the change type.
func (c ChangeType) String() string {
	switch c {
	case ChangeTypeNew:
		return "New"
	case ChangeTypeRemoved:
		return "Removed"
	case ChangeTypeReviewStatus:
		return "Review Status"
	case ChangeTypeCheckStatus:
		return "Check Status"
	case ChangeTypeMergeStatus:
		return "Merge Status"
	case ChangeTypeThreadCount:
		return "Thread Count"
	default:
		return "Unknown"
	}
}

// Change represents a detected change in a PR.
type Change struct {
	// Key is the PR key that changed.
	Key string

	// Type is the type of change.
	Type ChangeType

	// OldValue is the previous value (for display).
	OldValue string

	// NewValue is the new value (for display).
	NewValue string
}

// DetectChanges compares old and new PR lists and returns detected changes.
// Ignores DaysOpen changes unless crossing a day boundary (handled by spec).
func DetectChanges(old, new []PullRequest) []Change {
	var changes []Change

	// Build maps for efficient lookup
	oldMap := make(map[string]PullRequest)
	for _, pr := range old {
		oldMap[pr.Key] = pr
	}

	newMap := make(map[string]PullRequest)
	for _, pr := range new {
		newMap[pr.Key] = pr
	}

	// Detect new and modified PRs
	for _, newPR := range new {
		oldPR, existed := oldMap[newPR.Key]
		if !existed {
			// New PR appeared
			changes = append(changes, Change{
				Key:      newPR.Key,
				Type:     ChangeTypeNew,
				NewValue: newPR.Title,
			})
			continue
		}

		// Check for status changes
		if oldPR.ReviewStatus != newPR.ReviewStatus {
			changes = append(changes, Change{
				Key:      newPR.Key,
				Type:     ChangeTypeReviewStatus,
				OldValue: oldPR.ReviewStatus.String(),
				NewValue: newPR.ReviewStatus.String(),
			})
		}

		if oldPR.CheckStatus != newPR.CheckStatus {
			changes = append(changes, Change{
				Key:      newPR.Key,
				Type:     ChangeTypeCheckStatus,
				OldValue: oldPR.CheckStatus.String(),
				NewValue: newPR.CheckStatus.String(),
			})
		}

		if oldPR.MergeStatus != newPR.MergeStatus {
			changes = append(changes, Change{
				Key:      newPR.Key,
				Type:     ChangeTypeMergeStatus,
				OldValue: oldPR.MergeStatus.String(),
				NewValue: newPR.MergeStatus.String(),
			})
		}

		if oldPR.UnresolvedCount != newPR.UnresolvedCount {
			changes = append(changes, Change{
				Key:      newPR.Key,
				Type:     ChangeTypeThreadCount,
				OldValue: oldPR.UnresolvedThreads,
				NewValue: newPR.UnresolvedThreads,
			})
		}

		// Note: DaysOpen changes are intentionally ignored per spec
		// (ignore time-derived changes unless crossing day boundary)
	}

	// Detect removed PRs
	for _, oldPR := range old {
		if _, exists := newMap[oldPR.Key]; !exists {
			changes = append(changes, Change{
				Key:      oldPR.Key,
				Type:     ChangeTypeRemoved,
				OldValue: oldPR.Title,
			})
		}
	}

	return changes
}

// GetChangedKeys returns a set of PR keys that have any changes.
// Excludes removed PRs since they're no longer visible.
func GetChangedKeys(changes []Change) map[string]bool {
	keys := make(map[string]bool)
	for _, change := range changes {
		if change.Type != ChangeTypeRemoved {
			keys[change.Key] = true
		}
	}
	return keys
}

// HasChanges returns true if any changes were detected (excluding removals).
func HasChanges(changes []Change) bool {
	for _, change := range changes {
		if change.Type != ChangeTypeRemoved {
			return true
		}
	}
	return false
}

// FilterVisibleChanges returns changes for keys that are still in the new list.
func FilterVisibleChanges(changes []Change, newPRs []PullRequest) []Change {
	newMap := make(map[string]bool)
	for _, pr := range newPRs {
		newMap[pr.Key] = true
	}

	var visible []Change
	for _, change := range changes {
		if change.Type == ChangeTypeRemoved || newMap[change.Key] {
			visible = append(visible, change)
		}
	}
	return visible
}
