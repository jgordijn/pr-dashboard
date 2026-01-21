## ADDED Requirements

### Requirement: Watch Mode
The system SHALL support automatic periodic refresh of PR data.

#### Scenario: Enable watch mode
- **WHEN** user presses 'w'
- **THEN** watch mode SHALL be enabled
- **AND** the status bar SHALL indicate watch mode is active with the refresh interval

#### Scenario: Disable watch mode
- **WHEN** user presses 'w' while watch mode is active
- **THEN** watch mode SHALL be disabled

#### Scenario: Periodic refresh
- **WHEN** watch mode is active
- **THEN** the system SHALL refresh data at the configured interval (default: 30 seconds)
- **AND** preserve the current selection by stable key during refresh

#### Scenario: Watch mode suspended during action
- **WHEN** watch mode is active and an action (update branch) is in progress
- **THEN** scheduled refreshes SHALL be suspended until the action completes
- **AND** a queued refresh SHALL run after the action completes

### Requirement: Change Detection
The system SHALL detect changes in PR data between refreshes.

#### Scenario: Detect status changes
- **WHEN** a PR's check status, review status, merge status, or unresolved count changes
- **THEN** the system SHALL mark that PR as changed

#### Scenario: Detect new PRs
- **WHEN** a new PR appears in the list
- **THEN** the system SHALL mark that PR as changed

#### Scenario: Ignore time-derived changes
- **WHEN** only the "days open" field would change (due to time passing)
- **THEN** the system SHALL NOT mark the PR as changed unless crossing a day boundary

### Requirement: Change Highlighting
The system SHALL visually highlight changed PRs when highlight_changes is enabled.

#### Scenario: Highlight changed PR
- **WHEN** a PR is marked as changed after refresh
- **THEN** the PR row SHALL be highlighted with a distinct color

#### Scenario: Clear highlight after timeout
- **WHEN** a PR has been highlighted for 2 seconds
- **THEN** the highlight SHALL be cleared and return to normal styling

### Requirement: Last Refresh Display
The system SHALL display the timestamp of the last successful refresh.

#### Scenario: Show last refresh time
- **WHEN** data is successfully refreshed
- **THEN** the status bar SHALL display "Last refresh: HH:MM"
