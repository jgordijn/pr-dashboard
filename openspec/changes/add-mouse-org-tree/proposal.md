# Change: Add mouse activation and organization-rooted repository trees

## Why
Mouse input is enabled but ignored, so clicking a PR does nothing. Repository grouping also flattens `owner/repository` into one node, obscuring the organization hierarchy users expect.

## What Changes
- Open the exact PR clicked with the left mouse button.
- Make organization and repository tree nodes clickable for focus/collapse without opening PRs.
- Render repository grouping as organization roots, repository children, and PR leaves.
- Add stable organization-node focus/collapse state and complete keyboard/mouse traversal semantics.
- Keep organization grouping and title-before-project PR rows intact.

## Impact
- Affected specs: `pr-display`, `tui-navigation`
- Affected code: `internal/tui`, main-program mouse wiring tests, README/help
- Related active changes: extends `add-repository-grouping` and the implemented status-row behavior.
