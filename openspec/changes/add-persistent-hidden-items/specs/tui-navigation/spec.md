## ADDED Requirements
### Requirement: Hide Focused Item
The system SHALL support reversible hiding of focused repositories and PRs.

#### Scenario: Hide repository or PR
- **WHEN** `H` is pressed on a repository node or PR leaf
- **THEN** the exact rule SHALL persist
- **AND** the item SHALL disappear from the dashboard
- **AND** focus SHALL move to the nearest spatially stable visible item

#### Scenario: Reject organization hide
- **WHEN** `H` is pressed on an organization node or invalid focus
- **THEN** nothing SHALL be hidden
- **AND** actionable guidance SHALL be shown

#### Scenario: Undo latest hide
- **WHEN** `z` is pressed after a successful hide in the current process
- **THEN** exactly that newly created rule SHALL be restored and persisted

#### Scenario: Roll back save failure
- **WHEN** persistence fails during hide, undo, or unhide
- **THEN** visible state SHALL remain unchanged and an error SHALL be reported

### Requirement: Hidden Items Manager
The system SHALL provide a full-screen manager for reviewing and selectively restoring hidden rules.

#### Scenario: Open and close manager
- **WHEN** `M` is pressed
- **THEN** the manager SHALL open without changing dashboard focus
- **AND** `q` or Escape SHALL return to a valid dashboard focus

#### Scenario: Browse hidden rules
- **WHEN** rules exist
- **THEN** repositories and PRs SHALL render newest-first with type, identity, and saved context
- **AND** `j/k`, arrows, `gg/G` SHALL navigate results

#### Scenario: Search and filter
- **WHEN** `/` search or Tab type filtering is used
- **THEN** results SHALL update case-insensitively while preserving nearest selection

#### Scenario: Unhide focused rule
- **WHEN** `u` or Enter is pressed on a manager row
- **THEN** exactly that rule SHALL be removed and persisted
- **AND** the manager SHALL remain open with stable selection
