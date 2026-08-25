## ADDED Requirements
### Requirement: Repository Tree Grouping
The system SHALL provide a repository tree projection in addition to the existing organization-grouped projection.

#### Scenario: Render organization and repository nodes
- **WHEN** repository grouping is active
- **THEN** each organization with visible pull requests SHALL appear as a focusable top-level node
- **AND** each repository SHALL appear as a focusable child node beneath its organization
- **AND** visible pull requests SHALL appear as leaves beneath their repository
- **AND** organizations and repositories SHALL sort case-insensitively with deterministic raw-key tie-breakers
- **AND** PR children SHALL sort by updated time descending with a stable-key tie-breaker

#### Scenario: Put project after title
- **WHEN** a repository-mode PR leaf is rendered
- **THEN** its information order SHALL begin `#number title repository`
- **AND** the repository SHALL not precede the title

#### Scenario: Preserve aligned status scanning
- **WHEN** repository-mode leaves with different content lengths render in one frame
- **THEN** title, repository, author, and age fields SHALL use measured frame-wide widths
- **AND** every status triad SHALL begin in the same terminal column
- **AND** no tree line SHALL exceed terminal width

#### Scenario: Render collapsed node risk
- **WHEN** an organization or repository node is collapsed
- **THEN** it SHALL summarize failing checks, behind branches, and unresolved threads using the active symbol language

#### Scenario: Respect draft filtering
- **WHEN** drafts are hidden and a repository has no remaining visible PRs
- **THEN** its repository node SHALL be omitted

#### Scenario: Render ASCII tree
- **WHEN** ASCII display mode and repository grouping are active
- **THEN** repository indicators, connectors, fill, rows, and rollups SHALL contain only ASCII symbols

### Requirement: Runtime Grouping Toggle
The system SHALL allow toggling between organization and repository projections without refetching data.

#### Scenario: Preserve PR focus
- **WHEN** grouping is toggled while a visible PR is focused
- **THEN** that PR SHALL remain focused

#### Scenario: Leave repository-node focus
- **WHEN** grouping changes to organization mode while a repository node is focused
- **THEN** focus SHALL move to that repository's first visible PR or the nearest visible PR

#### Scenario: Preserve independent collapse state
- **WHEN** grouping is toggled or data is refreshed
- **THEN** repository and organization collapse state SHALL remain independent and be preserved for still-existing groups
