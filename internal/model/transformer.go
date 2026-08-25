package model

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jgordijn/pr-dashboard/internal/github"
)

// TransformPR converts a GitHub API PullRequestNode to a domain PullRequest.
func TransformPR(apiPR github.PullRequestNode) PullRequest {
	pr := PullRequest{
		Title:     apiPR.Title,
		Number:    apiPR.Number,
		IsDraft:   apiPR.IsDraft,
		URL:       apiPR.URL,
		UpdatedAt: apiPR.UpdatedAt,
		CreatedAt: apiPR.CreatedAt,
	}

	// Extract organization and repository
	pr.Organization = apiPR.Repository.Owner.Login
	pr.Repository = apiPR.Repository.Name

	// Generate stable key: owner/repo#number
	pr.Key = fmt.Sprintf("%s/%s#%d", pr.Organization, pr.Repository, pr.Number)

	// Extract author
	if apiPR.Author != nil {
		pr.Author = apiPR.Author.Login
	}

	// Calculate days open
	pr.DaysOpen = calculateDaysOpen(apiPR.CreatedAt, time.Now())

	// Map review status
	pr.ReviewStatus = mapReviewStatus(apiPR.ReviewDecision)

	// Map check status
	pr.CheckStatus = mapCheckStatus(apiPR.Commits)

	// Store raw merge values
	if apiPR.Mergeable != nil {
		pr.Mergeable = *apiPR.Mergeable
	}
	if apiPR.MergeStateStatus != nil {
		pr.MergeStateStatus = *apiPR.MergeStateStatus
	}

	// Derive merge status
	pr.MergeStatus = deriveMergeStatus(apiPR.IsDraft, apiPR.Mergeable, apiPR.MergeStateStatus)

	// Count unresolved threads
	pr.UnresolvedCount, pr.UnresolvedThreads = countUnresolvedThreads(apiPR.ReviewThreads)

	// Flatten reviewers
	pr.Reviewers = flattenReviewers(apiPR.ReviewRequests)

	return pr
}

// TransformPRs converts a slice of GitHub API PullRequestNodes to domain PullRequests.
func TransformPRs(apiPRs []github.PullRequestNode) []PullRequest {
	if len(apiPRs) == 0 {
		return nil
	}

	prs := make([]PullRequest, 0, len(apiPRs))
	for _, apiPR := range apiPRs {
		// Skip non-PullRequest nodes (defensive check)
		if !apiPR.IsPullRequest() {
			continue
		}
		prs = append(prs, TransformPR(apiPR))
	}
	return prs
}

// calculateDaysOpen calculates the number of days since creation.
func calculateDaysOpen(created, now time.Time) int {
	duration := now.Sub(created)
	days := int(duration.Hours() / 24)
	if days < 0 {
		return 0
	}
	return days
}

// mapReviewStatus maps the API reviewDecision to a ReviewStatus.
func mapReviewStatus(decision *string) ReviewStatus {
	if decision == nil {
		return ReviewStatusNone
	}
	switch *decision {
	case "APPROVED":
		return ReviewStatusApproved
	case "CHANGES_REQUESTED":
		return ReviewStatusChangesRequested
	case "REVIEW_REQUIRED":
		return ReviewStatusReviewRequired
	default:
		return ReviewStatusNone
	}
}

// mapCheckStatus maps the API commits to a CheckStatus using statusCheckRollup.state.
func mapCheckStatus(commits github.CommitConnection) CheckStatus {
	if len(commits.Nodes) == 0 {
		return CheckStatusNone
	}

	// Use the first (most recent) commit's status check rollup
	commit := commits.Nodes[0].Commit
	if commit.StatusCheckRollup == nil {
		return CheckStatusNone
	}
	if commit.StatusCheckRollup.State == nil {
		return CheckStatusNone
	}

	switch *commit.StatusCheckRollup.State {
	case "SUCCESS":
		return CheckStatusPassing
	case "FAILURE", "ERROR":
		return CheckStatusFailing
	case "PENDING":
		return CheckStatusPending
	default:
		return CheckStatusNone
	}
}

// deriveMergeStatus derives the MergeStatus from API values.
func deriveMergeStatus(isDraft bool, mergeable, mergeStateStatus *string) MergeStatus {
	// Conflicts are more important than draft state; draft remains a visual style.
	if mergeable != nil && *mergeable == "CONFLICTING" {
		return MergeStatusConflicts
	}
	if isDraft {
		return MergeStatusDraft
	}

	// Handle null/unknown mergeable
	if mergeable == nil {
		return MergeStatusUnknown
	}

	// Handle unknown mergeable state
	if *mergeable == "UNKNOWN" {
		return MergeStatusUnknown
	}

	// At this point, mergeable should be "MERGEABLE"
	// Now check mergeStateStatus
	if mergeStateStatus == nil {
		return MergeStatusUnknown
	}

	switch *mergeStateStatus {
	case "CLEAN":
		return MergeStatusReady
	case "BEHIND":
		return MergeStatusBehind
	case "BLOCKED":
		return MergeStatusBlocked
	case "DIRTY":
		return MergeStatusDirty
	case "UNSTABLE":
		return MergeStatusUnstable
	case "HAS_HOOKS":
		return MergeStatusHasHooks
	default:
		return MergeStatusUnknown
	}
}

// countUnresolvedThreads counts unresolved threads and formats the display string.
// Returns the numeric count and the formatted string.
func countUnresolvedThreads(threads github.ReviewThreadConnection) (int, string) {
	totalCount := threads.TotalCount
	nodes := threads.Nodes

	// Count unresolved threads from the sampled nodes
	unresolved := 0
	for _, thread := range nodes {
		if !thread.IsResolved {
			unresolved++
		}
	}

	// If totalCount > len(nodes), we have sampling truncation
	// Append "+" to indicate more threads exist
	if totalCount > len(nodes) {
		// Special case: show "0+" when sampled shows 0 but there are more
		return unresolved, strconv.Itoa(unresolved) + "+"
	}

	return unresolved, strconv.Itoa(unresolved)
}

// flattenReviewers flattens review requests into a string slice.
// Teams are prefixed with "team:".
func flattenReviewers(requests github.ReviewRequestConnection) []string {
	if len(requests.Nodes) == 0 {
		return nil
	}

	reviewers := make([]string, 0, len(requests.Nodes))
	for _, node := range requests.Nodes {
		reviewer := node.RequestedReviewer
		if reviewer.IsUser() {
			login := reviewer.GetLogin()
			if login != "" {
				reviewers = append(reviewers, login)
			}
		} else if reviewer.IsTeam() {
			name := reviewer.GetTeamName()
			if name != "" {
				reviewers = append(reviewers, "team:"+name)
			}
		}
	}
	return reviewers
}
