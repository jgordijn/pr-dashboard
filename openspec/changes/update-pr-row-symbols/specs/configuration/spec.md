## ADDED Requirements
### Requirement: ASCII Display Mode
The system SHALL allow users to select a pure-ASCII status-symbol set.

#### Scenario: Default Unicode rendering
- **WHEN** `display.ascii` is absent or false
- **THEN** the dashboard SHALL use the Unicode symbol set

#### Scenario: Configured ASCII rendering
- **WHEN** `display.ascii` is true
- **THEN** the dashboard SHALL use only ASCII symbols for PR-list status communication
