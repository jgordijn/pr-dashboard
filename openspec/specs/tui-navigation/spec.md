## Purpose

Define keyboard-driven navigation, selection behavior, help, refresh, browser opening, and clean exit behavior for the dashboard.

## Requirements

### Requirement: Keyboard Navigation
The system SHALL support vim-style and arrow key navigation.

#### Scenario: Move selection down
- **WHEN** user presses 'j' or down arrow
- **THEN** the selection SHALL move to the next PR in the list

#### Scenario: Move selection up
- **WHEN** user presses 'k' or up arrow
- **THEN** the selection SHALL move to the previous PR in the list

#### Scenario: Jump to top
- **WHEN** user presses 'gg'
- **THEN** the selection SHALL move to the first PR in the list

#### Scenario: Jump to bottom
- **WHEN** user presses 'G'
- **THEN** the selection SHALL move to the last PR in the list

#### Scenario: Skip collapsed organizations
- **WHEN** navigating and an organization is collapsed
- **THEN** navigation SHALL skip over the collapsed section

### Requirement: Selection Preservation
The system SHALL preserve selection across data refreshes using stable identifiers.

#### Scenario: Preserve selection by stable key
- **WHEN** data is refreshed
- **THEN** the previously selected PR SHALL remain selected by matching its stable key (owner/repo#number)

#### Scenario: Handle selected PR disappearing
- **WHEN** the selected PR is no longer in the data (merged/closed)
- **THEN** selection SHALL move to the nearest visible PR

#### Scenario: Handle selection hidden by filter
- **WHEN** the selected PR becomes hidden (draft toggle, org collapse)
- **THEN** selection SHALL move to the nearest visible PR

### Requirement: Organization Collapse Toggle
The system SHALL allow collapsing and expanding organization sections.

#### Scenario: Toggle current organization
- **WHEN** user presses 'o' with selection in an organization
- **THEN** that organization section SHALL toggle between collapsed and expanded

#### Scenario: Toggle all organizations
- **WHEN** user presses 'O' (shift+o)
- **THEN** all organization sections SHALL toggle their collapsed state

### Requirement: Open PR in Browser
The system SHALL allow opening the selected PR in a web browser.

#### Scenario: Open selected PR
- **WHEN** user presses Enter on a selected PR
- **THEN** the system SHALL open the PR URL in the default browser using `gh pr view --web`

### Requirement: Manual Refresh
The system SHALL allow manual data refresh.

#### Scenario: Trigger refresh
- **WHEN** user presses 'r'
- **THEN** the system SHALL fetch fresh data from GitHub
- **AND** display a loading indicator during refresh
- **AND** preserve the current selection by stable key

#### Scenario: Refresh blocked during action
- **WHEN** user presses 'r' while an action (update branch) is in progress
- **THEN** the refresh SHALL be queued until the action completes

### Requirement: Help Display
The system SHALL provide a help screen showing all keybindings.

#### Scenario: Show help
- **WHEN** user presses '?'
- **THEN** a modal SHALL display listing all available keybindings

#### Scenario: Dismiss help
- **WHEN** help modal is displayed and user presses 'q', Escape, or Enter
- **THEN** the help modal SHALL close

### Requirement: Application Exit
The system SHALL allow clean application exit.

#### Scenario: Quit application
- **WHEN** user presses 'q' or Escape (with no modal open)
- **THEN** the application SHALL exit gracefully
