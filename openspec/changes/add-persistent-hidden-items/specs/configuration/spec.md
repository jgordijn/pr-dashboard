## ADDED Requirements
### Requirement: Persistent Hidden Items State
The system SHALL persist account-scoped repository and PR hide rules separately from user configuration.

#### Scenario: Missing state file
- **WHEN** the hidden-state file does not exist
- **THEN** the system SHALL start with no hidden rules

#### Scenario: Save state atomically
- **WHEN** hidden rules change
- **THEN** the system SHALL atomically write versioned state with restrictive permissions

#### Scenario: Reject corrupt state
- **WHEN** hidden state is malformed or uses an unsupported version
- **THEN** the corrupt state SHALL not be overwritten
- **AND** the dashboard SHALL report that persistence is unavailable

#### Scenario: Scope by account
- **WHEN** multiple GitHub accounts use the dashboard
- **THEN** each account SHALL see and manage only its own hide rules
