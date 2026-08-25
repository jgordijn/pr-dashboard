# Change: Add persistent project and PR hiding

## Why
Users need to remove irrelevant repositories or individual pull requests from the dashboard without losing the ability to review and restore those exclusions later.

## What Changes
- Hide the focused repository or PR with a deliberate `H` shortcut.
- Persist account-scoped hide rules across refreshes and restarts.
- Add `z` to undo the most recent hide from the current session.
- Add a polished `M` Hidden Items manager for browsing, searching, filtering, and selectively restoring rules.
- Apply hidden filtering consistently before grouping, counts, navigation, rendering, status rollups, and mouse hit testing.

## Impact
- Affected specs: `configuration`, `pr-display`, `tui-navigation`, `account-switching`
- Affected code: new `internal/hidden`, `internal/tui`, startup wiring, README and CLI help
- Related active changes: extends repository grouping and mouse target projections without changing their identity formats.
