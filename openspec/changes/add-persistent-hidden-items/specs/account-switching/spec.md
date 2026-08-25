## ADDED Requirements
### Requirement: Account-Scoped Hidden Visibility
The system SHALL switch hidden-item projections with the active GitHub account.

#### Scenario: Switch account
- **WHEN** account switching succeeds
- **THEN** the dashboard SHALL immediately use the destination account's hide rules
- **AND** the hidden manager SHALL reset its query, filter, and selection

#### Scenario: Switch back
- **WHEN** the user returns to a previously used account
- **THEN** that account's persisted hide rules SHALL be restored
