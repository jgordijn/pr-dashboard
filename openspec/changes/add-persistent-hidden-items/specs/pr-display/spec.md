## ADDED Requirements
### Requirement: Hidden Item Projection
The system SHALL exclude hidden repositories and PRs consistently from every dashboard projection.

#### Scenario: Hide repository
- **WHEN** a repository rule exists
- **THEN** all current and future PRs in that repository SHALL be omitted from organization and repository views

#### Scenario: Hide individual PR
- **WHEN** an individual PR rule exists without a repository rule
- **THEN** only that PR SHALL be omitted

#### Scenario: Compose filters
- **WHEN** hidden and draft filters are active
- **THEN** empty organizations/repositories SHALL be omitted
- **AND** counts, navigation, layout, rollups, and mouse targets SHALL use the same filtered data

#### Scenario: Explain hidden-empty state
- **WHEN** fetched PRs exist but all are hidden
- **THEN** the dashboard SHALL explain that everything is hidden and provide the manager shortcut

#### Scenario: Show hidden summary
- **WHEN** the active account has hide rules
- **THEN** dashboard chrome SHALL show a compact rule count and manager hint
