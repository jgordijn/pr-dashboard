package model

import (
	"testing"
	"time"

	"github.com/jgordijn/pr-dashboard/internal/github"
)

func ptr(s string) *string {
	return &s
}

func TestTransformPR_BasicFields(t *testing.T) {
	now := time.Now()
	created := now.Add(-48 * time.Hour) // 2 days ago

	apiPR := github.PullRequestNode{
		Typename:  "PullRequest",
		ID:        "PR_123",
		Number:    42,
		Title:     "Test PR",
		URL:       "https://github.com/org/repo/pull/42",
		IsDraft:   true,
		CreatedAt: created,
		UpdatedAt: now,
		Repository: github.Repository{
			Name: "repo",
			Owner: struct {
				Login string `json:"login"`
			}{Login: "org"},
		},
		Author: &github.Actor{Login: "author"},
	}

	pr := TransformPR(apiPR)

	if pr.Key != "org/repo#42" {
		t.Errorf("Key = %s, want org/repo#42", pr.Key)
	}
	if pr.Title != "Test PR" {
		t.Errorf("Title = %s, want Test PR", pr.Title)
	}
	if pr.Author != "author" {
		t.Errorf("Author = %s, want author", pr.Author)
	}
	if pr.Number != 42 {
		t.Errorf("Number = %d, want 42", pr.Number)
	}
	if pr.Organization != "org" {
		t.Errorf("Organization = %s, want org", pr.Organization)
	}
	if pr.Repository != "repo" {
		t.Errorf("Repository = %s, want repo", pr.Repository)
	}
	if pr.URL != "https://github.com/org/repo/pull/42" {
		t.Errorf("URL = %s, want https://github.com/org/repo/pull/42", pr.URL)
	}
	if !pr.IsDraft {
		t.Error("IsDraft = false, want true")
	}
	if pr.DaysOpen != 2 {
		t.Errorf("DaysOpen = %d, want 2", pr.DaysOpen)
	}
}

func TestTransformPR_NilAuthor(t *testing.T) {
	apiPR := github.PullRequestNode{
		Typename: "PullRequest",
		Author:   nil,
		Repository: github.Repository{
			Name: "repo",
			Owner: struct {
				Login string `json:"login"`
			}{Login: "org"},
		},
	}

	pr := TransformPR(apiPR)
	if pr.Author != "" {
		t.Errorf("Author = %s, want empty", pr.Author)
	}
}

func TestTransformPR_ReviewStatus(t *testing.T) {
	tests := []struct {
		name     string
		decision *string
		want     ReviewStatus
	}{
		{"approved", ptr("APPROVED"), ReviewStatusApproved},
		{"changes requested", ptr("CHANGES_REQUESTED"), ReviewStatusChangesRequested},
		{"review required", ptr("REVIEW_REQUIRED"), ReviewStatusReviewRequired},
		{"nil decision", nil, ReviewStatusNone},
		{"unknown value", ptr("UNKNOWN"), ReviewStatusNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiPR := github.PullRequestNode{
				Typename:       "PullRequest",
				ReviewDecision: tt.decision,
				Repository: github.Repository{
					Name: "repo",
					Owner: struct {
						Login string `json:"login"`
					}{Login: "org"},
				},
			}
			pr := TransformPR(apiPR)
			if pr.ReviewStatus != tt.want {
				t.Errorf("ReviewStatus = %v, want %v", pr.ReviewStatus, tt.want)
			}
		})
	}
}

func TestTransformPR_CheckStatus(t *testing.T) {
	tests := []struct {
		name  string
		state *string
		want  CheckStatus
	}{
		{"success", ptr("SUCCESS"), CheckStatusPassing},
		{"failure", ptr("FAILURE"), CheckStatusFailing},
		{"error", ptr("ERROR"), CheckStatusFailing},
		{"pending", ptr("PENDING"), CheckStatusPending},
		{"nil state", nil, CheckStatusNone},
		{"unknown value", ptr("UNKNOWN"), CheckStatusNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var rollup *github.StatusCheckRollup
			if tt.state != nil || tt.name == "nil state" {
				rollup = &github.StatusCheckRollup{State: tt.state}
			}

			apiPR := github.PullRequestNode{
				Typename: "PullRequest",
				Commits: github.CommitConnection{
					Nodes: []github.CommitNode{
						{Commit: github.Commit{StatusCheckRollup: rollup}},
					},
				},
				Repository: github.Repository{
					Name: "repo",
					Owner: struct {
						Login string `json:"login"`
					}{Login: "org"},
				},
			}
			pr := TransformPR(apiPR)
			if pr.CheckStatus != tt.want {
				t.Errorf("CheckStatus = %v, want %v", pr.CheckStatus, tt.want)
			}
		})
	}
}

