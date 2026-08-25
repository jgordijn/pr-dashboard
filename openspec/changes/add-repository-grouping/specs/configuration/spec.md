## ADDED Requirements
### Requirement: Initial Grouping Projection
The system SHALL allow selecting the initial pull-request grouping projection.

#### Scenario: Preserve current default
- **WHEN** `display.grouping` is omitted
- **THEN** the system SHALL use `organization` grouping

#### Scenario: Configure repository grouping
- **WHEN** `display.grouping` is `repository`
- **THEN** the system SHALL start in repository tree mode

#### Scenario: Reject invalid grouping
- **WHEN** `display.grouping` is neither `organization` nor `repository` after case and whitespace normalization
- **THEN** validation SHALL report a targeted configuration error
