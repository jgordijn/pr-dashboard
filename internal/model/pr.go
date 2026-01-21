// Package model provides domain types and transformers for PR data.
package model

import "time"

// ReviewStatus represents the aggregated review state of a PR.
type ReviewStatus int

const (
	// ReviewStatusNone indicates no review activity.
	ReviewStatusNone ReviewStatus = iota
	// ReviewStatusReviewRequired indicates the PR is awaiting review.
	ReviewStatusReviewRequired
	// ReviewStatusChangesRequested indicates a reviewer has requested changes.
	ReviewStatusChangesRequested
	// ReviewStatusApproved indicates the PR has been approved.
	ReviewStatusApproved
)

// String returns the display string for the review status.
func (s ReviewStatus) String() string {
	switch s {
	case ReviewStatusApproved:
		return "Approved"
	case ReviewStatusChangesRequested:
		return "Changes Requested"
	case ReviewStatusReviewRequired:
		return "Review Required"
	case ReviewStatusNone:
		return "None"
	default:
		return "Unknown"
	}
}

// CheckStatus represents the aggregated CI status of a PR.
type CheckStatus int

const (
	// CheckStatusNone indicates no checks are configured.
	CheckStatusNone CheckStatus = iota
	// CheckStatusPending indicates checks are in progress.
	CheckStatusPending
	// CheckStatusPassing indicates all checks passed.
	CheckStatusPassing
	// CheckStatusFailing indicates one or more checks failed.
	CheckStatusFailing
)

// String returns the display string for the check status.
func (s CheckStatus) String() string {
	switch s {
	case CheckStatusPassing:
		return "Passing"
	case CheckStatusFailing:
		return "Failing"
	case CheckStatusPending:
		return "Pending"
	case CheckStatusNone:
		return "None"
	default:
		return "Unknown"
	}
}

// MergeStatus represents the computed merge readiness of a PR.
type MergeStatus int

const (
	// MergeStatusUnknown indicates the merge status cannot be determined.
	MergeStatusUnknown MergeStatus = iota
	// MergeStatusReady indicates the PR can be merged (MERGEABLE + CLEAN).
	MergeStatusReady
	// MergeStatusBehind indicates the branch is behind base (MERGEABLE + BEHIND).
	MergeStatusBehind
	// MergeStatusBlocked indicates the merge is blocked (MERGEABLE + BLOCKED).
	MergeStatusBlocked
	// MergeStatusConflicts indicates merge conflicts exist (CONFLICTING).
	MergeStatusConflicts
	// MergeStatusDirty indicates a dirty state (DIRTY).
	MergeStatusDirty
	// MergeStatusUnstable indicates an unstable state (UNSTABLE).
	MergeStatusUnstable
	// MergeStatusHasHooks indicates hooks are present (HAS_HOOKS).
	MergeStatusHasHooks
	// MergeStatusDraft indicates the PR is a draft (overrides other states).
	MergeStatusDraft
)

// String returns the display string for the merge status.
func (s MergeStatus) String() string {
	switch s {
	case MergeStatusReady:
		return "Clean"
	case MergeStatusBehind:
		return "Behind"
	case MergeStatusBlocked:
		return "Blocked"
	case MergeStatusConflicts:
		return "Conflicts"
	case MergeStatusDirty:
		return "Dirty"
	case MergeStatusUnstable:
		return "Unstable"
	case MergeStatusHasHooks:
		return "Has Hooks"
	case MergeStatusDraft:
		return "Draft"
	case MergeStatusUnknown:
		return "Unknown"
	default:
		return "Unknown"
	}
}

// DisplayMode represents the UI display mode.
type DisplayMode int

const (
	// DisplayModeFull shows all columns with full details.
	DisplayModeFull DisplayMode = iota
	// DisplayModeCompact shows summary icons with counts.
	DisplayModeCompact
	// DisplayModeMinimal shows only essential info.
	DisplayModeMinimal
)

// String returns the display string for the display mode.
func (m DisplayMode) String() string {
	switch m {
	case DisplayModeFull:
		return "full"
	case DisplayModeCompact:
		return "compact"
	case DisplayModeMinimal:
		return "minimal"
	default:
		return "full"
	}
}

// ParseDisplayMode converts a string to a DisplayMode.
func ParseDisplayMode(s string) DisplayMode {
	switch s {
	case "full":
		return DisplayModeFull
	case "compact":
		return DisplayModeCompact
	case "minimal":
		return DisplayModeMinimal
	default:
		return DisplayModeFull
	}
}

// PullRequest represents the domain model for a pull request.
type PullRequest struct {
	// Key is the stable identifier: owner/repo#number
	Key string

	// Title is the PR title.
	Title string

	// Author is the GitHub username of the PR author.
	Author string

	// DaysOpen is the number of days since the PR was created.
	DaysOpen int

	// ReviewStatus is the aggregated review state.
	ReviewStatus ReviewStatus

	// CheckStatus is the aggregated CI status.
	CheckStatus CheckStatus

	// MergeStatus is the computed merge readiness.
	MergeStatus MergeStatus

	// Mergeable is the raw mergeable state from GitHub API.
	// Values: "MERGEABLE", "CONFLICTING", "UNKNOWN", or empty if null.
	Mergeable string

	// MergeStateStatus is the raw merge state status from GitHub API.
	// Values: "CLEAN", "BEHIND", "BLOCKED", "DIRTY", "UNSTABLE", "HAS_HOOKS", "UNKNOWN", or empty if null.
	MergeStateStatus string

	// UnresolvedThreads is the count string with "+" suffix for >100.
	UnresolvedThreads string

	// UnresolvedCount is the numeric count of unresolved threads (for change detection).
	UnresolvedCount int

	// Reviewers is a flattened list with "team:" prefix for teams.
	Reviewers []string

	// IsDraft indicates if the PR is a draft.
	IsDraft bool

	// URL is the full GitHub URL.
	URL string

	// Organization is the owner login for grouping.
	Organization string

	// Repository is the repo name.
	Repository string

	// Number is the PR number.
	Number int

	// UpdatedAt is the last update time for sorting.
	UpdatedAt time.Time

	// CreatedAt is the creation time for calculating DaysOpen.
	CreatedAt time.Time
}

// CanUpdateBranch returns true if the branch can be updated.
// Branch update is only allowed when mergeable=MERGEABLE and mergeStateStatus=BEHIND.
func (pr *PullRequest) CanUpdateBranch() bool {
	return pr.Mergeable == "MERGEABLE" && pr.MergeStateStatus == "BEHIND"
}

// UpdateBranchBlockedReason returns the reason why branch update is blocked.
// Returns empty string if update is allowed.
func (pr *PullRequest) UpdateBranchBlockedReason() string {
	if pr.IsDraft {
		return "Cannot update: PR is a draft"
	}
	if pr.Mergeable == "CONFLICTING" {
		return "Cannot update: PR has merge conflicts"
	}
	if pr.Mergeable == "UNKNOWN" || pr.Mergeable == "" {
		return "Cannot update: merge status unknown (GitHub is still computing)"
	}
	if pr.MergeStateStatus == "CLEAN" {
		return "Branch is already up to date"
	}
	if pr.MergeStateStatus == "BLOCKED" {
		return "Cannot update: merge is blocked"
	}
	if pr.CanUpdateBranch() {
		return ""
	}
	return "Cannot update branch"
}