func TestTransformPR_CheckStatusNoCommits(t *testing.T) {
	apiPR := github.PullRequestNode{
		Typename: "PullRequest",
		Commits:  github.CommitConnection{Nodes: nil},
		Repository: github.Repository{
			Name: "repo",
			Owner: struct {
				Login string `json:"login"`
			}{Login: "org"},
		},
	}
	pr := TransformPR(apiPR)
	if pr.CheckStatus != CheckStatusNone {
		t.Errorf("CheckStatus = %v, want None", pr.CheckStatus)
	}
}

func TestTransformPR_MergeStatus(t *testing.T) {
	tests := []struct {
		name             string
		isDraft          bool
		mergeable        *string
		mergeStateStatus *string
		want             MergeStatus
	}{
		{"draft overrides clean", true, ptr("MERGEABLE"), ptr("CLEAN"), MergeStatusDraft},
		{"conflicts remain visible on draft", true, ptr("CONFLICTING"), ptr("DIRTY"), MergeStatusConflicts},
		{"clean", false, ptr("MERGEABLE"), ptr("CLEAN"), MergeStatusReady},
		{"behind", false, ptr("MERGEABLE"), ptr("BEHIND"), MergeStatusBehind},
		{"blocked", false, ptr("MERGEABLE"), ptr("BLOCKED"), MergeStatusBlocked},
		{"dirty", false, ptr("MERGEABLE"), ptr("DIRTY"), MergeStatusDirty},
		{"unstable", false, ptr("MERGEABLE"), ptr("UNSTABLE"), MergeStatusUnstable},
		{"has hooks", false, ptr("MERGEABLE"), ptr("HAS_HOOKS"), MergeStatusHasHooks},
		{"conflicts", false, ptr("CONFLICTING"), nil, MergeStatusConflicts},
		{"unknown mergeable", false, ptr("UNKNOWN"), nil, MergeStatusUnknown},
		{"nil mergeable", false, nil, nil, MergeStatusUnknown},
		{"nil state status", false, ptr("MERGEABLE"), nil, MergeStatusUnknown},
		{"unexpected state", false, ptr("MERGEABLE"), ptr("WEIRD"), MergeStatusUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiPR := github.PullRequestNode{
				Typename:         "PullRequest",
				IsDraft:          tt.isDraft,
				Mergeable:        tt.mergeable,
				MergeStateStatus: tt.mergeStateStatus,
				Repository: github.Repository{
					Name: "repo",
					Owner: struct {
						Login string `json:"login"`
					}{Login: "org"},
				},
			}
			pr := TransformPR(apiPR)
			if pr.MergeStatus != tt.want {
				t.Errorf("MergeStatus = %v, want %v", pr.MergeStatus, tt.want)
			}
		})
	}
}

func TestTransformPR_UnresolvedThreads(t *testing.T) {
	tests := []struct {
		name           string
		totalCount     int
		threads        []github.ReviewThread
		wantCount      int
		wantDisplay    string
		wantPlusSuffix bool
	}{
		{
			name:        "no threads",
			totalCount:  0,
			threads:     nil,
			wantCount:   0,
			wantDisplay: "0",
		},
		{
			name:       "all resolved",
			totalCount: 3,
			threads: []github.ReviewThread{
				{IsResolved: true},
				{IsResolved: true},
				{IsResolved: true},
			},
			wantCount:   0,
			wantDisplay: "0",
		},
		{
			name:       "some unresolved",
			totalCount: 3,
			threads: []github.ReviewThread{
				{IsResolved: false},
				{IsResolved: true},
				{IsResolved: false},
			},
			wantCount:   2,
			wantDisplay: "2",
		},
		{
			name:       "truncated - 0+ case",
			totalCount: 150,
			threads: []github.ReviewThread{
				{IsResolved: true},
			},
			wantCount:      0,
			wantDisplay:    "0+",
			wantPlusSuffix: true,
		},
		{
			name:       "truncated - 5+ case",
			totalCount: 150,
			threads: []github.ReviewThread{
				{IsResolved: false},
				{IsResolved: false},
				{IsResolved: false},
				{IsResolved: true},
				{IsResolved: false},
				{IsResolved: false},
			},
			wantCount:      5,
			wantDisplay:    "5+",
			wantPlusSuffix: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiPR := github.PullRequestNode{
				Typename: "PullRequest",
				ReviewThreads: github.ReviewThreadConnection{
					TotalCount: tt.totalCount,
					Nodes:      tt.threads,
				},
				Repository: github.Repository{
					Name: "repo",
					Owner: struct {
						Login string `json:"login"`
					}{Login: "org"},
				},
			}
			pr := TransformPR(apiPR)

			if pr.UnresolvedCount != tt.wantCount {
				t.Errorf("UnresolvedCount = %d, want %d", pr.UnresolvedCount, tt.wantCount)
			}
			if pr.UnresolvedThreads != tt.wantDisplay {
				t.Errorf("UnresolvedThreads = %s, want %s", pr.UnresolvedThreads, tt.wantDisplay)
			}
		})
	}
}

