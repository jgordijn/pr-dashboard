package model

import (
	"testing"
	"time"
)

func TestPRGroup_Count(t *testing.T) {
	group := PRGroup{
		Organization: "test-org",
		PRs: []PullRequest{
			{Key: "test/repo#1"},
			{Key: "test/repo#2"},
			{Key: "test/repo#3"},
		},
	}

	if got := group.Count(); got != 3 {
		t.Errorf("PRGroup.Count() = %d, want 3", got)
	}
}

func TestPRGroup_CountVisible(t *testing.T) {
	group := PRGroup{
		Organization: "test-org",
		PRs: []PullRequest{
			{Key: "test/repo#1", IsDraft: false},
			{Key: "test/repo#2", IsDraft: true},
			{Key: "test/repo#3", IsDraft: false},
		},
	}

	tests := []struct {
		name       string
		showDrafts bool
		want       int
	}{
		{"show drafts", true, 3},
		{"hide drafts", false, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := group.CountVisible(tt.showDrafts); got != tt.want {
				t.Errorf("PRGroup.CountVisible(%v) = %d, want %d", tt.showDrafts, got, tt.want)
			}
		})
	}
}

func TestGroupByOrganization(t *testing.T) {
	now := time.Now()
	prs := []PullRequest{
		{Key: "org-a/repo#1", Organization: "org-a", UpdatedAt: now.Add(-1 * time.Hour)},
		{Key: "org-b/repo#1", Organization: "org-b", UpdatedAt: now.Add(-2 * time.Hour)},
		{Key: "org-a/repo#2", Organization: "org-a", UpdatedAt: now},
		{Key: "org-b/repo#2", Organization: "org-b", UpdatedAt: now.Add(-3 * time.Hour)},
	}

	groups := GroupByOrganization(prs)

	if len(groups) != 2 {
		t.Fatalf("GroupByOrganization() returned %d groups, want 2", len(groups))
	}

	// Groups should be sorted alphabetically
	if groups[0].Organization != "org-a" {
		t.Errorf("First group org = %s, want org-a", groups[0].Organization)
	}
	if groups[1].Organization != "org-b" {
		t.Errorf("Second group org = %s, want org-b", groups[1].Organization)
	}

	// PRs within groups should be sorted by updatedAt descending
	if len(groups[0].PRs) != 2 {
		t.Errorf("org-a has %d PRs, want 2", len(groups[0].PRs))
	}
	// org-a/repo#2 should be first (most recently updated)
	if groups[0].PRs[0].Key != "org-a/repo#2" {
		t.Errorf("First PR in org-a = %s, want org-a/repo#2", groups[0].PRs[0].Key)
	}
}

func TestGroupByOrganization_Empty(t *testing.T) {
	groups := GroupByOrganization(nil)
	if groups != nil {
		t.Errorf("GroupByOrganization(nil) = %v, want nil", groups)
	}

	groups = GroupByOrganization([]PullRequest{})
	if groups != nil {
		t.Errorf("GroupByOrganization([]) = %v, want nil", groups)
	}
}

func TestTotalPRCount(t *testing.T) {
	groups := []PRGroup{
		{PRs: []PullRequest{{Key: "a"}, {Key: "b"}}},
		{PRs: []PullRequest{{Key: "c"}}},
		{PRs: []PullRequest{}},
	}

	if got := TotalPRCount(groups); got != 3 {
		t.Errorf("TotalPRCount() = %d, want 3", got)
	}

	if got := TotalPRCount(nil); got != 0 {
		t.Errorf("TotalPRCount(nil) = %d, want 0", got)
	}
}

func TestTotalVisibleCount(t *testing.T) {
	groups := []PRGroup{
		{PRs: []PullRequest{
			{Key: "a", IsDraft: false},
			{Key: "b", IsDraft: true},
		}},
		{PRs: []PullRequest{
			{Key: "c", IsDraft: false},
		}},
	}

	if got := TotalVisibleCount(groups, true); got != 3 {
		t.Errorf("TotalVisibleCount(showDrafts=true) = %d, want 3", got)
	}

	if got := TotalVisibleCount(groups, false); got != 2 {
		t.Errorf("TotalVisibleCount(showDrafts=false) = %d, want 2", got)
	}
}

