// Package github provides types and client for GitHub GraphQL API integration.
package github

// prSearchQuery is the GraphQL query template for fetching pull requests.
// It searches for open PRs authored by a user and retrieves all fields needed
// for the PR dashboard display.
//
// Key features:
// - Uses cursor-based pagination with $after variable
// - Includes __typename at nodes level for unambiguous type discrimination
// - Fetches rate limit info to monitor API usage
// - Handles both User and Team types in reviewRequests
// - Uses last: 20 for reviews to get most recent reviews
//
// Search query template: "is:pr is:open author:{user} org:{org} archived:false"
const prSearchQuery = `
query($query: String!, $first: Int!, $after: String) {
  search(query: $query, type: ISSUE, first: $first, after: $after) {
    nodes {
      __typename
      ... on PullRequest {
        id
        number
        title
        url
        isDraft
        createdAt
        updatedAt
        mergeable
        mergeStateStatus
        reviewDecision
        repository {
          owner { login }
          name
        }
        author { login }
        reviewRequests(first: 10) {
          nodes {
            requestedReviewer {
              __typename
              ... on User { login }
              ... on Team { name slug }
            }
          }
        }
        reviews(last: 20, states: [APPROVED, CHANGES_REQUESTED]) {
          nodes {
            author { login }
            state
          }
        }
        reviewThreads(first: 100) {
          totalCount
          nodes { isResolved }
        }
        commits(last: 1) {
          nodes {
            commit {
              statusCheckRollup {
                state
                contexts(first: 50) {
                  nodes {
                    __typename
                    ... on CheckRun {
                      name
                      conclusion
                      status
                    }
                    ... on StatusContext {
                      context
                      state
                    }
                  }
                }
              }
            }
          }
        }
      }
    }
    pageInfo {
      hasNextPage
      endCursor
    }
  }
  rateLimit {
    remaining
    resetAt
    cost
  }
}
`
