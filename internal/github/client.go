// Package github provides types and client for GitHub GraphQL API integration.
package github

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
)

// SearchResult contains the results of a PR search query.
type SearchResult struct {
	PullRequests []PullRequestNode
	RateLimit    *RateLimit
	Warning      string
}

// Client provides methods to interact with GitHub's GraphQL API.
type Client struct {
	graphql GraphQLClient
}

// GraphQLClient abstracts the GraphQL client for testing.
type GraphQLClient interface {
	Do(query string, variables map[string]interface{}, response interface{}) error
}

// ghGraphQLAdapter adapts the go-gh GraphQL client to our interface.
type ghGraphQLAdapter struct {
	client *api.GraphQLClient
}

func (a *ghGraphQLAdapter) Do(query string, variables map[string]interface{}, response interface{}) error {
	return a.client.Do(query, variables, response)
}

// httpClientTimeout is the timeout for HTTP requests to the GitHub API.
const httpClientTimeout = 30 * time.Second

// Retry configuration constants
const (
	// maxRetries is the maximum number of retry attempts for transient errors.
	maxRetries = 3
	// initialBackoff is the initial delay before the first retry.
	initialBackoff = 500 * time.Millisecond
	// maxBackoff is the maximum delay between retries.
	maxBackoff = 10 * time.Second
	// backoffMultiplier is the factor by which backoff increases after each retry.
	backoffMultiplier = 2.0
)

// NewClient creates a new GitHub GraphQL client.
// It uses the gh CLI's authentication automatically.
func NewClient() (*Client, error) {
	graphqlClient, err := api.NewGraphQLClient(api.ClientOptions{
		Timeout: httpClientTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL client: %w", err)
	}

	return &Client{
		graphql: &ghGraphQLAdapter{client: graphqlClient},
	}, nil
}

// NewClientWithToken creates a new GitHub GraphQL client using an explicit auth token.
// This bypasses go-gh's in-process config cache, which is necessary after
// switching accounts via `gh auth switch` within a running process.
func NewClientWithToken(token string) (*Client, error) {
	graphqlClient, err := api.NewGraphQLClient(api.ClientOptions{
		AuthToken: token,
		Timeout:   httpClientTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL client: %w", err)
	}

	return &Client{
		graphql: &ghGraphQLAdapter{client: graphqlClient},
	}, nil
}


// NewClientWithGraphQL creates a new Client with a custom GraphQL client.
// This is primarily used for testing.
func NewClientWithGraphQL(graphql GraphQLClient) *Client {
	return &Client{
		graphql: graphql,
	}
}

// pageSize is the number of results to request per page.
// Using 50 as a reasonable tradeoff between API cost and number of requests.
const pageSize = 50

// maxPages is the maximum number of pages to fetch to prevent infinite loops.
const maxPages = 100

// rateLimitWarningThreshold triggers a warning when remaining requests fall below this value.
const rateLimitWarningThreshold = 100

// ErrEmptyUsername is returned when the username is empty after trimming.
var ErrEmptyUsername = errors.New("username cannot be empty")

// BuildSearchQuery constructs a GitHub search query for PRs authored by a user
// in specified organizations. Returns an error if username is empty after trimming.
func BuildSearchQuery(username string, organizations []string) (string, error) {
	username = strings.TrimSpace(username)
	if username == "" {
		return "", ErrEmptyUsername
	}

	if len(organizations) == 0 {
		return fmt.Sprintf("is:pr is:open author:%s archived:false", username), nil
	}

	// Build org filter: for single org use "org:X", for multiple use "org:X org:Y ..."
	// Skip empty orgs after trimming
	var orgFilters []string
	for _, org := range organizations {
		org = strings.TrimSpace(org)
		if org != "" {
			orgFilters = append(orgFilters, fmt.Sprintf("org:%s", org))
		}
	}

	if len(orgFilters) == 0 {
		return fmt.Sprintf("is:pr is:open author:%s archived:false", username), nil
	}

	return fmt.Sprintf("is:pr is:open author:%s %s archived:false", username, strings.Join(orgFilters, " ")), nil
}

// ErrPaginationLoop is returned when pagination appears to be stuck in an infinite loop.
var ErrPaginationLoop = errors.New("pagination loop detected: hasNextPage is true but endCursor is nil")

// ErrMaxPagesExceeded is returned when the maximum number of pages has been fetched.
var ErrMaxPagesExceeded = errors.New("maximum number of pages exceeded")

// isRetryableError determines if an error is transient and should be retried.
// Network errors, timeouts, and server errors (5xx) are retryable.
// Client errors (4xx) like auth or rate limit are not retryable.
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errStr := strings.ToLower(err.Error())

	// Network-related errors that are retryable
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"connection timed out",
		"timeout",
		"temporary failure",
		"eof",
		"network is unreachable",
		"no such host",
		"i/o timeout",
		"server error",
		"503",
		"502",
		"500",
		"504",
		"service unavailable",
		"bad gateway",
		"gateway timeout",
		"internal server error",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(errStr, pattern) {
			return true
		}
	}

	// Non-retryable errors - these indicate client-side issues
	nonRetryablePatterns := []string{
		"401",
		"403",
		"404",
		"422",
		"rate limit",
		"unauthorized",
		"forbidden",
		"not found",
		"bad credentials",
	}

	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(errStr, pattern) {
			return false
		}
	}

	// For unknown errors, assume they might be transient
	return false
}

