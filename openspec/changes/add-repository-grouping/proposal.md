# Change: Add optional repository tree grouping

## Why
The organization-grouped dashboard is useful for a flat scan, but larger organizations need a repository-oriented tree to reveal project boundaries. Users also need to switch back to the current view without restarting or changing configuration.

## What Changes
- Add `display.grouping = "organization" | "repository"`, defaulting to the current organization view.
- Add a runtime `v` view toggle that preserves the focused PR where possible.
- Render focusable, collapsible `owner/repository` nodes in repository mode.
- Put repository-mode PR identity in the order `#number title repository`, so the project follows the title.
- Add tree-aware navigation, collapse behavior, responsive alignment, ASCII connectors, and collapsed-node status rollups.

## Impact
- Affected specs: `configuration`, `pr-display`, `tui-navigation`
- Affected code: `internal/config`, `internal/model`, `internal/tui`, README and example configuration
- Related active change: `update-pr-row-symbols`; this change extends its implemented symbol rows without replacing their requirements.
