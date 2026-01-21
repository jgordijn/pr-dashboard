## ADDED Requirements

### Requirement: Update Branch Action
The system SHALL allow updating a PR branch to be current with its base branch.

#### Scenario: Update available branch
- **WHEN** user presses 'u' on a PR with mergeStateStatus=BEHIND and mergeable=MERGEABLE
- **THEN** the system SHALL execute `gh pr update-branch <number> --repo <owner/name>`
- **AND** display a loading indicator during the operation
- **AND** suspend watch mode refreshes until completion

#### Scenario: Update success
- **WHEN** the branch update succeeds
- **THEN** a modal SHALL display "Branch updated successfully" with success styling
- **AND** the PR data SHALL be refreshed

#### Scenario: Update failure
- **WHEN** the branch update fails
- **THEN** a modal SHALL display the error message with error styling

#### Scenario: Update not available - conflicts
- **WHEN** user presses 'u' on a PR with mergeable=CONFLICTING
- **THEN** the system SHALL display a modal: "Cannot update: PR has merge conflicts"

#### Scenario: Update not available - unknown merge status
- **WHEN** user presses 'u' on a PR with mergeable=UNKNOWN
- **THEN** the system SHALL display a modal: "Cannot update: merge status unknown (GitHub is still computing)"

#### Scenario: Update not available - already current
- **WHEN** user presses 'u' on a PR with mergeStateStatus=CLEAN
- **THEN** the system SHALL display a modal: "Branch is already up to date"

#### Scenario: Update not available - blocked
- **WHEN** user presses 'u' on a PR with mergeStateStatus=BLOCKED
- **THEN** the system SHALL display a modal: "Cannot update: merge is blocked"

### Requirement: Action Concurrency Control
The system SHALL prevent concurrent actions and manage refresh scheduling.

#### Scenario: Block actions during action
- **WHEN** an action (update branch) is in progress
- **THEN** pressing 'u' on another PR SHALL be ignored

#### Scenario: Block refresh during action
- **WHEN** an action is in progress and watch mode attempts to refresh
- **THEN** the refresh SHALL be queued for after action completion

#### Scenario: Resume after action
- **WHEN** an action completes
- **THEN** any queued refresh SHALL run immediately

### Requirement: Modal Feedback
The system SHALL use modal dialogs to display action results.

#### Scenario: Success modal
- **WHEN** an action succeeds
- **THEN** a modal with success styling (green) SHALL display the result

#### Scenario: Error modal
- **WHEN** an action fails
- **THEN** a modal with error styling (red) SHALL display the error details

#### Scenario: Dismiss modal
- **WHEN** a modal is displayed and user presses Enter, 'q', or Escape
- **THEN** the modal SHALL close
