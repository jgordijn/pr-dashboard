package github

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// mockGraphQLClient is a mock implementation of GraphQLClient for testing.
type mockGraphQLClient struct {
	responses []mockResponse
	callIndex int
	calls     []mockCall
}

type mockResponse struct {
	response interface{}
	err      error
}

type mockCall struct {
	query     string
	variables map[string]interface{}
}

func (m *mockGraphQLClient) Do(query string, variables map[string]interface{}, response interface{}) error {
	m.calls = append(m.calls, mockCall{query: query, variables: variables})

	if m.callIndex >= len(m.responses) {
		return errors.New("no more mock responses configured")
	}

	resp := m.responses[m.callIndex]
	m.callIndex++

	if resp.err != nil {
		return resp.err
	}

	// Copy the mock response to the target
	// go-gh's Do method already unwraps the "data" field, so we use ResponseData directly
	if resp.response != nil {
		// Type assert and copy - fail fast if types don't match
		responseData, ok := resp.response.(*ResponseData)
		if !ok {
			return errors.New("mock response is not *ResponseData")
		}
		target, ok := response.(*ResponseData)
		if !ok {
			return errors.New("response target is not *ResponseData")
		}
		*target = *responseData
	}

	return nil
}

func TestBuildSearchQuery(t *testing.T) {
	tests := []struct {
		name          string
		username      string
		organizations []string
		want          string
		wantErr       error
	}{
		{
			name:          "single organization",
			username:      "testuser",
			organizations: []string{"myorg"},
			want:          "is:pr is:open author:testuser org:myorg archived:false",
		},
		{
			name:          "multiple organizations",
			username:      "testuser",
			organizations: []string{"org1", "org2", "org3"},
			want:          "is:pr is:open author:testuser org:org1 org:org2 org:org3 archived:false",
		},
		{
			name:          "no organizations",
			username:      "testuser",
			organizations: []string{},
			want:          "is:pr is:open author:testuser archived:false",
		},
		{
			name:          "nil organizations",
			username:      "testuser",
			organizations: nil,
			want:          "is:pr is:open author:testuser archived:false",
		},
		{
			name:          "username with hyphen",
			username:      "test-user",
			organizations: []string{"my-org"},
			want:          "is:pr is:open author:test-user org:my-org archived:false",
		},
		{
			name:          "username with whitespace is trimmed",
			username:      "  testuser  ",
			organizations: []string{"myorg"},
			want:          "is:pr is:open author:testuser org:myorg archived:false",
		},
		{
			name:          "orgs with whitespace are trimmed",
			username:      "testuser",
			organizations: []string{"  org1  ", "org2"},
			want:          "is:pr is:open author:testuser org:org1 org:org2 archived:false",
		},
		{
			name:          "empty orgs after trimming are skipped",
			username:      "testuser",
			organizations: []string{"org1", "  ", "", "org2"},
			want:          "is:pr is:open author:testuser org:org1 org:org2 archived:false",
		},
		{
			name:          "all orgs empty after trimming",
			username:      "testuser",
			organizations: []string{"  ", ""},
			want:          "is:pr is:open author:testuser archived:false",
		},
		{
			name:          "empty username",
			username:      "",
			organizations: []string{"myorg"},
			wantErr:       ErrEmptyUsername,
		},
		{
			name:          "whitespace-only username",
			username:      "   ",
			organizations: []string{"myorg"},
			wantErr:       ErrEmptyUsername,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildSearchQuery(tt.username, tt.organizations)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("BuildSearchQuery() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("BuildSearchQuery() unexpected error = %v", err)
			}
			if got != tt.want {
				t.Errorf("BuildSearchQuery() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFetchPRs_SinglePage(t *testing.T) {
	resetTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes: []PullRequestNode{
							{
								Typename: "PullRequest",
								ID:       "PR_123",
								Number:   123,
								Title:    "Test PR",
								URL:      "https://github.com/org/repo/pull/123",
								IsDraft:  false,
								Repository: Repository{
									Owner: RepositoryOwner{Login: "myorg"},
									Name:  "myrepo",
								},
								Author: &Actor{Login: "testuser"},
							},
						},
						PageInfo: PageInfo{
							HasNextPage: false,
							EndCursor:   nil,
						},
					},
					RateLimit: &RateLimit{
						Remaining: 4999,
						ResetAt:   resetTime,
						Cost:      1,
					},
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err != nil {
		t.Fatalf("FetchPRs() error = %v", err)
	}

	if len(result.PullRequests) != 1 {
		t.Errorf("Expected 1 PR, got %d", len(result.PullRequests))
	}

	if result.PullRequests[0].Number != 123 {
		t.Errorf("Expected PR number 123, got %d", result.PullRequests[0].Number)
	}

	if result.RateLimit == nil {
		t.Fatal("Expected RateLimit to be set")
	}

	if result.RateLimit.Remaining != 4999 {
		t.Errorf("Expected remaining 4999, got %d", result.RateLimit.Remaining)
	}

	if result.Warning != "" {
		t.Errorf("Expected no warning, got %q", result.Warning)
	}

	// Verify the query was called with correct variables
	if len(mockClient.calls) != 1 {
		t.Fatalf("Expected 1 call, got %d", len(mockClient.calls))
	}

	expectedQuery := "is:pr is:open author:testuser org:myorg archived:false"
	if mockClient.calls[0].variables["query"] != expectedQuery {
		t.Errorf("Query = %q, want %q", mockClient.calls[0].variables["query"], expectedQuery)
	}
}

func TestFetchPRs_Pagination(t *testing.T) {
	resetTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	cursor := "cursor123"

	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes: []PullRequestNode{
							{
								Typename: "PullRequest",
								ID:       "PR_1",
								Number:   1,
								Title:    "First PR",
							},
						},
						PageInfo: PageInfo{
							HasNextPage: true,
							EndCursor:   &cursor,
						},
					},
					RateLimit: &RateLimit{
						Remaining: 4998,
						ResetAt:   resetTime,
						Cost:      1,
					},
				},
			},
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes: []PullRequestNode{
							{
								Typename: "PullRequest",
								ID:       "PR_2",
								Number:   2,
								Title:    "Second PR",
							},
						},
						PageInfo: PageInfo{
							HasNextPage: false,
							EndCursor:   nil,
						},
					},
					RateLimit: &RateLimit{
						Remaining: 4997,
						ResetAt:   resetTime,
						Cost:      1,
					},
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err != nil {
		t.Fatalf("FetchPRs() error = %v", err)
	}

	if len(result.PullRequests) != 2 {
		t.Errorf("Expected 2 PRs, got %d", len(result.PullRequests))
	}

	if result.PullRequests[0].Number != 1 {
		t.Errorf("Expected first PR number 1, got %d", result.PullRequests[0].Number)
	}

	if result.PullRequests[1].Number != 2 {
		t.Errorf("Expected second PR number 2, got %d", result.PullRequests[1].Number)
	}

	// Verify pagination used the cursor
	if len(mockClient.calls) != 2 {
		t.Fatalf("Expected 2 calls, got %d", len(mockClient.calls))
	}

	// First call should have nil cursor
	if mockClient.calls[0].variables["after"] != (*string)(nil) {
		t.Errorf("First call should have nil cursor")
	}

	// Second call should have the cursor from first response
	if mockClient.calls[1].variables["after"] != &cursor {
		t.Errorf("Second call should have cursor %q", cursor)
	}

	// Rate limit should be from the last response
	if result.RateLimit.Remaining != 4997 {
		t.Errorf("Expected remaining 4997, got %d", result.RateLimit.Remaining)
	}
}

