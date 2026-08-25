## ADDED Requirements
### Requirement: Grouping Projection Navigation
The system SHALL provide keyboard navigation for switching and traversing repository tree grouping.

#### Scenario: Toggle grouping
- **WHEN** the user presses `v`
- **THEN** the projection SHALL toggle between organization and repository grouping without a network request

#### Scenario: Traverse repository tree
- **WHEN** repository grouping is active
- **THEN** `j/k`, down/up, `gg`, and `G` SHALL traverse visible repository nodes and PR leaves

#### Scenario: Collapse repository with toggle
- **WHEN** `o` is pressed on an expanded repository node or one of its PR children
- **THEN** that repository SHALL collapse
- **AND** focus SHALL move to or remain on the repository node

#### Scenario: Navigate to and collapse parent
- **WHEN** left or `h` is pressed on a PR child
- **THEN** focus SHALL move to its repository node without collapsing it
- **AND** pressing left or `h` on that expanded repository node SHALL collapse it

#### Scenario: Expand repository
- **WHEN** right or `l` is pressed on a collapsed repository node
- **THEN** the repository SHALL expand
- **AND** pressing right or `l` again SHALL focus its first visible PR

#### Scenario: Toggle all repositories
- **WHEN** `O` is pressed in repository mode
- **THEN** all repository nodes SHALL expand if any is collapsed, otherwise all SHALL collapse

#### Scenario: Activate focused item
- **WHEN** Enter is pressed on a repository node
- **THEN** that node SHALL toggle collapse
- **AND** Enter on a PR SHALL retain the existing browser action

#### Scenario: Protect PR-only actions
- **WHEN** a repository node is focused
- **THEN** PR-only actions SHALL be no-ops
