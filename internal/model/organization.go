package model

import (
	"sort"
	"strings"
)

// GroupingMode controls how pull requests are grouped in the TUI.
type GroupingMode int

const (
	// GroupingModeOrganization groups pull requests directly by organization.
	GroupingModeOrganization GroupingMode = iota
	// GroupingModeRepository groups pull requests by organization and repository.
	GroupingModeRepository
)

// String returns the configuration value for the grouping mode.
func (m GroupingMode) String() string {
	if m == GroupingModeRepository {
		return "repository"
	}
	return "organization"
}

// ParseGroupingMode converts a configuration value to a grouping mode.
// Unknown values safely fall back to organization grouping.
func ParseGroupingMode(value string) GroupingMode {
	if strings.ToLower(strings.TrimSpace(value)) == "repository" {
		return GroupingModeRepository
	}
	return GroupingModeOrganization
}

// RepositoryGroup represents pull requests belonging to one repository.
type RepositoryGroup struct {
	Organization string
	Repository   string
	PRs          []PullRequest
}

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

// GroupByRepository groups pull requests by their exact owner and repository.
// Groups are ordered case-insensitively by owner then repository, with raw
// names as deterministic tie-breakers. PRs are ordered by update time
// descending, then by stable key. The input slice is never mutated.
func GroupByRepository(prs []PullRequest) []RepositoryGroup {
	if len(prs) == 0 {
		return nil
	}

	type repositoryKey struct {
		organization string
		repository   string
	}

	grouped := make(map[repositoryKey][]PullRequest)
	for _, pr := range prs {
		key := repositoryKey{organization: pr.Organization, repository: pr.Repository}
		grouped[key] = append(grouped[key], pr)
	}

	groups := make([]RepositoryGroup, 0, len(grouped))
	for key, repositoryPRs := range grouped {
		sort.Slice(repositoryPRs, func(i, j int) bool {
			if repositoryPRs[i].UpdatedAt.Equal(repositoryPRs[j].UpdatedAt) {
				return repositoryPRs[i].Key < repositoryPRs[j].Key
			}
			return repositoryPRs[i].UpdatedAt.After(repositoryPRs[j].UpdatedAt)
		})
		groups = append(groups, RepositoryGroup{
			Organization: key.organization,
			Repository:   key.repository,
			PRs:          repositoryPRs,
		})
	}

	sort.Slice(groups, func(i, j int) bool {
		leftOwner, rightOwner := strings.ToLower(groups[i].Organization), strings.ToLower(groups[j].Organization)
		if leftOwner != rightOwner {
			return leftOwner < rightOwner
		}
		if groups[i].Organization != groups[j].Organization {
			return groups[i].Organization < groups[j].Organization
		}
		leftRepository, rightRepository := strings.ToLower(groups[i].Repository), strings.ToLower(groups[j].Repository)
		if leftRepository != rightRepository {
			return leftRepository < rightRepository
		}
		return groups[i].Repository < groups[j].Repository
	})

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