func TestTransformPR_Reviewers(t *testing.T) {
	apiPR := github.PullRequestNode{
		Typename: "PullRequest",
		ReviewRequests: github.ReviewRequestConnection{
			Nodes: []github.ReviewRequest{
				{RequestedReviewer: github.RequestedReviewer{
					Typename: "User",
					Login:    ptr("user1"),
				}},
				{RequestedReviewer: github.RequestedReviewer{
					Typename: "Team",
					Name:     ptr("reviewers"),
				}},
				{RequestedReviewer: github.RequestedReviewer{
					Typename: "User",
					Login:    ptr("user2"),
				}},
			},
		},
		Repository: github.Repository{
			Name: "repo",
			Owner: struct {
				Login string `json:"login"`
			}{Login: "org"},
		},
	}

	pr := TransformPR(apiPR)

	if len(pr.Reviewers) != 3 {
		t.Fatalf("len(Reviewers) = %d, want 3", len(pr.Reviewers))
	}

	expected := []string{"user1", "team:reviewers", "user2"}
	for i, r := range pr.Reviewers {
		if r != expected[i] {
			t.Errorf("Reviewers[%d] = %s, want %s", i, r, expected[i])
		}
	}
}

func TestTransformPR_ReviewersEmpty(t *testing.T) {
	apiPR := github.PullRequestNode{
		Typename:       "PullRequest",
		ReviewRequests: github.ReviewRequestConnection{Nodes: nil},
		Repository: github.Repository{
			Name: "repo",
			Owner: struct {
				Login string `json:"login"`
			}{Login: "org"},
		},
	}

	pr := TransformPR(apiPR)
	if pr.Reviewers != nil {
		t.Errorf("Reviewers = %v, want nil", pr.Reviewers)
	}
}

func TestTransformPRs(t *testing.T) {
	apiPRs := []github.PullRequestNode{
		{
			Typename: "PullRequest",
			Number:   1,
			Title:    "PR 1",
			Repository: github.Repository{
				Name: "repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "org"},
			},
		},
		{
			Typename: "Issue", // Should be skipped
			Number:   2,
		},
		{
			Typename: "PullRequest",
			Number:   3,
			Title:    "PR 3",
			Repository: github.Repository{
				Name: "repo",
				Owner: struct {
					Login string `json:"login"`
				}{Login: "org"},
			},
		},
	}

	prs := TransformPRs(apiPRs)

	if len(prs) != 2 {
		t.Fatalf("len(prs) = %d, want 2", len(prs))
	}

	if prs[0].Number != 1 {
		t.Errorf("prs[0].Number = %d, want 1", prs[0].Number)
	}
	if prs[1].Number != 3 {
		t.Errorf("prs[1].Number = %d, want 3", prs[1].Number)
	}
}

func TestTransformPRs_Empty(t *testing.T) {
	if prs := TransformPRs(nil); prs != nil {
		t.Errorf("TransformPRs(nil) = %v, want nil", prs)
	}
	if prs := TransformPRs([]github.PullRequestNode{}); prs != nil {
		t.Errorf("TransformPRs([]) = %v, want nil", prs)
	}
}

func TestCalculateDaysOpen(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name    string
		created time.Time
		want    int
	}{
		{"just created", now, 0},
		{"1 day ago", now.Add(-24 * time.Hour), 1},
		{"2.5 days ago", now.Add(-60 * time.Hour), 2},
		{"10 days ago", now.Add(-240 * time.Hour), 10},
		{"future date (edge case)", now.Add(24 * time.Hour), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculateDaysOpen(tt.created, now)
			if got != tt.want {
				t.Errorf("calculateDaysOpen() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestTransformPR_RawMergeValues(t *testing.T) {
	apiPR := github.PullRequestNode{
		Typename:         "PullRequest",
		Mergeable:        ptr("MERGEABLE"),
		MergeStateStatus: ptr("BEHIND"),
		Repository: github.Repository{
			Name: "repo",
			Owner: struct {
				Login string `json:"login"`
			}{Login: "org"},
		},
	}

	pr := TransformPR(apiPR)

	if pr.Mergeable != "MERGEABLE" {
		t.Errorf("Mergeable = %s, want MERGEABLE", pr.Mergeable)
	}
	if pr.MergeStateStatus != "BEHIND" {
		t.Errorf("MergeStateStatus = %s, want BEHIND", pr.MergeStateStatus)
	}
}

func TestTransformPR_NilMergeValues(t *testing.T) {
	apiPR := github.PullRequestNode{
		Typename:         "PullRequest",
		Mergeable:        nil,
		MergeStateStatus: nil,
		Repository: github.Repository{
			Name: "repo",
			Owner: struct {
				Login string `json:"login"`
			}{Login: "org"},
		},
	}

	pr := TransformPR(apiPR)

	if pr.Mergeable != "" {
		t.Errorf("Mergeable = %s, want empty", pr.Mergeable)
	}
	if pr.MergeStateStatus != "" {
		t.Errorf("MergeStateStatus = %s, want empty", pr.MergeStateStatus)
	}
}
