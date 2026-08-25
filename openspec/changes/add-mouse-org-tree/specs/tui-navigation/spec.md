## ADDED Requirements
### Requirement: Mouse Activation
The system SHALL support precise left-click activation of visible dashboard items.

#### Scenario: Click PR
- **WHEN** the user presses the left mouse button on a visible PR row
- **THEN** that exact PR SHALL become focused
- **AND** its existing browser-open command SHALL execute

#### Scenario: Click tree node
- **WHEN** the user left-clicks an organization or repository node
- **THEN** that exact node SHALL become focused and toggle its collapse state
- **AND** no PR browser command SHALL execute

#### Scenario: Ignore inert mouse events
- **WHEN** a mouse event is not a left-button press or targets header, blank, status, loading, error, empty, modal, or out-of-bounds content
- **THEN** application state SHALL remain unchanged and no command SHALL execute

#### Scenario: Track rendered rows exactly
- **WHEN** grouping, collapse, or draft visibility changes
- **THEN** mouse hit targets SHALL match the currently rendered item on each terminal row

### Requirement: Three-Level Tree Navigation
The system SHALL support keyboard navigation across organization nodes, repository nodes, and PR leaves in repository mode.

#### Scenario: Traverse all levels
- **WHEN** repository mode is active
- **THEN** vertical navigation SHALL traverse visible organizations, repositories, and PRs in render order

#### Scenario: Navigate parent and child
- **WHEN** left or `h` is pressed on a PR, repository, or organization node
- **THEN** focus/collapse SHALL move toward its logical parent
- **AND** right or `l` SHALL expand or move toward its first visible child

#### Scenario: Preserve independent collapse
- **WHEN** view toggles or data refreshes
- **THEN** organization-tree and repository collapse state SHALL remain independently keyed and preserved
