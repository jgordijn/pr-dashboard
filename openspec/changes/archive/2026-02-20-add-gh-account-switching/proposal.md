## Why

Users with multiple GitHub accounts (e.g., personal and work) must currently quit the dashboard, run `gh auth switch`, and restart to view PRs for a different account. The `gh` CLI already supports multiple authenticated accounts, so the dashboard should let users switch between them without leaving the TUI.

## What Changes

- Add a TUI keybinding (e.g., `s`) that lists authenticated `gh` CLI accounts and lets the user pick one
- On account switch, invoke `gh auth switch --user <login>` to change the active `gh` account
- Update the in-memory username and re-fetch PRs for the newly active account
- Display the currently active account in the TUI header/status bar

## Capabilities

### New Capabilities
- `account-switching`: Detecting available `gh` CLI accounts, switching the active account, and refreshing the dashboard state after a switch

### Modified Capabilities
_(none — no existing specs)_

## Impact

- **Affected code**: `internal/config/auth.go` (account detection), `internal/tui/` (keybinding, account picker UI, header display), `internal/github/client.go` (re-create client after switch)
- **Dependencies**: Relies on `gh auth status` (list accounts) and `gh auth switch --user` (switch account) — no new external dependencies
- **Config**: The `username` field in `config.toml` continues to set the initial/default account; switching is a runtime-only action