func TestFindPRByKey(t *testing.T) {
	groups := []PRGroup{
		{PRs: []PullRequest{{Key: "org/repo#1", Title: "PR 1"}}},
		{PRs: []PullRequest{{Key: "org/repo#2", Title: "PR 2"}}},
	}

	// Found
	if pr := FindPRByKey(groups, "org/repo#2"); pr == nil || pr.Title != "PR 2" {
		t.Errorf("FindPRByKey(org/repo#2) = %v, want PR 2", pr)
	}

	// Not found
	if pr := FindPRByKey(groups, "org/repo#99"); pr != nil {
		t.Errorf("FindPRByKey(org/repo#99) = %v, want nil", pr)
	}

	// Empty groups
	if pr := FindPRByKey(nil, "org/repo#1"); pr != nil {
		t.Errorf("FindPRByKey(nil, ...) = %v, want nil", pr)
	}
}

func TestGetVisiblePRs(t *testing.T) {
	groups := []PRGroup{
		{
			Organization: "org-a",
			Collapsed:    false,
			PRs: []PullRequest{
				{Key: "org-a/repo#1", IsDraft: false},
				{Key: "org-a/repo#2", IsDraft: true},
			},
		},
		{
			Organization: "org-b",
			Collapsed:    true,
			PRs: []PullRequest{
				{Key: "org-b/repo#1", IsDraft: false},
			},
		},
		{
			Organization: "org-c",
			Collapsed:    false,
			PRs: []PullRequest{
				{Key: "org-c/repo#1", IsDraft: false},
			},
		},
	}

	tests := []struct {
		name       string
		showDrafts bool
		wantKeys   []string
	}{
		{
			name:       "show drafts",
			showDrafts: true,
			wantKeys:   []string{"org-a/repo#1", "org-a/repo#2", "org-c/repo#1"},
		},
		{
			name:       "hide drafts",
			showDrafts: false,
			wantKeys:   []string{"org-a/repo#1", "org-c/repo#1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visible := GetVisiblePRs(groups, tt.showDrafts)
			if len(visible) != len(tt.wantKeys) {
				t.Fatalf("GetVisiblePRs() returned %d PRs, want %d", len(visible), len(tt.wantKeys))
			}
			for i, pr := range visible {
				if pr.Key != tt.wantKeys[i] {
					t.Errorf("visible[%d].Key = %s, want %s", i, pr.Key, tt.wantKeys[i])
				}
			}
		})
	}
}

func TestFindNearestVisiblePR(t *testing.T) {
	groups := []PRGroup{
		{
			Organization: "org-a",
			Collapsed:    false,
			PRs: []PullRequest{
				{Key: "org-a/repo#1", IsDraft: false},
				{Key: "org-a/repo#2", IsDraft: true},
			},
		},
	}

	tests := []struct {
		name       string
		key        string
		showDrafts bool
		want       string
	}{
		{"found and visible", "org-a/repo#1", true, "org-a/repo#1"},
		{"draft visible", "org-a/repo#2", true, "org-a/repo#2"},
		{"draft hidden, fallback to first", "org-a/repo#2", false, "org-a/repo#1"},
		{"not found, fallback to first", "org-a/repo#99", true, "org-a/repo#1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FindNearestVisiblePR(groups, tt.key, tt.showDrafts)
			if got != tt.want {
				t.Errorf("FindNearestVisiblePR() = %s, want %s", got, tt.want)
			}
		})
	}

	// Empty groups
	if got := FindNearestVisiblePR(nil, "any", true); got != "" {
		t.Errorf("FindNearestVisiblePR(nil) = %s, want empty", got)
	}
}

func TestAllKeys(t *testing.T) {
	groups := []PRGroup{
		{PRs: []PullRequest{{Key: "a"}, {Key: "b"}}},
		{PRs: []PullRequest{{Key: "c"}}},
	}

	keys := AllKeys(groups)
	if len(keys) != 3 {
		t.Fatalf("AllKeys() returned %d keys, want 3", len(keys))
	}

	expected := []string{"a", "b", "c"}
	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("keys[%d] = %s, want %s", i, key, expected[i])
		}
	}

	// Empty
	if keys := AllKeys(nil); keys != nil {
		t.Errorf("AllKeys(nil) = %v, want nil", keys)
	}
}

func TestNewPRList(t *testing.T) {
	list := NewPRList()
	if list == nil {
		t.Fatal("NewPRList() returned nil")
	}
	if list.Groups == nil {
		t.Error("NewPRList().Groups is nil")
	}
	if len(list.Groups) != 0 {
		t.Errorf("NewPRList().Groups has %d elements, want 0", len(list.Groups))
	}
	if list.TotalCount != 0 {
		t.Errorf("NewPRList().TotalCount = %d, want 0", list.TotalCount)
	}
}