func TestFetchPRs_RateLimitWarning(t *testing.T) {
	// Set reset time to 10 minutes in the future
	resetTime := time.Now().Add(10 * time.Minute)

	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes:    []PullRequestNode{},
						PageInfo: PageInfo{HasNextPage: false},
					},
					RateLimit: &RateLimit{
						Remaining: 50, // Below threshold of 100
						ResetAt:   resetTime,
						Cost:      1,
					},
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err != nil {
		t.Fatalf("FetchPRs() error = %v", err)
	}

	if result.Warning == "" {
		t.Error("Expected warning for low rate limit")
	}

	// Check warning contains expected components (minutes may vary slightly due to timing)
	if !strings.Contains(result.Warning, "GitHub API rate limit low: 50 remaining") {
		t.Errorf("Warning should contain rate limit info, got %q", result.Warning)
	}
	if !strings.Contains(result.Warning, "resets in") && !strings.Contains(result.Warning, "minutes") {
		t.Errorf("Warning should contain reset time in minutes, got %q", result.Warning)
	}
}

func TestFetchPRs_RateLimitAtThreshold(t *testing.T) {
	resetTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes:    []PullRequestNode{},
						PageInfo: PageInfo{HasNextPage: false},
					},
					RateLimit: &RateLimit{
						Remaining: 100, // Exactly at threshold - no warning
						ResetAt:   resetTime,
						Cost:      1,
					},
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err != nil {
		t.Fatalf("FetchPRs() error = %v", err)
	}

	if result.Warning != "" {
		t.Errorf("Expected no warning at threshold, got %q", result.Warning)
	}
}

