# Change: Persist dashboard view state and add sorting

## Why
Users currently rebuild their preferred dashboard setup after every restart and cannot reorder pull requests for different review tasks.

## What Changes
- Persist the active grouping, display density, draft visibility, focused item, and independent collapse maps per GitHub account.
- Add deterministic sorting by PR name, age, or state/severity.
- Add runtime sort-field cycling and ascending/descending direction toggle.
- Restore the saved setup on startup/account switch and display the active sort compactly.
- Persist every successful view-state change atomically across sessions.

## Impact
- Affected specs: `configuration`, `pr-display`, `tui-navigation`, `account-switching`
- Affected code: new session-state persistence package, TUI state/update/tree/view, startup wiring, docs and tests
