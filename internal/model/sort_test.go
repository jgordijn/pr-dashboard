package model

import (
	"reflect"
	"testing"
	"time"
)

func TestSortEnums(t *testing.T) {
	fieldCases := []struct {
		input string
		want  SortField
	}{{"name", SortFieldName}, {" AGE ", SortFieldAge}, {"State", SortFieldState}, {"bad", SortFieldName}}
	for _, tc := range fieldCases {
		if got := ParseSortField(tc.input); got != tc.want {
			t.Fatalf("ParseSortField(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
	if SortFieldName.String() != "name" || SortFieldAge.String() != "age" || SortFieldState.String() != "state" || SortField("bad").String() != "name" {
		t.Fatal("unexpected sort field strings")
	}
	if SortFieldName.Cycle() != SortFieldAge || SortFieldAge.Cycle() != SortFieldState || SortFieldState.Cycle() != SortFieldName || SortField("bad").Cycle() != SortFieldAge {
		t.Fatal("unexpected sort field cycle")
	}

	directionCases := []struct {
		input string
		want  SortDirection
	}{{"ascending", SortAscending}, {" ASC ", SortAscending}, {"descending", SortDescending}, {"DESC", SortDescending}, {"bad", SortAscending}}
	for _, tc := range directionCases {
		if got := ParseSortDirection(tc.input); got != tc.want {
			t.Fatalf("ParseSortDirection(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
	if SortAscending.String() != "ascending" || SortDescending.String() != "descending" || SortDirection("bad").String() != "ascending" {
		t.Fatal("unexpected sort direction strings")
	}
	if SortAscending.Toggle() != SortDescending || SortDescending.Toggle() != SortAscending || SortDirection("bad").Toggle() != SortDescending {
		t.Fatal("unexpected direction toggle")
	}
}

func TestSortPullRequestsNameAndIdentity(t *testing.T) {
	input := []PullRequest{
		{Key: "z/repo#2", Title: "beta"},
		{Key: "b/repo#1", Title: "Alpha"},
		{Key: "a/repo#2", Title: "alpha"},
		{Key: "A/repo#1", Title: "alpha"},
	}
	before := append([]PullRequest(nil), input...)
	ascending := SortPullRequests(input, SortFieldName, SortAscending)
	assertKeys(t, ascending, "b/repo#1", "A/repo#1", "a/repo#2", "z/repo#2")
	descending := SortPullRequests(input, SortFieldName, SortDescending)
	assertKeys(t, descending, "z/repo#2", "b/repo#1", "A/repo#1", "a/repo#2")
	if !reflect.DeepEqual(input, before) {
		t.Fatalf("input mutated: %#v", input)
	}
	if got := SortPullRequests(nil, SortFieldName, SortAscending); got != nil {
		t.Fatalf("nil sort = %#v", got)
	}
	if got := SortPullRequests([]PullRequest{}, SortFieldName, SortAscending); got == nil || len(got) != 0 {
		t.Fatalf("empty sort = %#v", got)
	}
	assertKeys(t, SortPullRequests(input, SortField("bad"), SortDirection("bad")), "b/repo#1", "A/repo#1", "a/repo#2", "z/repo#2")
}

func TestSortPullRequestsAge(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	input := []PullRequest{
		{Key: "old", CreatedAt: now.Add(-72 * time.Hour)},
		{Key: "new", CreatedAt: now.Add(-time.Hour)},
		{Key: "fallback-old", DaysOpen: 8},
		{Key: "fallback-new", DaysOpen: 2},
		{Key: "unknown-b"},
		{Key: "unknown-a"},
	}
	assertKeys(t, SortPullRequests(input, SortFieldAge, SortAscending), "new", "old", "fallback-new", "fallback-old", "unknown-a", "unknown-b")
	assertKeys(t, SortPullRequests(input, SortFieldAge, SortDescending), "old", "new", "fallback-old", "fallback-new", "unknown-a", "unknown-b")

	ties := []PullRequest{{Key: "b", CreatedAt: now}, {Key: "a", CreatedAt: now}}
	assertKeys(t, SortPullRequests(ties, SortFieldAge, SortDescending), "a", "b")
}

func TestSortPullRequestsStateSeverityAndDetails(t *testing.T) {
	input := []PullRequest{
		{Key: "healthy", CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved, MergeStatus: MergeStatusReady},
		{Key: "draft", CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved, MergeStatus: MergeStatusDraft, IsDraft: true},
		{Key: "unknown", CheckStatus: CheckStatusNone, ReviewStatus: ReviewStatusNone, MergeStatus: MergeStatusUnknown},
		{Key: "threads", CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved, MergeStatus: MergeStatusReady, UnresolvedCount: 2},
		{Key: "behind", CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved, MergeStatus: MergeStatusBehind},
		{Key: "blocked", CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved, MergeStatus: MergeStatusBlocked},
		{Key: "critical-review", CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusChangesRequested, MergeStatus: MergeStatusReady},
		{Key: "critical-ci", CheckStatus: CheckStatusFailing, ReviewStatus: ReviewStatusApproved, MergeStatus: MergeStatusReady},
		{Key: "critical-merge", CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved, MergeStatus: MergeStatusConflicts},
	}
	ascending := SortPullRequests(input, SortFieldState, SortAscending)
	assertKeys(t, ascending, "healthy", "draft", "unknown", "threads", "behind", "blocked", "critical-review", "critical-ci", "critical-merge")
	descending := SortPullRequests(input, SortFieldState, SortDescending)
	// Severity reverses, while equal-severity details and identity remain ascending.
	assertKeys(t, descending, "critical-review", "critical-ci", "critical-merge", "blocked", "threads", "behind", "unknown", "draft", "healthy")
}

func TestComparatorTieBreakPaths(t *testing.T) {
	now := time.Date(2026, 1, 10, 0, 0, 0, 0, time.UTC)
	if primary, _ := compareAge(PullRequest{CreatedAt: now.Add(-time.Hour)}, PullRequest{CreatedAt: now}); primary != 1 {
		t.Fatalf("older exact age comparison = %d", primary)
	}
	equalState := PullRequest{MergeStatus: MergeStatusReady, CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved}
	if primary, detail := compareState(equalState, equalState); primary != 0 || detail != 0 {
		t.Fatalf("equal state = %d, %d", primary, detail)
	}
	if got := stateSeverity(PullRequest{IsDraft: true, MergeStatus: MergeStatusReady, CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved}); got != 1 {
		t.Fatalf("draft fallback severity = %d", got)
	}

	identityCases := []struct {
		left, right PullRequest
	}{
		{PullRequest{Key: "A"}, PullRequest{Key: "a"}},
		{PullRequest{Key: "same", Organization: "a"}, PullRequest{Key: "same", Organization: "b"}},
		{PullRequest{Key: "same", Organization: "A"}, PullRequest{Key: "same", Organization: "a"}},
		{PullRequest{Key: "same", Organization: "same", Repository: "a"}, PullRequest{Key: "same", Organization: "same", Repository: "b"}},
		{PullRequest{Key: "same", Organization: "same", Repository: "A"}, PullRequest{Key: "same", Organization: "same", Repository: "a"}},
		{PullRequest{Key: "same", Organization: "same", Repository: "same", Number: 1}, PullRequest{Key: "same", Organization: "same", Repository: "same", Number: 2}},
		{PullRequest{Key: "same", Organization: "same", Repository: "same", Number: 1, Title: "a"}, PullRequest{Key: "same", Organization: "same", Repository: "same", Number: 1, Title: "b"}},
		{PullRequest{Key: "same", Organization: "same", Repository: "same", Number: 1, Title: "A"}, PullRequest{Key: "same", Organization: "same", Repository: "same", Number: 1, Title: "a"}},
		{PullRequest{Key: "same"}, PullRequest{Key: "same"}},
	}
	for i, tc := range identityCases {
		got := compareIdentity(tc.left, tc.right)
		if i < len(identityCases)-1 && got >= 0 {
			t.Fatalf("identity case %d = %d", i, got)
		}
		if i == len(identityCases)-1 && got != 0 {
			t.Fatalf("equal identity = %d", got)
		}
	}
}

func TestStateSeverityAllPaths(t *testing.T) {
	cases := []struct {
		name string
		pr   PullRequest
		want int
	}{
		{"dirty", PullRequest{MergeStatus: MergeStatusDirty}, 5},
		{"conflict", PullRequest{MergeStatus: MergeStatusConflicts}, 5},
		{"failing", PullRequest{CheckStatus: CheckStatusFailing, MergeStatus: MergeStatusReady, ReviewStatus: ReviewStatusApproved}, 5},
		{"changes", PullRequest{ReviewStatus: ReviewStatusChangesRequested, MergeStatus: MergeStatusReady, CheckStatus: CheckStatusPassing}, 5},
		{"blocked", PullRequest{MergeStatus: MergeStatusBlocked}, 4},
		{"pending", PullRequest{CheckStatus: CheckStatusPending, MergeStatus: MergeStatusReady, ReviewStatus: ReviewStatusApproved}, 3},
		{"required", PullRequest{ReviewStatus: ReviewStatusReviewRequired, MergeStatus: MergeStatusReady, CheckStatus: CheckStatusPassing}, 3},
		{"behind", PullRequest{MergeStatus: MergeStatusBehind}, 3},
		{"unstable", PullRequest{MergeStatus: MergeStatusUnstable}, 3},
		{"hooks", PullRequest{MergeStatus: MergeStatusHasHooks}, 3},
		{"threads numeric", PullRequest{MergeStatus: MergeStatusReady, CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved, UnresolvedCount: 1}, 3},
		{"threads display", PullRequest{MergeStatus: MergeStatusReady, CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved, UnresolvedThreads: "2+"}, 3},
		{"unknown merge", PullRequest{MergeStatus: MergeStatusUnknown, CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved}, 2},
		{"unknown check", PullRequest{MergeStatus: MergeStatusReady, CheckStatus: CheckStatusNone, ReviewStatus: ReviewStatusApproved}, 2},
		{"unknown review", PullRequest{MergeStatus: MergeStatusReady, CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusNone}, 2},
		{"draft", PullRequest{MergeStatus: MergeStatusDraft, CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved, IsDraft: true}, 1},
		{"healthy", PullRequest{MergeStatus: MergeStatusReady, CheckStatus: CheckStatusPassing, ReviewStatus: ReviewStatusApproved}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := stateSeverity(tc.pr); got != tc.want {
				t.Fatalf("stateSeverity() = %d, want %d", got, tc.want)
			}
		})
	}
}

func assertKeys(t *testing.T, prs []PullRequest, want ...string) {
	t.Helper()
	got := make([]string, len(prs))
	for i := range prs {
		got[i] = prs[i].Key
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("keys = %v, want %v", got, want)
	}
}