func TestFetchPRs_NetworkError(t *testing.T) {
	// Use a non-retryable error to test error handling without retries
	networkErr := errors.New("HTTP 401: unauthorized")

	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				err: networkErr,
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result on error, got %+v", result)
	}

	if !errors.Is(err, networkErr) {
		t.Errorf("Expected error to wrap networkErr, got %v", err)
	}
}

func TestFetchPRs_GraphQLErrors(t *testing.T) {
	// go-gh returns GraphQL errors as Go errors, not in the response struct
	graphqlErr := errors.New("GraphQL: Field 'invalid' doesn't exist")

	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				err: graphqlErr,
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result on error, got %+v", result)
	}

	if !strings.Contains(err.Error(), "GraphQL") {
		t.Errorf("Error should contain GraphQL info, got %q", err.Error())
	}
}

func TestFetchPRs_EmptyResponse(t *testing.T) {
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				response: &ResponseData{
					Search: nil, // Search is nil
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err == nil {
		t.Fatal("Expected error for empty response, got nil")
	}

	if result != nil {
		t.Errorf("Expected nil result on error, got %+v", result)
	}
}

func TestFetchPRs_FilterNonPullRequests(t *testing.T) {
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes: []PullRequestNode{
							{
								Typename: "PullRequest",
								ID:       "PR_1",
								Number:   1,
								Title:    "Valid PR",
							},
							{
								Typename: "Issue", // Should be filtered out
								ID:       "ISSUE_1",
								Number:   2,
								Title:    "This is an issue",
							},
							{
								Typename: "PullRequest",
								ID:       "PR_2",
								Number:   3,
								Title:    "Another valid PR",
							},
							{
								Typename: "", // Missing typename should be filtered
								ID:       "UNKNOWN_1",
								Number:   4,
							},
						},
						PageInfo: PageInfo{HasNextPage: false},
					},
					RateLimit: &RateLimit{Remaining: 5000},
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err != nil {
		t.Fatalf("FetchPRs() error = %v", err)
	}

	if len(result.PullRequests) != 2 {
		t.Errorf("Expected 2 PRs after filtering, got %d", len(result.PullRequests))
	}

	for _, pr := range result.PullRequests {
		if !pr.IsPullRequest() {
			t.Errorf("Non-PullRequest node not filtered: %+v", pr)
		}
	}
}

func TestFetchPRs_ContextCancellation(t *testing.T) {
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{}, // No responses needed
	}

	client := NewClientWithGraphQL(mockClient)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	result, err := client.FetchPRs(ctx, "testuser", []string{"myorg"})

	if err == nil {
		t.Fatal("Expected error for cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result, got %+v", result)
	}
}

