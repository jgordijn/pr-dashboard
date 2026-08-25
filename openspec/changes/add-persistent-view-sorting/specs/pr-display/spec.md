## ADDED Requirements
### Requirement: PR Sorting
The system SHALL sort PR leaves within their parent by a selectable field and direction.

#### Scenario: Sort by name
- **WHEN** name sorting is active
- **THEN** PR titles SHALL compare case-insensitively with stable-key tie-breaking

#### Scenario: Sort by age
- **WHEN** age sorting is active
- **THEN** PR creation age SHALL determine order with stable-key tie-breaking

#### Scenario: Sort by state
- **WHEN** state sorting is active
- **THEN** PRs SHALL order by documented readiness severity derived from merge, review, CI, and draft state
- **AND** equally ranked PRs SHALL use stable-key tie-breaking

#### Scenario: Apply direction
- **WHEN** direction is toggled
- **THEN** ascending or descending order SHALL apply without changing organization/repository node ordering

#### Scenario: Show active sorting
- **WHEN** the dashboard renders
- **THEN** compact chrome SHALL identify the active sort field and direction
