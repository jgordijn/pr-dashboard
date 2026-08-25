## ADDED Requirements
### Requirement: Sorting Controls
The system SHALL provide keyboard controls for changing persistent PR sorting.

#### Scenario: Cycle sort field
- **WHEN** `t` is pressed on the dashboard
- **THEN** sorting SHALL cycle through name, age, and state
- **AND** the new field SHALL persist

#### Scenario: Toggle sort direction
- **WHEN** `T` is pressed on the dashboard
- **THEN** direction SHALL toggle ascending/descending
- **AND** the new direction SHALL persist

### Requirement: View State Restoration
The system SHALL persist successful changes to restorable dashboard setup.

#### Scenario: Persist runtime setup
- **WHEN** grouping, display mode, draft visibility, focus, collapse, or sorting changes
- **THEN** the active account's UI state SHALL be saved

#### Scenario: Restore stale setup safely
- **WHEN** saved focus or collapse keys no longer exist
- **THEN** valid state SHALL restore
- **AND** focus SHALL fall back to the nearest visible item without failure
