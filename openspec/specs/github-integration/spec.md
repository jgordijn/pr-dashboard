## Purpose

Define how `pr-dashboard` retrieves pull request data from GitHub and transforms it into domain models used by the TUI.

## Requirements

### Requirement: GitHub GraphQL API Integration
The system SHALL use GitHub GraphQL API to fetch pull request data efficiently.

#### Scenario: Fetch PRs for user in organization
- **WHEN** fetching PRs for a configured user and organization
- **THEN** the system SHALL execute a GraphQL search query with:
  - Filter: `is:pr is:open author:{username} org:{organization} archived:false`
- **AND** fetch all required fields including id, updatedAt, and rateLimit in a single request

#### Scenario: Handle pagination
- **WHEN** the user has more than 50 open PRs
- **THEN** the system SHALL paginate through all results using cursors

#### Scenario: Handle rate limiting
- **WHEN** the GitHub API returns rate limit information
- **THEN** the system SHALL track remaining quota and reset time
- **AND** display a warning in the status bar when remaining quota is 100 or less
- **AND** show the reset time in the warning
- **AND** continue allowing manual refresh attempts even when limited

#### Scenario: Handle network errors
- **WHEN** a network error occurs during API requests
- **THEN** the system SHALL display an error in the status bar
- **AND** retain previously fetched data in the view

### Requirement: PR Data Transformation
The system SHALL transform GitHub API responses into domain models.

#### Scenario: Calculate days open
- **WHEN** a PR is fetched from the API
- **THEN** the system SHALL calculate days since creation from createdAt

#### Scenario: Count unresolved review threads
- **WHEN** a PR has review threads
- **THEN** the system SHALL count unresolved threads from the first 100 nodes
- **AND** use the totalCount field to detect if there are more than 100 threads
- **AND** display with "+" suffix (e.g., "15+") when totalCount exceeds 100
- **AND** if count is 0 but totalCount exceeds 100, display "0+" to indicate sampling

#### Scenario: Aggregate check status
- **WHEN** a PR has CI checks
- **THEN** the system SHALL use statusCheckRollup.state for aggregate status:
  - SUCCESS → PASSING
  - FAILURE or ERROR → FAILING
  - PENDING → PENDING
  - null → NONE

#### Scenario: Generate stable PR key
- **WHEN** a PR is transformed from API response
- **THEN** the system SHALL generate a stable key in format `owner/repo#number`

#### Scenario: Handle team review requests
- **WHEN** a PR has team review requests
- **THEN** the system SHALL include team name with "team:" prefix