func TestFetchPRs_CompleteResponseParsing(t *testing.T) {
	resetTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	createdAt := time.Date(2024, 1, 10, 9, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2024, 1, 14, 15, 30, 0, 0, time.UTC)
	mergeable := "MERGEABLE"
	mergeStateStatus := "CLEAN"
	reviewDecision := "APPROVED"
	login := "reviewer1"
	teamName := "core-team"
	teamSlug := "core-team"
	checkState := "SUCCESS"
	checkConclusion := "SUCCESS"
	checkStatus := "COMPLETED"
	statusState := "success"
	ciContext := "ci/build"

	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes: []PullRequestNode{
							{
								Typename:         "PullRequest",
								ID:               "PR_123",
								Number:           123,
								Title:            "Add new feature",
								URL:              "https://github.com/org/repo/pull/123",
								IsDraft:          false,
								CreatedAt:        createdAt,
								UpdatedAt:        updatedAt,
								Mergeable:        &mergeable,
								MergeStateStatus: &mergeStateStatus,
								ReviewDecision:   &reviewDecision,
								Repository: Repository{
									Owner: RepositoryOwner{Login: "myorg"},
									Name:  "myrepo",
								},
								Author: &Actor{Login: "testuser"},
								ReviewRequests: ReviewRequestConnection{
									Nodes: []ReviewRequest{
										{
											RequestedReviewer: RequestedReviewer{
												Typename: "User",
												Login:    &login,
											},
										},
										{
											RequestedReviewer: RequestedReviewer{
												Typename: "Team",
												Name:     &teamName,
												Slug:     &teamSlug,
											},
										},
									},
								},
								Reviews: ReviewConnection{
									Nodes: []Review{
										{
											Author: &Actor{Login: "reviewer1"},
											State:  "APPROVED",
										},
									},
								},
								ReviewThreads: ReviewThreadConnection{
									TotalCount: 2,
									Nodes: []ReviewThread{
										{IsResolved: true},
										{IsResolved: false},
									},
								},
								Commits: CommitConnection{
									Nodes: []CommitNode{
										{
											Commit: Commit{
												StatusCheckRollup: &StatusCheckRollup{
													State: &checkState,
													Contexts: CheckStatusContexts{
														Nodes: []CheckStatusContext{
															{
																Typename:   "CheckRun",
																Name:       ptr("build"),
																Conclusion: &checkConclusion,
																Status:     &checkStatus,
															},
															{
																Typename: "StatusContext",
																Context:  &ciContext,
																State:    &statusState,
															},
														},
													},
												},
											},
										},
									},
								},
							},
						},
						PageInfo: PageInfo{HasNextPage: false},
					},
					RateLimit: &RateLimit{
						Remaining: 4999,
						ResetAt:   resetTime,
						Cost:      1,
					},
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err != nil {
		t.Fatalf("FetchPRs() error = %v", err)
	}

	if len(result.PullRequests) != 1 {
		t.Fatalf("Expected 1 PR, got %d", len(result.PullRequests))
	}

	pr := result.PullRequests[0]

	// Verify all fields are parsed correctly
	if pr.Title != "Add new feature" {
		t.Errorf("Title = %q, want %q", pr.Title, "Add new feature")
	}

	if pr.Author == nil || pr.Author.Login != "testuser" {
		t.Errorf("Author login not parsed correctly")
	}

	if *pr.Mergeable != "MERGEABLE" {
		t.Errorf("Mergeable = %v, want MERGEABLE", pr.Mergeable)
	}

	if *pr.ReviewDecision != "APPROVED" {
		t.Errorf("ReviewDecision = %v, want APPROVED", pr.ReviewDecision)
	}

	if len(pr.ReviewRequests.Nodes) != 2 {
		t.Errorf("Expected 2 review requests, got %d", len(pr.ReviewRequests.Nodes))
	}

	if len(pr.Reviews.Nodes) != 1 {
		t.Errorf("Expected 1 review, got %d", len(pr.Reviews.Nodes))
	}

	if pr.ReviewThreads.TotalCount != 2 {
		t.Errorf("Expected 2 review threads, got %d", pr.ReviewThreads.TotalCount)
	}

	// Verify commit status check parsing
	if len(pr.Commits.Nodes) != 1 {
		t.Fatalf("Expected 1 commit node, got %d", len(pr.Commits.Nodes))
	}

	statusRollup := pr.Commits.Nodes[0].Commit.StatusCheckRollup
	if statusRollup == nil {
		t.Fatal("Expected StatusCheckRollup to be set")
	}

	if len(statusRollup.Contexts.Nodes) != 2 {
		t.Errorf("Expected 2 check contexts, got %d", len(statusRollup.Contexts.Nodes))
	}
}

func TestFetchPRs_MultipleOrganizations(t *testing.T) {
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes:    []PullRequestNode{},
						PageInfo: PageInfo{HasNextPage: false},
					},
					RateLimit: &RateLimit{Remaining: 5000},
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	_, err := client.FetchPRs(context.Background(), "testuser", []string{"org1", "org2", "org3"})

	if err != nil {
		t.Fatalf("FetchPRs() error = %v", err)
	}

	expectedQuery := "is:pr is:open author:testuser org:org1 org:org2 org:org3 archived:false"
	actualQuery := mockClient.calls[0].variables["query"]
	if actualQuery != expectedQuery {
		t.Errorf("Query = %q, want %q", actualQuery, expectedQuery)
	}
}

func TestNewClientWithGraphQL(t *testing.T) {
	mockClient := &mockGraphQLClient{}
	client := NewClientWithGraphQL(mockClient)

	if client == nil {
		t.Fatal("NewClientWithGraphQL returned nil")
	}

	if client.graphql != mockClient {
		t.Error("GraphQL client not set correctly")
	}
}

