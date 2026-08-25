## ADDED Requirements
### Requirement: Account-Scoped UI Setup
The system SHALL maintain independent saved dashboard setups per GitHub account.

#### Scenario: Switch account
- **WHEN** account switching succeeds
- **THEN** the destination account's saved setup SHALL restore before its PR projection renders

#### Scenario: Switch back
- **WHEN** returning to an account with saved UI state
- **THEN** its previous grouping, display, drafts, sort, focus, and collapse setup SHALL be restored
