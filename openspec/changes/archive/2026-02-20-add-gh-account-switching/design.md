## Context

The `gh` CLI supports multiple authenticated accounts per host since v2.40. Users can add accounts with `gh auth login` and toggle with `gh auth switch --user <name>`. The `go-gh` library used by pr-dashboard automatically picks up whichever account is currently active.

Currently, `pr-dashboard` reads a single `username` from config and creates one `github.Client` at startup. Switching accounts requires quitting, running `gh auth switch`, and restarting.

## Goals / Non-Goals

**Goals:**
- Let users see which `gh` accounts are available and switch between them from inside the TUI
- After switching, re-fetch PRs for the newly active account
- Show the active account name in the status bar so it's always visible

**Non-Goals:**
- Managing `gh` authentication (login/logout) from inside the dashboard
- Supporting accounts on different GitHub hosts (e.g., GHES) — only `github.com`
- Persisting the switched account back to `config.toml` — switching is runtime-only

## Decisions

### 1. Account detection via `gh auth status`

Parse the output of `gh auth status --hostname github.com` to extract account logins and which one is active. The output format is stable and already used by the existing `CheckGHAuth` function.

**Alternative considered:** Read `~/.config/gh/hosts.yml` directly. Rejected because the YAML structure is an internal implementation detail of `gh` and could change without notice.

### 2. Account picker as a modal overlay

Reuse the existing modal pattern (`ModalState` + `ModalType`) to show a list of accounts. This keeps the UI consistent — the app already has help, success, and error modals. The picker will be a numbered list; pressing `1`, `2`, etc. selects an account.

**Alternative considered:** A full sub-view (replacing the PR list). Rejected as overkill for a 2–3 item list and inconsistent with the app's existing interaction patterns.

### 3. Keybinding: `s` for switch account

The `s` key is unused and mnemonic for "switch". It fits the existing single-key convention alongside `d` (drafts), `c` (cycle mode), `w` (watch).

### 4. Recreate GraphQL client after switch

After running `gh auth switch --user <name>`, create a new `github.Client` via `github.NewClient()`. The `go-gh` library reads the active token at client creation time, so a fresh client picks up the switched account's credentials. Update `Model.Client` and `Model.Config.General.Username` in memory.

**Alternative considered:** Pass a token explicitly to the client. Rejected because `go-gh` intentionally abstracts token management and the existing code relies on this.

### 5. Account switching happens in the `internal/config` package

Add `ListGHAccounts()` and `SwitchGHAccount(user string)` functions to `internal/config/auth.go`. This keeps all `gh` CLI interactions in one package and follows the existing pattern of `CheckGHCLI()` and `CheckGHAuth()`.

## Risks / Trade-offs

- **`gh auth status` output format change** → Low risk; the format has been stable. Mitigate with clear parsing tests and a fallback error message if parsing fails.
- **Race condition during switch** → The `gh auth switch` command and client recreation are sequential, so there's no race. If the switch command fails, the old client remains valid.
- **Single-account users see a no-op** → If only one account is detected, show a modal message ("Only one account available") instead of a picker. No confusion.
