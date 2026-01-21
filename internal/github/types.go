// Package github provides types and client for GitHub GraphQL API integration.
package github

import "time"

// GraphQLResponse represents the top-level response from GitHub GraphQL API.
type GraphQLResponse struct {
	Data   *ResponseData  `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

// GraphQLError represents an error returned by the GitHub GraphQL API.
type GraphQLError struct {
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

// ResponseData contains the data payload from a GraphQL response.
// Fields are pointers to distinguish "missing" from "present but empty".
type ResponseData struct {
	Search    *SearchResponse `json:"search,omitempty"`
	RateLimit *RateLimit      `json:"rateLimit,omitempty"`
}

// SearchResponse represents the search results from GitHub GraphQL API.
// Note: GitHub's search returns a union type (SearchResultItem), but our query
// constrains results to PRs using `is:pr` filter and requests __typename.
// Callers should filter nodes using IsPullRequest() for defensive validation.
type SearchResponse struct {
	Nodes    []PullRequestNode `json:"nodes"`
	PageInfo PageInfo          `json:"pageInfo"`
}

// PageInfo contains pagination information for cursor-based pagination.
type PageInfo struct {
	HasNextPage bool    `json:"hasNextPage"`
	EndCursor   *string `json:"endCursor"`
}

// PullRequestNode represents a pull request returned from the search query.
// Note: The search query uses inline fragments (... on PullRequest), which spread
// PR fields directly onto each node. The Typename field allows validation that
// the node is indeed a PullRequest (defensive check against query changes).
type PullRequestNode struct {
	// Typename is the GraphQL __typename field for type discrimination.
	// Expected value: "PullRequest". Used for defensive validation.
	Typename string `json:"__typename,omitempty"`

	ID        string    `json:"id"`
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	URL       string    `json:"url"`
	IsDraft   bool      `json:"isDraft"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// Mergeable is nullable in GitHub GraphQL schema. It returns UNKNOWN, MERGEABLE,
	// or CONFLICTING, but can be null while GitHub is computing the merge state.
	Mergeable *string `json:"mergeable"`
	// MergeStateStatus is nullable in GitHub GraphQL schema. It can be null when
	// Mergeable is null (i.e., when GitHub is still computing merge state).
	MergeStateStatus *string `json:"mergeStateStatus"`
	// ReviewDecision is nullable in GitHub GraphQL schema. It represents the review
	// decision state (APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED) or null if not applicable.
	ReviewDecision *string                 `json:"reviewDecision"`
	Repository     Repository              `json:"repository"`
	Author         *Actor                  `json:"author"`
	ReviewRequests ReviewRequestConnection `json:"reviewRequests"`
	Reviews        ReviewConnection        `json:"reviews"`
	ReviewThreads  ReviewThreadConnection  `json:"reviewThreads"`
	Commits        CommitConnection        `json:"commits"`
}

// IsPullRequest returns true if the node is confirmed to be a PullRequest type.
// Returns false if __typename is missing or not "PullRequest".
// This is a defensive check - callers should filter out non-PR nodes.
func (p PullRequestNode) IsPullRequest() bool {
	return p.Typename == "PullRequest"
}

// HasTypename returns true if the __typename field was populated in the response.
// Use this to detect if the query is requesting __typename as expected.
func (p PullRequestNode) HasTypename() bool {
	return p.Typename != ""
}

// Repository represents a GitHub repository.
type Repository struct {
	Owner RepositoryOwner `json:"owner"`
	Name  string          `json:"name"`
}

// RepositoryOwner represents the owner of a repository.
type RepositoryOwner struct {
	Login string `json:"login"`
}

// Actor represents a GitHub user or bot that performs actions.
type Actor struct {
	Login string `json:"login"`
}

// ReviewRequestConnection contains a list of review requests.
type ReviewRequestConnection struct {
	Nodes []ReviewRequest `json:"nodes"`
}

// ReviewRequest represents a request for review on a pull request.
type ReviewRequest struct {
	RequestedReviewer RequestedReviewer `json:"requestedReviewer"`
}

// RequestedReviewer represents the reviewer, which can be either a User or Team.
// This handles the GraphQL union type by flattening fields from both types.
// Use Typename to determine which type of reviewer this is.
//
// GraphQL inline fragments spread fields directly onto the parent object:
//
//	requestedReviewer {
//	  ... on User { login }
//	  ... on Team { name slug }
//	}
type RequestedReviewer struct {
	// Typename is the GraphQL __typename discriminator ("User" or "Team").
	Typename string `json:"__typename"`

	// Login is populated when Typename == "User"
	Login *string `json:"login,omitempty"`

	// Name and Slug are populated when Typename == "Team"
	Name *string `json:"name,omitempty"`
	Slug *string `json:"slug,omitempty"`
}

// IsUser returns true if the reviewer is a User.
func (r RequestedReviewer) IsUser() bool {
	return r.Typename == "User"
}

// IsTeam returns true if the reviewer is a Team.
func (r RequestedReviewer) IsTeam() bool {
	return r.Typename == "Team"
}

// GetLogin returns the login for User reviewers, or empty string for teams.
func (r RequestedReviewer) GetLogin() string {
	if r.Login != nil {
		return *r.Login
	}
	return ""
}