func TestFetchPRs_PaginationLoopDetection(t *testing.T) {
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes: []PullRequestNode{
							{
								Typename: "PullRequest",
								ID:       "PR_1",
								Number:   1,
								Title:    "First PR",
							},
						},
						PageInfo: PageInfo{
							HasNextPage: true,
							EndCursor:   nil, // This should trigger the loop detection
						},
					},
					RateLimit: &RateLimit{Remaining: 5000},
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	_, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err == nil {
		t.Fatal("Expected error for pagination loop")
	}

	if !errors.Is(err, ErrPaginationLoop) {
		t.Errorf("Expected ErrPaginationLoop, got %v", err)
	}
}

func TestFetchPRs_EmptyUsernameError(t *testing.T) {
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{}, // No responses needed
	}

	client := NewClientWithGraphQL(mockClient)
	_, err := client.FetchPRs(context.Background(), "", []string{"myorg"})

	if err == nil {
		t.Fatal("Expected error for empty username")
	}

	if !errors.Is(err, ErrEmptyUsername) {
		t.Errorf("Expected ErrEmptyUsername, got %v", err)
	}
}

// ptr is a helper function to create a pointer to a string.
func ptr(s string) *string {
	return &s
}

// TestIsRetryableError tests the isRetryableError helper function.
func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		// Retryable errors
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "connection refused",
			err:       errors.New("dial tcp: connection refused"),
			retryable: true,
		},
		{
			name:      "connection reset",
			err:       errors.New("connection reset by peer"),
			retryable: true,
		},
		{
			name:      "timeout error",
			err:       errors.New("request timeout"),
			retryable: true,
		},
		{
			name:      "i/o timeout",
			err:       errors.New("i/o timeout"),
			retryable: true,
		},
		{
			name:      "temporary failure",
			err:       errors.New("temporary failure in name resolution"),
			retryable: true,
		},
		{
			name:      "eof error",
			err:       errors.New("unexpected EOF"),
			retryable: true,
		},
		{
			name:      "500 internal server error",
			err:       errors.New("HTTP 500: internal server error"),
			retryable: true,
		},
		{
			name:      "502 bad gateway",
			err:       errors.New("HTTP 502: bad gateway"),
			retryable: true,
		},
		{
			name:      "503 service unavailable",
			err:       errors.New("HTTP 503: service unavailable"),
			retryable: true,
		},
		{
			name:      "504 gateway timeout",
			err:       errors.New("HTTP 504: gateway timeout"),
			retryable: true,
		},
		// Non-retryable errors
		{
			name:      "401 unauthorized",
			err:       errors.New("HTTP 401: unauthorized"),
			retryable: false,
		},
		{
			name:      "403 forbidden",
			err:       errors.New("HTTP 403: forbidden"),
			retryable: false,
		},
		{
			name:      "404 not found",
			err:       errors.New("HTTP 404: not found"),
			retryable: false,
		},
		{
			name:      "422 unprocessable entity",
			err:       errors.New("HTTP 422: unprocessable entity"),
			retryable: false,
		},
		{
			name:      "rate limit exceeded",
			err:       errors.New("rate limit exceeded"),
			retryable: false,
		},
		{
			name:      "bad credentials",
			err:       errors.New("bad credentials"),
			retryable: false,
		},
		{
			name:      "unknown error",
			err:       errors.New("some random error"),
			retryable: false, // Unknown errors are not retried by default
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isRetryableError(tt.err)
			if got != tt.retryable {
				t.Errorf("isRetryableError(%v) = %v, want %v", tt.err, got, tt.retryable)
			}
		})
	}
}

// TestFetchPRs_RetryOnTransientError tests that transient errors trigger retries.
func TestFetchPRs_RetryOnTransientError(t *testing.T) {
	resetTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// First two calls fail with transient error, third succeeds
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{err: errors.New("connection refused")},
			{err: errors.New("connection refused")},
			{
				response: &ResponseData{
					Search: &SearchResponse{
						Nodes: []PullRequestNode{
							{
								Typename: "PullRequest",
								ID:       "PR_1",
								Number:   1,
								Title:    "Test PR",
							},
						},
						PageInfo: PageInfo{HasNextPage: false},
					},
					RateLimit: &RateLimit{Remaining: 5000, ResetAt: resetTime},
				},
			},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	result, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err != nil {
		t.Fatalf("FetchPRs() should succeed after retries, got error: %v", err)
	}

	if len(result.PullRequests) != 1 {
		t.Errorf("Expected 1 PR, got %d", len(result.PullRequests))
	}

	// Verify that 3 calls were made (2 failures + 1 success)
	if len(mockClient.calls) != 3 {
		t.Errorf("Expected 3 calls (with retries), got %d", len(mockClient.calls))
	}
}

