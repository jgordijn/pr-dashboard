package github

import (
	"strings"
	"testing"
)

func TestPrSearchQuery_ContainsRequiredFields(t *testing.T) {
	// Verify the query contains all required fields for the PR dashboard
	requiredFields := []string{
		"id",           // for stable selection tracking
		"updatedAt",    // for sorting
		"__typename",   // for type discrimination
		"number",       // PR number
		"title",        // PR title
		"url",          // PR URL
		"isDraft",      // draft status
		"createdAt",    // creation time
		"mergeable",    // merge state
		"reviewDecision",
	}

	for _, field := range requiredFields {
		if !strings.Contains(prSearchQuery, field) {
			t.Errorf("prSearchQuery missing required field: %s", field)
		}
	}
}

func TestPrSearchQuery_ContainsRateLimit(t *testing.T) {
	// Verify the query includes rate limit information
	rateLimitFields := []string{
		"rateLimit",
		"remaining",
		"resetAt",
		"cost",
	}

	for _, field := range rateLimitFields {
		if !strings.Contains(prSearchQuery, field) {
			t.Errorf("prSearchQuery missing rate limit field: %s", field)
		}
	}
}

func TestPrSearchQuery_HandlesTeamType(t *testing.T) {
	// Verify the query handles Team type in reviewRequests union
	teamFragments := []string{
		"... on Team",
		"name",
		"slug",
	}

	for _, fragment := range teamFragments {
		if !strings.Contains(prSearchQuery, fragment) {
			t.Errorf("prSearchQuery missing Team handling: %s", fragment)
		}
	}
}

func TestPrSearchQuery_HasPaginationSupport(t *testing.T) {
	// Verify the query supports cursor-based pagination
	paginationElements := []string{
		"$after",
		"$first",
		"pageInfo",
		"hasNextPage",
		"endCursor",
	}

	for _, element := range paginationElements {
		if !strings.Contains(prSearchQuery, element) {
			t.Errorf("prSearchQuery missing pagination element: %s", element)
		}
	}
}

func TestPrSearchQuery_TypenameAtNodesLevel(t *testing.T) {
	// Verify __typename is selected at the nodes level (not just inside PullRequest fragment)
	// This ensures we can distinguish non-PR nodes from PR nodes unambiguously
	//
	// Correct structure:
	//   nodes {
	//     __typename
	//     ... on PullRequest { ... }
	//   }
	//
	// Incorrect structure (what we want to avoid):
	//   nodes {
	//     ... on PullRequest {
	//       __typename
	//       ...
	//     }
	//   }

	// Find "nodes {" and verify __typename appears before "... on PullRequest"
	nodesIndex := strings.Index(prSearchQuery, "nodes {")
	if nodesIndex == -1 {
		t.Fatal("prSearchQuery missing 'nodes {' block")
	}

	afterNodes := prSearchQuery[nodesIndex:]
	typenameIndex := strings.Index(afterNodes, "__typename")
	prFragmentIndex := strings.Index(afterNodes, "... on PullRequest")

	if typenameIndex == -1 {
		t.Fatal("prSearchQuery missing __typename")
	}
	if prFragmentIndex == -1 {
		t.Fatal("prSearchQuery missing '... on PullRequest' fragment")
	}

	if typenameIndex > prFragmentIndex {
		t.Error("__typename should appear at nodes level (before '... on PullRequest'), not inside the fragment")
	}
}

func TestPrSearchQuery_UsesLastForReviews(t *testing.T) {
	// Verify reviews uses 'last' instead of 'first' to get most recent reviews
	if !strings.Contains(prSearchQuery, "reviews(last:") {
		t.Error("prSearchQuery should use 'last:' for reviews to get most recent reviews, not 'first:'")
	}
}
