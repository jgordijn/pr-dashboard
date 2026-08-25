## ADDED Requirements
### Requirement: Persistent UI State
The system SHALL persist account-scoped dashboard view state independently from user configuration.

#### Scenario: Restore saved setup
- **WHEN** valid saved UI state exists for the active account
- **THEN** grouping, display mode, draft visibility, sort settings, focus, and collapse maps SHALL be restored

#### Scenario: Missing state
- **WHEN** no UI-state file or account entry exists
- **THEN** configuration defaults SHALL initialize the view

#### Scenario: Save atomically
- **WHEN** a persisted view setting changes
- **THEN** versioned state SHALL be written atomically with restrictive permissions

#### Scenario: Corrupt state
- **WHEN** UI state is malformed or unsupported
- **THEN** it SHALL not be overwritten silently
- **AND** configuration defaults SHALL apply with a visible persistence warning
