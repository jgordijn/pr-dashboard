## ADDED Requirements

### Requirement: PR List Display
The system SHALL display a list of open pull requests grouped by organization.

#### Scenario: Display PR with full status information
- **WHEN** the dashboard loads successfully
- **THEN** each PR SHALL display:
  - PR number and title
  - Draft/Ready status badge
  - Days since opened
  - CI check status (passing/failing/running)
  - Review status (approved/pending/changes requested) with reviewer names
  - Unresolved review thread count (with "+" suffix if truncated)
  - Merge status (clean/behind base branch/blocked/conflicts/dirty/unstable/has_hooks/unknown)
  - Repository name

#### Scenario: Display organization headers
- **WHEN** PRs are fetched from multiple organizations
- **THEN** PRs SHALL be grouped under collapsible organization headers
- **AND** each header SHALL show the organization name and PR count
- **AND** PRs within each org SHALL be sorted by updatedAt descending

#### Scenario: Display draft PRs distinctly
- **WHEN** a PR is marked as draft
- **THEN** the PR SHALL display a "DRAFT" badge
- **AND** the PR SHALL be visually dimmed compared to ready PRs

#### Scenario: Handle empty PR list
- **WHEN** the user has no open PRs
- **THEN** the system SHALL display a friendly message "No open PRs - nice work! 🎉"

#### Scenario: Truncate long PR titles
- **WHEN** a PR title exceeds available terminal width
- **THEN** the title SHALL be truncated with an ellipsis
- **AND** status columns SHALL remain visible

#### Scenario: Handle team review requests
- **WHEN** a PR has a team requested as reviewer
- **THEN** the team SHALL be displayed as "team:<name>"

### Requirement: Display Mode Cycling
The system SHALL support three display modes: full, compact, and minimal.

#### Scenario: Full display mode
- **WHEN** display mode is set to "full"
- **THEN** all status columns SHALL be visible with detailed information including reviewer names

#### Scenario: Compact display mode
- **WHEN** display mode is set to "compact"
- **THEN** status SHALL be shown as summary icons only (✓/⏳/✗) with counts

#### Scenario: Minimal display mode
- **WHEN** display mode is set to "minimal"
- **THEN** only PR title, status badge (Draft/Ready), and age SHALL be displayed

### Requirement: Draft Visibility Toggle
The system SHALL allow toggling visibility of draft PRs.

#### Scenario: Hide drafts
- **WHEN** user presses 'd' key with drafts visible
- **THEN** all draft PRs SHALL be hidden from the list
- **AND** if the selected PR was a draft, selection SHALL move to nearest visible PR

#### Scenario: Show drafts
- **WHEN** user presses 'd' key with drafts hidden
- **THEN** all draft PRs SHALL become visible again

### Requirement: Merge Status Display
The system SHALL display merge status based on GitHub mergeable and mergeStateStatus fields.

#### Scenario: Clean status
- **WHEN** mergeable is MERGEABLE and mergeStateStatus is CLEAN
- **THEN** display "✓ Clean" in green

#### Scenario: Behind base branch status
- **WHEN** mergeable is MERGEABLE and mergeStateStatus is BEHIND
- **THEN** display "⚠ Behind" in yellow

#### Scenario: Blocked status
- **WHEN** mergeable is MERGEABLE and mergeStateStatus is BLOCKED
- **THEN** display "⚠ Blocked" in yellow

#### Scenario: Conflicts status
- **WHEN** mergeable is CONFLICTING
- **THEN** display "✗ Conflicts" in red

#### Scenario: Dirty status
- **WHEN** mergeStateStatus is DIRTY
- **THEN** display "✗ Dirty" in red

#### Scenario: Unstable status
- **WHEN** mergeStateStatus is UNSTABLE
- **THEN** display "⚠ Unstable" in yellow

#### Scenario: Has hooks status
- **WHEN** mergeStateStatus is HAS_HOOKS
- **THEN** display "⚠ Has Hooks" in yellow

#### Scenario: Unknown status
- **WHEN** mergeable is UNKNOWN
- **OR** mergeStateStatus is UNKNOWN or null
- **THEN** display "? Unknown" in gray

#### Scenario: Unexpected merge status values
- **WHEN** mergeStateStatus has an unexpected value not covered above
- **THEN** display "? Unknown" in gray
