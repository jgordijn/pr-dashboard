## ADDED Requirements
### Requirement: Organization-Rooted Repository Tree
The repository grouping projection SHALL render organizations as top-level nodes containing repository nodes and PR leaves.

#### Scenario: Render three-level hierarchy
- **WHEN** repository grouping is active
- **THEN** each organization with visible PRs SHALL render as a top-level organization node
- **AND** each repository SHALL render beneath its organization
- **AND** each PR SHALL render beneath its repository
- **AND** repository-mode PR rows SHALL continue to place the repository after the PR title

#### Scenario: Preserve duplicate repository names
- **WHEN** different organizations contain repositories with the same name
- **THEN** each repository SHALL remain beneath its owning organization with collision-proof focus keys

#### Scenario: Respect filtering and collapse
- **WHEN** drafts are filtered or organization/repository nodes are collapsed
- **THEN** counts, visible descendants, and risk rollups SHALL reflect only applicable visible PR data

#### Scenario: Render ASCII hierarchy
- **WHEN** ASCII mode is active
- **THEN** organization and repository nodes, connectors, fills, and PR leaves SHALL remain pure ASCII and within terminal width
