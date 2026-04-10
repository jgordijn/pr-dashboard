# Design: PR Dashboard TUI

## Context

`pr-dashboard` is a terminal UI for monitoring pull requests authored by a GitHub user across one or more organizations. The implementation needs to present rich PR state in a compact interface, refresh without disrupting user context, and reuse the user's existing `gh` CLI authentication instead of introducing a separate credential flow.

Key constraints:

- Interactive terminal required
- GitHub authentication must come from `gh`
- Refreshes must be safe during long-running actions such as branch updates
- Configuration should be simple for end users and easy to validate

## Goals / Non-Goals

### Goals

- Display PRs grouped by organization with keyboard navigation
- Show actionable status information in full, compact, and minimal modes
- Support watch mode with change highlighting and selection preservation
- Allow updating a behind branch from the TUI
- Provide a setup wizard when configuration is missing
- Support switching between authenticated GitHub accounts without restarting

### Non-Goals

- Supporting GitHub Enterprise hosts
- Editing PR metadata from the dashboard
- Replacing the `gh` CLI authentication model
- Persisting complex UI session state across restarts

## Architecture Overview

The application is split into four main areas:

- `internal/config`: load, validate, and generate configuration; interact with `gh` auth state
- `internal/github`: GraphQL data retrieval and branch-update actions
- `internal/model`: transform API responses into view-friendly domain models and diffs
- `internal/tui`: Bubble Tea model, update loop, rendering, modal flows, and watch mode

`cmd/pr-dashboard/main.go` performs startup checks, loads configuration, creates a GitHub client, and launches the Bubble Tea program.

## Decisions

### Use the Charm stack for the UI

Bubble Tea provides a predictable model-update-view architecture for handling refreshes, modal state, keyboard input, and asynchronous actions. Lip Gloss is used for styling and Bubbles provides reusable primitives such as the spinner.

### Use GitHub GraphQL for data retrieval

The dashboard needs repository, review, thread, check, and merge state information at once. GraphQL allows the app to fetch the required fields efficiently and paginate through large PR result sets.

### Keep selection stable with PR keys

Selection is tracked by stable keys in the form `owner/repo#number` rather than list indexes. This allows refreshes, filtering, and organization collapse state changes without unexpectedly moving the user's focus.

### Separate GitHub transport models from display models

Raw GitHub responses are transformed into domain models that compute derived values such as days open, unresolved-thread counts, aggregate check status, and merge state labels. This keeps TUI rendering logic simpler and easier to test.

### Queue refreshes during actions

The update-branch action can take time and should not race with background refreshes. While an action is in progress, scheduled and manual refreshes are queued and executed immediately after the action completes.

### Recreate clients after account switching

Because the active `gh` account can change while the process is running, account switching recreates the GitHub client with a fresh token for the selected login. This avoids stale authentication state in long-lived clients.

## Data Flow

1. Startup validates TTY, terminal size, `gh` installation, and `gh` authentication.
2. Configuration is loaded from disk or created with the setup wizard.
3. The app obtains an auth token for the configured user and creates a GitHub client.
4. Bubble Tea initializes and triggers the first PR fetch.
5. API results are transformed into grouped domain models.
6. The TUI renders grouped PRs, status bar state, and modal overlays.
7. Refreshes, branch updates, and account switches feed back into the same update loop through typed messages.

## Risks / Trade-offs

- Depending on `gh` keeps auth simple, but limits behavior to environments where `gh` is installed and configured.
- GraphQL responses can become expensive if queries grow; rate-limit feedback is surfaced in the status bar to make this visible.
- Watch mode improves responsiveness but increases request volume; the refresh interval is configurable and change highlighting is lightweight.
- Manual branch updates rely on external command execution and can fail for repo-specific reasons; modal feedback is used to make failures explicit.

## Validation Approach

- Unit tests cover config loading/validation, auth helpers, GraphQL client behavior, domain transforms, change detection, and TUI state transitions.
- `go test`, `go vet`, and `staticcheck` are used as automated quality gates.
- Manual smoke testing is documented in the project README for interactive scenarios that depend on live GitHub state.
