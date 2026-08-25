## MODIFIED Requirements
### Requirement: PR List Display
The system SHALL display open pull requests grouped by organization using a project-first, width-aware row.

#### Scenario: Display status-dense PR row
- **WHEN** a pull request row is rendered
- **THEN** it SHALL display `repository#number`, title, and a positional CI/review/merge status triad
- **AND** full and compact layouts SHALL include available age, author, thread, and reviewer context as width permits
- **AND** status fields SHALL remain visible when the title is truncated

#### Scenario: Display organization headers
- **WHEN** PRs are fetched from multiple organizations
- **THEN** PRs SHALL be grouped under collapsible organization headers with visible PR counts
- **AND** PRs within each organization SHALL be sorted by updated time descending
- **AND** a collapsed header SHALL summarize failing checks, behind branches, and unresolved threads

#### Scenario: Display draft PRs distinctly
- **WHEN** a PR is a draft
- **THEN** its row SHALL be dimmed and its merge slot SHALL display the draft symbol unless it has conflicts

#### Scenario: Preserve conflicts on drafts
- **WHEN** a draft PR has merge conflicts
- **THEN** its merge slot SHALL display the conflict symbol while the row remains dimmed

#### Scenario: Align status columns
- **WHEN** rows with different repository, title, author, and age lengths are rendered in one frame
- **THEN** their CI/review/merge triads SHALL begin in the same terminal column
- **AND** identity, title, author, and age fields SHALL be padded using measured display-cell widths

#### Scenario: Handle empty PR list
- **WHEN** the user has no open PRs
- **THEN** the system SHALL display `No open PRs - nice work! 🎉`

#### Scenario: Truncate safely
- **WHEN** row content exceeds the terminal width
- **THEN** flexible fields SHALL be truncated with a one-cell ellipsis without splitting a grapheme
- **AND** no rendered list line SHALL exceed the available width

### Requirement: Display Mode Cycling
The system SHALL support full, compact, and minimal display modes that share one status language.

#### Scenario: Full display mode
- **WHEN** full mode has enough width
- **THEN** repository identity, title, author, age, status triad, threads, and reviewer context SHALL be shown as space permits

#### Scenario: Compact display mode
- **WHEN** compact mode is selected or full mode degrades for width
- **THEN** repository identity, title, status triad, and available compact metadata SHALL be shown

#### Scenario: Minimal display mode
- **WHEN** minimal mode is selected or another mode degrades for width
- **THEN** repository identity, title, and the complete status triad SHALL remain visible

### Requirement: Merge Status Display
The system SHALL encode merge status in the third fixed status slot.

#### Scenario: Merge status symbols
- **WHEN** merge state is ready, behind, blocked, conflicts, unstable, unknown, or draft
- **THEN** it SHALL display respectively `✓`, `↓`, `⊘`, `≠`, `~`, `?`, or `○` in Unicode mode

#### Scenario: ASCII merge symbols
- **WHEN** ASCII mode is enabled
- **THEN** ready, behind, blocked, conflicts, unstable, unknown, and draft SHALL display respectively `=`, `v`, `#`, `X`, `~`, `?`, or `o`

## ADDED Requirements
### Requirement: Status Symbol Language
The system SHALL communicate CI, review, and merge states with fixed positional symbols that remain distinguishable without color.

#### Scenario: CI and review status symbols
- **WHEN** Unicode mode is active
- **THEN** CI passing/failing/pending/none SHALL use `✓/✗/◐/·`
- **AND** review approved/changes/required/none SHALL use `✓/!/?/·`

#### Scenario: Selected and changed rows
- **WHEN** a row is selected and/or recently changed
- **THEN** a two-cell gutter SHALL independently show `▶` for selected and `●` for changed

### Requirement: Contextual Status Decoder
The system SHALL explain the selected row's symbols in the status bar when width permits.

#### Scenario: Decode selected row
- **WHEN** a PR is selected and sufficient width is available
- **THEN** the status bar SHALL pair its symbols with canonical status words
- **AND** indicate the update action when the branch can be updated

### Requirement: ASCII Symbol Fallback
The system SHALL offer a pure-ASCII rendering with column geometry equivalent to Unicode mode.

#### Scenario: Render ASCII symbols
- **WHEN** `display.ascii` is true
- **THEN** all status, gutter, organization, and thread symbols SHALL be ASCII
