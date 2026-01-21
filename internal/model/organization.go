package model

import "sort"

// PRGroup represents a group of PRs belonging to an organization.
type PRGroup struct {
	// Organization is the GitHub org login.
	Organization string

	// PRs is the slice of pull requests in this group.
	PRs []PullRequest

	// Collapsed indicates if the group is collapsed in the UI.
	Collapsed bool
}

// Count returns the number of PRs in this group.
func (g *PRGroup) Count() int {
	return len(g.PRs)
}

// CountVisible returns the number of visible PRs (excluding drafts if hidden).
func (g *PRGroup) CountVisible(showDrafts bool) int {
	if showDrafts {
		return len(g.PRs)
	}
	count := 0
	for _, pr := range g.PRs {
		if !pr.IsDraft {
			count++
		}
	}
	return count
}

// PRList represents a collection of PRs grouped by organization.
type PRList struct {
	// Groups is the slice of PR groups.
	Groups []PRGroup

	// TotalCount is the total PR count across all groups.
	TotalCount int
}

// NewPRList creates a new empty PRList.
func NewPRList() *PRList {
	return &PRList{
		Groups: make([]PRGroup, 0),
	}
}

// GroupByOrganization groups pull requests by organization and sorts by updatedAt descending.
func GroupByOrganization(prs []PullRequest) []PRGroup {
	if len(prs) == 0 {
		return nil
	}

	// Group PRs by organization
	orgMap := make(map[string][]PullRequest)
	orgOrder := make([]string, 0)

	for _, pr := range prs {
		if _, exists := orgMap[pr.Organization]; !exists {
			orgOrder = append(orgOrder, pr.Organization)
		}
		orgMap[pr.Organization] = append(orgMap[pr.Organization], pr)
	}

	// Sort organizations alphabetically
	sort.Strings(orgOrder)

	// Create groups with sorted PRs
	groups := make([]PRGroup, 0, len(orgOrder))
	for _, org := range orgOrder {
		orgPRs := orgMap[org]
		// Sort PRs by updatedAt descending
		sort.Slice(orgPRs, func(i, j int) bool {
			return orgPRs[i].UpdatedAt.After(orgPRs[j].UpdatedAt)
		})
		groups = append(groups, PRGroup{
			Organization: org,
			PRs:          orgPRs,
			Collapsed:    false,
		})
	}

	return groups
}

// TotalPRCount returns the total number of PRs across all groups.
func TotalPRCount(groups []PRGroup) int {
	total := 0
	for _, g := range groups {
		total += len(g.PRs)
	}
	return total
}

// TotalVisibleCount returns the count of visible PRs respecting draft visibility.
func TotalVisibleCount(groups []PRGroup, showDrafts bool) int {
	total := 0
	for _, g := range groups {
		total += g.CountVisible(showDrafts)
	}
	return total
}

// FindPRByKey finds a PR by its stable key across all groups.
// Returns nil if not found.
func FindPRByKey(groups []PRGroup, key string) *PullRequest {
	for _, g := range groups {
		for i := range g.PRs {
			if g.PRs[i].Key == key {
				return &g.PRs[i]
			}
		}
	}
	return nil
}

// GetVisiblePRs returns a flat list of visible PRs respecting collapse and draft settings.
func GetVisiblePRs(groups []PRGroup, showDrafts bool) []PullRequest {
	var result []PullRequest
	for _, g := range groups {
		if g.Collapsed {
			continue
		}
		for _, pr := range g.PRs {
			if showDrafts || !pr.IsDraft {
				result = append(result, pr)
			}
		}
	}
	return result
}

// FindNearestVisiblePR finds the nearest visible PR to the given key.
// Returns empty string if no visible PRs exist.
func FindNearestVisiblePR(groups []PRGroup, key string, showDrafts bool) string {
	visiblePRs := GetVisiblePRs(groups, showDrafts)
	if len(visiblePRs) == 0 {
		return ""
	}

	// Find the index of the current key if it exists and is visible
	currentIndex := -1
	for i, pr := range visiblePRs {
		if pr.Key == key {
			currentIndex = i
			break
		}
	}

	// If current key is visible, return it
	if currentIndex >= 0 {
		return key
	}

	// Otherwise, return the first visible PR
	return visiblePRs[0].Key
}

// AllKeys returns all PR keys in order (for iteration).
func AllKeys(groups []PRGroup) []string {
	var keys []string
	for _, g := range groups {
		for _, pr := range g.PRs {
			keys = append(keys, pr.Key)
		}
	}
	return keys
}