// TestFetchPRs_NoRetryOnNonRetryableError tests that non-retryable errors don't trigger retries.
func TestFetchPRs_NoRetryOnNonRetryableError(t *testing.T) {
	// Non-retryable error (401 unauthorized)
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{err: errors.New("HTTP 401: unauthorized")},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	_, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	// Verify only 1 call was made (no retries)
	if len(mockClient.calls) != 1 {
		t.Errorf("Expected 1 call (no retries for 401), got %d", len(mockClient.calls))
	}
}

// TestFetchPRs_MaxRetriesExhausted tests that all retries are exhausted on persistent transient errors.
func TestFetchPRs_MaxRetriesExhausted(t *testing.T) {
	// All 4 calls (initial + 3 retries) fail with transient error
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{err: errors.New("connection refused")},
			{err: errors.New("connection refused")},
			{err: errors.New("connection refused")},
			{err: errors.New("connection refused")},
		},
	}

	client := NewClientWithGraphQL(mockClient)
	_, err := client.FetchPRs(context.Background(), "testuser", []string{"myorg"})

	if err == nil {
		t.Fatal("Expected error after max retries, got nil")
	}

	// Error should mention retries
	if !strings.Contains(err.Error(), "after 3 retries") {
		t.Errorf("Error should mention retries, got: %v", err)
	}

	// Verify all 4 attempts were made (initial + 3 retries)
	if len(mockClient.calls) != 4 {
		t.Errorf("Expected 4 calls (initial + 3 retries), got %d", len(mockClient.calls))
	}
}

// TestFetchPRs_RetryRespectsContextCancellation tests that context cancellation stops retries.
func TestFetchPRs_RetryRespectsContextCancellation(t *testing.T) {
	// Set up mock to return transient errors
	mockClient := &mockGraphQLClient{
		responses: []mockResponse{
			{err: errors.New("connection refused")},
			{err: errors.New("connection refused")},
			{err: errors.New("connection refused")},
		},
	}

	client := NewClientWithGraphQL(mockClient)

	// Create a context that we'll cancel during the test
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay (should happen during retry backoff)
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	_, err := client.FetchPRs(ctx, "testuser", []string{"myorg"})

	if err == nil {
		t.Fatal("Expected error from cancelled context")
	}

	if !errors.Is(err, context.Canceled) {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

// TestDoWithRetry_DirectTest tests the doWithRetry method directly.
func TestDoWithRetry_DirectTest(t *testing.T) {
	t.Run("success on first try", func(t *testing.T) {
		mockClient := &mockGraphQLClient{
			responses: []mockResponse{
				{
					response: &ResponseData{
						Search: &SearchResponse{
							Nodes:    []PullRequestNode{},
							PageInfo: PageInfo{HasNextPage: false},
						},
						RateLimit: &RateLimit{Remaining: 5000},
					},
				},
			},
		}

		client := NewClientWithGraphQL(mockClient)
		var response ResponseData
		err := client.doWithRetry(context.Background(), "query", nil, &response)

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if len(mockClient.calls) != 1 {
			t.Errorf("Expected 1 call, got %d", len(mockClient.calls))
		}
	})

	t.Run("success after retries", func(t *testing.T) {
		mockClient := &mockGraphQLClient{
			responses: []mockResponse{
				{err: errors.New("timeout")},
				{err: errors.New("timeout")},
				{
					response: &ResponseData{
						Search: &SearchResponse{
							Nodes:    []PullRequestNode{},
							PageInfo: PageInfo{HasNextPage: false},
						},
						RateLimit: &RateLimit{Remaining: 5000},
					},
				},
			},
		}

		client := NewClientWithGraphQL(mockClient)
		var response ResponseData
		err := client.doWithRetry(context.Background(), "query", nil, &response)

		if err != nil {
			t.Fatalf("Expected no error after retries, got: %v", err)
		}

		if len(mockClient.calls) != 3 {
			t.Errorf("Expected 3 calls, got %d", len(mockClient.calls))
		}
	})
}