// doWithRetry executes a GraphQL query with exponential backoff retry for transient errors.
// It respects context cancellation and returns immediately for non-retryable errors.
func (c *Client) doWithRetry(ctx context.Context, query string, variables map[string]interface{}, response interface{}) error {
	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait before retry (not on first attempt)
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
			// Increase backoff for next retry
			backoff = time.Duration(float64(backoff) * backoffMultiplier)
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		err := c.graphql.Do(query, variables, response)
		if err == nil {
			return nil
		}

		// Don't retry non-retryable errors
		if !isRetryableError(err) {
			return err
		}

		lastErr = err
	}

	// All retries exhausted
	return fmt.Errorf("after %d retries: %w", maxRetries, lastErr)
}

// FetchPRs fetches all open pull requests authored by the given username in the specified organizations.
// It handles pagination automatically and returns all results.
// Network errors are returned as errors (caller should preserve existing data).
// Rate limit warnings are included in the result but data is still returned.
func (c *Client) FetchPRs(ctx context.Context, username string, organizations []string) (*SearchResult, error) {
	searchQuery, err := BuildSearchQuery(username, organizations)
	if err != nil {
		return nil, fmt.Errorf("failed to build search query: %w", err)
	}

	var allPRs []PullRequestNode
	var cursor *string
	var latestRateLimit *RateLimit
	pageCount := 0

	for {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		// Check max pages to prevent infinite loops
		pageCount++
		if pageCount > maxPages {
			return nil, ErrMaxPagesExceeded
		}

		variables := map[string]interface{}{
			"query": searchQuery,
			"first": pageSize,
			"after": cursor,
		}

		// go-gh's Do method automatically unwraps the "data" field from GraphQL responses
		// and returns GraphQL errors as Go errors, so we use ResponseData directly.
		var response ResponseData
		err := c.doWithRetry(ctx, prSearchQuery, variables, &response)
		if err != nil {
			return nil, fmt.Errorf("GraphQL query failed: %w", err)
		}

		if response.Search == nil {
			return nil, fmt.Errorf("unexpected empty response from GitHub API")
		}

		// Collect PRs from this page, filtering only actual PullRequests
		for _, node := range response.Search.Nodes {
			if node.IsPullRequest() {
				allPRs = append(allPRs, node)
			}
		}

		// Update rate limit info
		latestRateLimit = response.RateLimit

		// Check if there are more pages
		if !response.Search.PageInfo.HasNextPage {
			break
		}

		// Guard against infinite loops: hasNextPage is true but endCursor is nil
		if response.Search.PageInfo.EndCursor == nil {
			return nil, ErrPaginationLoop
		}

		cursor = response.Search.PageInfo.EndCursor
	}

	result := &SearchResult{
		PullRequests: allPRs,
		RateLimit:    latestRateLimit,
	}

	// Check for rate limit warning
	if latestRateLimit != nil && latestRateLimit.Remaining < rateLimitWarningThreshold {
		resetIn := time.Until(latestRateLimit.ResetAt)
		resetMinutes := int(resetIn.Minutes())
		if resetMinutes < 1 {
			resetMinutes = 1
		}
		result.Warning = fmt.Sprintf(
			"GitHub API rate limit low: %d remaining, resets in %d minutes",
			latestRateLimit.Remaining,
			resetMinutes,
		)
	}

	return result, nil
}
