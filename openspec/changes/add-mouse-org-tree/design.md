## Context
Bubble Tea already starts with `WithMouseCellMotion`, but `Update` does not handle `tea.MouseMsg`. Repository mode currently projects exact owner/repository pairs as flat root nodes.

## Goals / Non-Goals
- Goals: reliable left-click PR activation, organization-rooted repository trees, stable focus/collapse, exact hit testing across filters/collapse/modes, keyboard parity, Unicode/ASCII width safety.
- Non-goals: hover styling, right-click menus, drag handling, or changing GitHub API data.

## Decisions
- Repository mode uses three focusable levels: `org:<owner>`, `repo:<owner>/<repo>`, and existing PR keys.
- Organization collapse inside repository mode uses its own stable map, independent from organization-view collapse and repository collapse.
- Hit targets are derived from the same visible ordering rules as rendering. Terminal row zero is the dashboard header; content targets begin at row one. Blank/status/modal/loading/error rows are inert.
- Only a left-button press activates. A PR click focuses it and immediately runs the existing browser command. Node clicks focus and toggle that exact node.
- Organization mode headers are clickable collapse targets; its PR rows open exactly as in repository mode.
- Repository mode renders organization roots, indented repository nodes, then indented PR leaves. Counts and rollups respect draft filtering.
- Keyboard left/right and toggle semantics extend naturally across all three tree levels.

## Risks / Trade-offs
- Rendering and hit-testing can drift. Shared ordered target helpers and golden row-coordinate tests prevent divergence.
- Browser commands have external side effects. Unit tests assert target selection and non-nil commands without executing them; command construction remains the existing tested path.

## Migration Plan
No configuration or data migration. Existing keyboard behavior remains available, and terminals without mouse support continue to work normally.
