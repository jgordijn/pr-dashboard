package model

import (
	"sort"
	"strconv"
	"strings"
)

// SortField identifies the pull-request property used for ordering.
type SortField string

const (
	SortFieldName  SortField = "name"
	SortFieldAge   SortField = "age"
	SortFieldState SortField = "state"
)

// ParseSortField parses a persisted or user-facing sort field. Unknown values
// safely fall back to name.
func ParseSortField(value string) SortField {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(SortFieldAge):
		return SortFieldAge
	case string(SortFieldState):
		return SortFieldState
	default:
		return SortFieldName
	}
}

func (f SortField) String() string {
	switch f {
	case SortFieldAge, SortFieldState:
		return string(f)
	default:
		return string(SortFieldName)
	}
}

// Cycle returns the next field in name, age, state order.
func (f SortField) Cycle() SortField {
	switch ParseSortField(string(f)) {
	case SortFieldName:
		return SortFieldAge
	case SortFieldAge:
		return SortFieldState
	default:
		return SortFieldName
	}
}

// SortDirection controls the primary comparison direction.
type SortDirection string

const (
	SortAscending  SortDirection = "ascending"
	SortDescending SortDirection = "descending"
)

// ParseSortDirection parses long and compact direction names. Unknown values
// safely fall back to ascending.
func ParseSortDirection(value string) SortDirection {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "descending", "desc":
		return SortDescending
	default:
		return SortAscending
	}
}

func (d SortDirection) String() string {
	if d == SortDescending {
		return string(SortDescending)
	}
	return string(SortAscending)
}

// Toggle returns the opposite normalized direction.
func (d SortDirection) Toggle() SortDirection {
	if ParseSortDirection(string(d)) == SortDescending {
		return SortAscending
	}
	return SortDescending
}

// SortPullRequests returns an independently sorted copy. Direction affects only
// the selected field's primary comparison; all detail and identity tie-breakers
// remain ascending so equal values have a stable deterministic order.
func SortPullRequests(prs []PullRequest, field SortField, direction SortDirection) []PullRequest {
	if prs == nil {
		return nil
	}
	result := make([]PullRequest, len(prs))
	copy(result, prs)
	field = ParseSortField(string(field))
	direction = ParseSortDirection(string(direction))
	sort.Slice(result, func(i, j int) bool {
		comparison := comparePrimary(result[i], result[j], field, direction)
		if comparison != 0 {
			return comparison < 0
		}
		return compareIdentity(result[i], result[j]) < 0
	})
	return result
}

func comparePrimary(left, right PullRequest, field SortField, direction SortDirection) int {
	var primary, detail int
	switch field {
	case SortFieldAge:
		primary, detail = compareAge(left, right)
	case SortFieldState:
		primary, detail = compareState(left, right)
	default:
		primary = compareString(strings.ToLower(left.Title), strings.ToLower(right.Title))
		detail = compareString(left.Title, right.Title)
	}
	if primary != 0 && direction == SortDescending {
		primary = -primary
	}
	if primary != 0 {
		return primary
	}
	return detail
}

// compareAge keeps less precise fallbacks after exact timestamps and unknown
// ages last in either direction. Within an exact timestamp, newer means younger.
func compareAge(left, right PullRequest) (primary, detail int) {
	leftKind, rightKind := ageKind(left), ageKind(right)
	if leftKind != rightKind {
		return 0, compareInt(leftKind, rightKind)
	}
	switch leftKind {
	case 0:
		if left.CreatedAt.Equal(right.CreatedAt) {
			return 0, 0
		}
		if left.CreatedAt.After(right.CreatedAt) {
			return -1, 0
		}
		return 1, 0
	case 1:
		return compareInt(left.DaysOpen, right.DaysOpen), 0
	default:
		return 0, 0
	}
}

func ageKind(pr PullRequest) int {
	if !pr.CreatedAt.IsZero() {
		return 0
	}
	if pr.DaysOpen > 0 {
		return 1
	}
	return 2
}

func compareState(left, right PullRequest) (primary, detail int) {
	if comparison := compareInt(stateSeverity(left), stateSeverity(right)); comparison != 0 {
		return comparison, 0
	}
	leftDetail := stateDetail(left)
	rightDetail := stateDetail(right)
	for i := range leftDetail {
		if comparison := compareInt(leftDetail[i], rightDetail[i]); comparison != 0 {
			return 0, comparison
		}
	}
	return 0, 0
}

// State severity is ordered healthy, draft, unknown, attention, blocked,
// critical. A PR's worst applicable condition wins.
func stateSeverity(pr PullRequest) int {
	severity := 0
	for _, rank := range []int{mergeSeverity(pr.MergeStatus), checkSeverity(pr.CheckStatus), reviewSeverity(pr.ReviewStatus)} {
		if rank > severity {
			severity = rank
		}
	}
	if hasUnresolvedThreads(pr) && severity < 3 {
		severity = 3
	}
	if pr.IsDraft && severity < 1 {
		severity = 1
	}
	return severity
}

func stateDetail(pr PullRequest) [5]int {
	return [5]int{
		mergeSeverity(pr.MergeStatus),
		checkSeverity(pr.CheckStatus),
		reviewSeverity(pr.ReviewStatus),
		boolRank(hasUnresolvedThreads(pr)),
		unresolvedCount(pr),
	}
}

func mergeSeverity(status MergeStatus) int {
	switch status {
	case MergeStatusConflicts, MergeStatusDirty:
		return 5
	case MergeStatusBlocked:
		return 4
	case MergeStatusBehind, MergeStatusUnstable, MergeStatusHasHooks:
		return 3
	case MergeStatusUnknown:
		return 2
	case MergeStatusDraft:
		return 1
	default:
		return 0
	}
}

func checkSeverity(status CheckStatus) int {
	switch status {
	case CheckStatusFailing:
		return 5
	case CheckStatusPending:
		return 3
	case CheckStatusNone:
		return 2
	default:
		return 0
	}
}

func reviewSeverity(status ReviewStatus) int {
	switch status {
	case ReviewStatusChangesRequested:
		return 5
	case ReviewStatusReviewRequired:
		return 3
	case ReviewStatusNone:
		return 2
	default:
		return 0
	}
}

func hasUnresolvedThreads(pr PullRequest) bool { return unresolvedCount(pr) > 0 }

func unresolvedCount(pr PullRequest) int {
	if pr.UnresolvedCount > 0 {
		return pr.UnresolvedCount
	}
	count, _ := strconv.Atoi(strings.TrimSuffix(pr.UnresolvedThreads, "+"))
	return count
}

func compareIdentity(left, right PullRequest) int {
	pairs := [][2]string{
		{strings.ToLower(left.Key), strings.ToLower(right.Key)},
		{left.Key, right.Key},
		{strings.ToLower(left.Organization), strings.ToLower(right.Organization)},
		{left.Organization, right.Organization},
		{strings.ToLower(left.Repository), strings.ToLower(right.Repository)},
		{left.Repository, right.Repository},
	}
	for _, pair := range pairs {
		if comparison := compareString(pair[0], pair[1]); comparison != 0 {
			return comparison
		}
	}
	if comparison := compareInt(left.Number, right.Number); comparison != 0 {
		return comparison
	}
	if comparison := compareString(strings.ToLower(left.Title), strings.ToLower(right.Title)); comparison != 0 {
		return comparison
	}
	return compareString(left.Title, right.Title)
}

func compareString(left, right string) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func compareInt(left, right int) int {
	if left < right {
		return -1
	}
	if left > right {
		return 1
	}
	return 0
}

func boolRank(value bool) int {
	if value {
		return 1
	}
	return 0
}