// GetTeamName returns the team name for Team reviewers, or empty string for users.
func (r RequestedReviewer) GetTeamName() string {
	if r.Name != nil {
		return *r.Name
	}
	return ""
}

// GetTeamSlug returns the team slug for Team reviewers, or empty string for users.
func (r RequestedReviewer) GetTeamSlug() string {
	if r.Slug != nil {
		return *r.Slug
	}
	return ""
}

// ReviewConnection contains a list of reviews.
type ReviewConnection struct {
	Nodes []Review `json:"nodes"`
}

// Review represents a review on a pull request.
type Review struct {
	Author *Actor `json:"author"`
	State  string `json:"state"`
}

// ReviewThreadConnection contains review threads and their count.
type ReviewThreadConnection struct {
	TotalCount int            `json:"totalCount"`
	Nodes      []ReviewThread `json:"nodes"`
}

// ReviewThread represents a review thread on a pull request.
type ReviewThread struct {
	IsResolved bool `json:"isResolved"`
}

// CommitConnection contains a list of commits.
type CommitConnection struct {
	Nodes []CommitNode `json:"nodes"`
}

// CommitNode represents a commit in a pull request.
type CommitNode struct {
	Commit Commit `json:"commit"`
}

// Commit represents a Git commit with its status information.
type Commit struct {
	StatusCheckRollup *StatusCheckRollup `json:"statusCheckRollup"`
}

// StatusCheckRollup contains the aggregated status of all checks.
type StatusCheckRollup struct {
	State    *string             `json:"state"`
	Contexts CheckStatusContexts `json:"contexts"`
}

// CheckStatusContexts contains the list of check contexts.
type CheckStatusContexts struct {
	Nodes []CheckStatusContext `json:"nodes"`
}

// CheckStatusContext represents either a CheckRun or StatusContext.
// This handles the GraphQL union type by flattening fields from both types.
// Use Typename to determine which type of context this is.
//
// GraphQL inline fragments spread fields directly onto the parent object:
//
//	contexts {
//	  ... on CheckRun { name conclusion status }
//	  ... on StatusContext { context state }
//	}
type CheckStatusContext struct {
	// Typename is the GraphQL __typename discriminator ("CheckRun" or "StatusContext").
	Typename string `json:"__typename"`

	// CheckRun fields (populated when Typename == "CheckRun")
	Name       *string `json:"name,omitempty"`
	Conclusion *string `json:"conclusion,omitempty"`
	Status     *string `json:"status,omitempty"`

	// StatusContext fields (populated when Typename == "StatusContext")
	Context *string `json:"context,omitempty"`
	State   *string `json:"state,omitempty"`
}

// IsCheckRun returns true if this is a CheckRun type.
func (c CheckStatusContext) IsCheckRun() bool {
	return c.Typename == "CheckRun"
}

// IsStatusContext returns true if this is a StatusContext type.
func (c CheckStatusContext) IsStatusContext() bool {
	return c.Typename == "StatusContext"
}

// GetCheckRun returns a CheckRun struct if this is a CheckRun, nil otherwise.
func (c CheckStatusContext) GetCheckRun() *CheckRun {
	if !c.IsCheckRun() {
		return nil
	}
	return &CheckRun{
		Name:       c.GetName(),
		Conclusion: c.Conclusion,
		Status:     c.GetStatus(),
	}
}

// GetStatusContext returns a StatusContext struct if this is a StatusContext, nil otherwise.
func (c CheckStatusContext) GetStatusContext() *StatusContext {
	if !c.IsStatusContext() {
		return nil
	}
	return &StatusContext{
		Context: c.GetContext(),
		State:   c.GetState(),
	}
}

// GetName returns the name for CheckRun contexts, or empty string for StatusContext.
func (c CheckStatusContext) GetName() string {
	if c.Name != nil {
		return *c.Name
	}
	return ""
}

// GetStatus returns the status for CheckRun contexts, or empty string for StatusContext.
func (c CheckStatusContext) GetStatus() string {
	if c.Status != nil {
		return *c.Status
	}
	return ""
}

// GetContext returns the context for StatusContext, or empty string for CheckRun.
func (c CheckStatusContext) GetContext() string {
	if c.Context != nil {
		return *c.Context
	}
	return ""
}

// GetState returns the state for StatusContext, or empty string for CheckRun.
func (c CheckStatusContext) GetState() string {
	if c.State != nil {
		return *c.State
	}
	return ""
}

// RateLimit represents GitHub API rate limit information.
type RateLimit struct {
	Remaining int       `json:"remaining"`
	ResetAt   time.Time `json:"resetAt"`
	Cost      int       `json:"cost"`
}

// CheckRun represents a GitHub Actions check run.
// This is a convenience type for domain model conversion.
type CheckRun struct {
	Name       string
	Conclusion *string
	Status     string
}

// StatusContext represents a commit status context (GitHub Commit Status API).
// This is a convenience type for domain model conversion.
type StatusContext struct {
	Context string
	State   string
}

// Team represents a GitHub team for team review requests.
// This is a convenience type for domain model conversion.
type Team struct {
	Name string
	Slug string
}

// User represents a GitHub user.
// This is a convenience type for domain model conversion.
type User struct {
	Login string
}
