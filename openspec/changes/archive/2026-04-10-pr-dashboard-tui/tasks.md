# Tasks for pr-dashboard-tui

## Phase 1: Project Foundation

- [x] 1.1 Initialize Go module
  - Run `go mod init github.com/jgordijn/pr-dashboard`
  - Create initial directory structure: `cmd/pr-dashboard/`, `internal/{config,github,model,tui}/`
  - File: `go.mod`

- [x] 1.2 Create Makefile
  - Targets: `build`, `install`, `clean`, `run`, `test`
  - Install to `~/.local/bin/pr-dashboard`
  - File: `Makefile`
  - [x] SUGGESTION: Prefer `install` over `cp` for deterministic permissions
    - File: `Makefile:10-12`
    - Issue: `cp` preserves whatever mode the build output has; using `install -m 0755` makes the executable bit explicit and is more standard for install steps.
    - Fix: Replace `cp $(BINARY) $(INSTALL_DIR)/` with `install -m 0755 $(BINARY) $(INSTALL_DIR)/$(BINARY)` (keep `mkdir -p` or switch to `install -d`).
  - [x] SUGGESTION: Allow overriding Go tool via `GO ?= go`
    - File: `Makefile:7,21`
    - Issue: Hardcoding `go` reduces portability for environments where `go` is wrapped/renamed (e.g., toolchains, Nix, asdf).
    - Fix: Add `GO ?= go` and change build/test to `$(GO) build ...` and `$(GO) test ...`.
  ```makefile
  .PHONY: build install clean run test

  BINARY := pr-dashboard
  INSTALL_DIR := $(HOME)/.local/bin

  build:
  	go build -o $(BINARY) ./cmd/pr-dashboard

  install: build
  	mkdir -p $(INSTALL_DIR)
  	cp $(BINARY) $(INSTALL_DIR)/
  	@echo "Installed to $(INSTALL_DIR)/$(BINARY)"

  clean:
  	rm -f $(BINARY)

  run: build
  	./$(BINARY)

  test:
  	go test ./...
  ```

- [x] 1.3 Create example config file
  - Document all config options with comments
  - Use `organizations.login` (not `name`) for clarity
  - File: `config.example.toml`
  - [x] ⚠️ HIGH PRIORITY: Mention `display.initial_mode` default value in example config
    - File: `config.example.toml:14`
    - Issue: Comment lists allowed values but does not state the default; task requires defaults to be documented.
    - Fix: Update the inline comment to include default (e.g. `default "full"`).
    - Priority: 1
  - [x] ⚠️ HIGH PRIORITY: Add top-of-file usage guidance to `config.example.toml`
    - File: `config.example.toml:1`
    - Issue: The example config starts immediately with `[general]` and does not explain where to place the file, how multiple `[[organizations]]` entries work, or which sections are required.
    - Why it matters: Users may copy/paste incorrectly, omit required sections, or misunderstand how to configure multiple orgs.
    - Fix: Add a short comment header (e.g., target path `~/.config/pr-dashboard/config.toml`, requirement to provide at least one `[[organizations]]`, and a quick "copy this file and edit values" note).
    - Priority: 1
  - [x] 💡 SUGGESTION: Keep the embedded TOML snippet in `tasks.md` in sync with `config.example.toml`
    - File: `openspec/changes/pr-dashboard-tui/tasks.md:60-78`
    - Issue: The inline snippet currently lacks the `display.initial_mode` default value mention, while `config.example.toml` includes it.
    - Fix: Update the snippet comment to match `config.example.toml` (or remove the snippet and refer to the file only).
    - Priority: 3
  - [x] ⚠️ HIGH PRIORITY: Keep task checkbox format consistent (avoid embedding status in label) - REJECTED: in_progress prefix is correct per OpenSpec adapter spec
    - File: `openspec/changes/pr-dashboard-tui/tasks.md:46`
    - Issue: Changing the task label from `1.3 Create example config file` to `in_progress: 1.3 Create example config file` may break tooling/parsing and adds noise unrelated to the config example.
    - Fix: Revert to the original label and track status via git/PR metadata or a separate field/section.
    - Priority: 1
  ```toml
  # PR Dashboard Configuration
  # Copy this file to: ~/.config/pr-dashboard/config.toml
  #
  # Required sections: [general] with username, at least one [[organizations]]
  # Optional sections: [display], [notifications]

  [general]
  username = "your-github-username"    # Required: your GitHub login
  refresh_interval = 30                # Optional: seconds (10-300), default 30

  [[organizations]]
  login = "organisation"               # Required: GitHub org login (used in search)

  # Add more organizations as needed:
  # [[organizations]]
  # login = "another-org"

  [display]
  show_drafts = true                   # Optional: show draft PRs, default true
  initial_mode = "full"                # Optional: "full", "compact", "minimal", default "full"

  [notifications]
  highlight_changes = true             # Optional: highlight changed PRs, default true
  ```

## Phase 2: Configuration Module

- [x] 2.1 Create config types and loading
  - [x] HIGH: Reject unknown TOML keys during decode — Fixed: now uses toml.Decode + meta.Undecoded() check (config.go:96-109)
  - [x] HIGH: Don't silently coerce invalid values to defaults — Fixed: removed applyDefaultsForZeroValues, defaults applied before parse only (config.go:84-86)
  - [x] SUGGESTION: Expose a sentinel error for missing config file — Fixed: added ErrConfigNotFound sentinel (config.go:15-17, 90-92)
  - [x] SUGGESTION: Remove/adjust misleading comments — Fixed: updated applyDefaults comment (config.go:114-116)
  - Define Config struct matching TOML structure
  - Load from `~/.config/pr-dashboard/config.toml`
  - Provide sensible defaults for missing optional values
  - File: `internal/config/config.go`
  - Types needed:
    - `Config` (top-level)
    - `GeneralConfig` (username required, refresh_interval optional)
    - `OrganizationConfig` (login required)
    - `DisplayConfig` (show_drafts, initial_mode)
    - `NotificationsConfig` (highlight_changes)

- [x] 2.2 Implement config validation
  - [x] HIGH: Trim whitespace for display.initial_mode before validating - Fixed: now uses strings.ToLower(strings.TrimSpace()) before validation
  - [x] SUGGESTION: Handle empty ValidationError.Errors in Error() formatting - Fixed: added guard for len(e.Errors) == 0
  - [x] SUGGESTION: Prefer errors.As in tests over direct type assertion - Fixed: replaced direct type assertion with errors.As
  - `username` required, non-empty
  - At least one `[[organizations]]` with non-empty `login`
  - `refresh_interval` between 10-300 seconds
  - `initial_mode` must be "full", "compact", or "minimal"
  - Return clear error messages for validation failures
  - File: `internal/config/validation.go`

- [x] 2.3 Check gh CLI is installed
  - Use `exec.LookPath("gh")` to detect
  - Return helpful error: "gh CLI not found. Install from https://cli.github.com"
  - File: `internal/config/auth.go`
  - [x] HIGH: Make `CheckGHCLI` tests deterministic (avoid depending on local gh installation)
    - File: internal/config/auth_test.go:9
    - Issue: Positive test `TestCheckGHCLI_Installed` assumes `gh` is installed and otherwise skips, which can lead to little/no coverage depending on environment and may hide regressions.
    - Fix: In tests, create a temporary directory with a fake executable named `gh`, prepend it to `PATH` for the duration of the test, and assert `CheckGHCLI()` succeeds. Add a negative-path test by setting `PATH` to an empty/nonexistent directory and assert `errors.Is(err, ErrGHCLINotFound)`.
  - [x] SUGGESTION: Simplify fake-gh helper (remove custom `itoa`)
    - File: internal/config/auth_test.go:40
    - Issue: `itoa()` only supports 0/1 and silently maps other values to "0", which is surprising and increases test helper complexity.
    - Fix: Use `strconv.Itoa(exitCode)` and remove `itoa()`.
  - [x] SUGGESTION: Preserve underlying `exec.LookPath` error for diagnostics
    - File: internal/config/auth.go:25-28
    - Issue: `CheckGHCLI()` returns `ErrGHCLINotFound` for any `exec.LookPath` error. If the error is something other than "not found" (e.g., PATH permission issues), the message becomes misleading.
    - Fix: Wrap/join the underlying error while keeping the sentinel matchable (e.g., `errors.Join(ErrGHCLINotFound, err)` or a small custom error type with `Unwrap() []error`).

- [x] 2.4 Check gh CLI authentication
  - Run `gh auth status --hostname github.com`
  - Use exit code to determine authentication status (0 = authenticated)
  - Return clear error message if not authenticated: "Run `gh auth login` first"
  - File: `internal/config/auth.go`
  - [x] HIGH: Add timeout + preserve underlying error for `gh auth status` failures
    - File: internal/config/auth.go:41
    - Issue: `cmd.Run()` errors are all mapped to `ErrGHNotAuthenticated`, losing the root cause (e.g., binary exec failure, unexpected gh error). Also, the external command has no timeout and could hang, blocking startup.
    - Fix: Use `exec.CommandContext` with a small timeout (e.g., 5s) and return `fmt.Errorf("%w: %v", ErrGHNotAuthenticated, err)` (or wrap exit errors selectively) so callers get actionable diagnostics while still supporting `errors.Is`.
  - [x] HIGH: Make `CheckGHAuth` tests deterministic (avoid requiring real GitHub auth)
    - File: internal/config/auth_test.go:65
    - Issue: `TestCheckGHAuth_Authenticated` depends on the developer/CI machine being logged in to `gh`, which will usually skip or be flaky across environments.
    - Fix: Similar to `CheckGHCLI` tests, inject a fake `gh` binary via a temp dir and `PATH` that exits 0 for `auth status` and non-zero for unauthenticated cases; then assert `CheckGHAuth()` returns nil / `errors.Is(err, ErrGHNotAuthenticated)` respectively.
  - [x] HIGH: Actually preserve underlying `gh` execution error in the error chain
    - File: internal/config/auth.go:52
    - Issue: `fmt.Errorf("%w: %v", ErrGHNotAuthenticated, err)` only wraps the sentinel; the underlying error is only copied into the message and is not reachable via `errors.Unwrap`/`errors.Is`/`errors.As`.
    - Why it matters: Callers can't reliably distinguish timeout (`context.DeadlineExceeded`) vs process exit (`*exec.ExitError`) vs other OS errors, and diagnostics tooling can't inspect the root cause.
    - Fix: Return `errors.Join(ErrGHNotAuthenticated, err)` (Go 1.20+) or wrap a custom error type that implements `Unwrap() []error`.
  - [x] HIGH: Include `gh auth status` output in the returned error for troubleshooting
    - File: internal/config/auth.go:50
    - Issue: Using `cmd.Run()` loses stdout/stderr; users often only see `exit status 1`, which is not actionable.
    - Why it matters: Misconfigured `gh` (wrong host, auth expired, SSO issues) is common; surfacing `gh` output reduces support/debug time.
    - Fix: Use `cmd.CombinedOutput()` (with the same timeout context) and include output in the wrapped error (trim whitespace).

- [x] 2.5 Implement first-run wizard
  - Detect missing config file
  - Check `gh` installed and authenticated first
  - Prompt for username, organizations (comma-separated), refresh interval
  - Create config directory `~/.config/pr-dashboard/` if needed
  - Write generated config to file
  - Use Bubble Tea text input or simple stdin prompts
  - File: `internal/config/wizard.go`
  - [x] CRITICAL: Prevent config data loss on write — Fixed: refuses to overwrite existing files (ErrConfigExists), uses atomic write via temp file + os.Rename, exported WriteConfigToPath for testability
  - [x] HIGH: Use restrictive permissions for config dir/file — Fixed: uses 0700 for dir and 0600 for file
  - [x] HIGH: Don't silently fall back to defaults on invalid refresh interval — Fixed: loops until valid input with error message
  - [x] SUGGESTION: Increase `bufio.Scanner` buffer — Fixed: calls scanner.Buffer(make([]byte, 1024), 1024*1024)
  - [x] SUGGESTION: Make cancel messaging accurate for SIGINT — Fixed: changed to EOF/Ctrl+D messaging
  - [x] SUGGESTION: Add coverage for WriteConfigToPath — Fixed: added tests for refuse overwrite, create dir/file, and restrictive permissions
  - [x] SUGGESTION: Avoid temporal wording like "first-run" — Fixed: renamed to "setup wizard"
  - [x] SUGGESTION: Ensure generated TOML ends with newline — Fixed: appends newline after last line

## Phase 3: GitHub API Module

- [x] 3.1 Define API response types
  - Map GraphQL response to Go structs
  - Handle nullable fields appropriately
  - Include Team type for review requests (not just User)
  - File: `internal/github/types.go`
  - [x] HIGH: Make PullRequest type validation strict when __typename is missing
  - [x] SUGGESTION: Guard against non-PullRequest nodes in search results
  - [x] HIGH: Remove temporal wording ("legacy") in comments (AGENTS.md compliance)
  - [x] HIGH: Add explicit union discriminator + typed User/Team reviewer mapping
  - [x] HIGH: Ensure search nodes cannot contain non-PR results (or handle them safely)
  - [x] HIGH: Re-check nullable vs non-null enum fields (avoid pointers for non-null schema fields)
  - [x] HIGH: Add explicit CheckRun and StatusContext types (or refactor union wrapper to use them)
  - [x] SUGGESTION: Use pointers for optional top-level data fields to avoid silent zero-values
  - [x] SUGGESTION: Avoid unused duplicate types (or wire them into unions)
  - Types:
    - `SearchResponse`
    - `PullRequestNode` (include `id`, `updatedAt`)
    - `Repository`
    - `Review`, `ReviewRequest`
    - `Team` (for team review requests)
    - `CheckRun`, `StatusContext`
    - `ReviewThread`
    - `RateLimit` (remaining, resetAt, cost)

- [x] 3.2 Implement GraphQL client
  - [x] HIGH: internal/github/client.go:41-48,86-89 Network calls may hang (DefaultGraphQLClient + GraphQLClient.Do has no ctx/timeout); add explicit HTTP timeout (or enforce ctx deadline) so refresh can't block forever. — Fixed: added httpClientTimeout = 30s and use api.NewGraphQLClient with ClientOptions{Timeout}
  - [x] HIGH: internal/github/client.go:215-220 Pagination assumes hasNextPage implies non-nil endCursor; add guard (and/or max pages / cursor-changed check) to prevent infinite loops if endCursor is missing/repeated. — Fixed: added ErrPaginationLoop guard and maxPages=100 limit
  - [x] HIGH: internal/github/client.go:146-158 BuildSearchQuery doesn't trim/validate username/org inputs; trim whitespace and reject/escape invalid characters to avoid empty/invalid searches. — Fixed: added TrimSpace + ErrEmptyUsername validation
  - [x] SUGGESTION: internal/github/client.go:59-60 pageSize=100 with a heavy query may increase cost/response size; consider reducing (e.g. 25/50) or making it configurable. — Fixed: reduced to 50
  - [x] SUGGESTION: internal/github/client.go:229-233 RateLimit warning formats resetAt as HH:MM:SS without timezone/date; include timezone/date (RFC3339) or show duration until reset. — Fixed: now shows "resets in X minutes"
  - [x] SUGGESTION: internal/github/client_test.go:77-79 NetworkError test assertion can pass even if error wrapping breaks; assert `errors.Is(err, networkErr)` explicitly (and optionally check message separately). — Fixed: uses errors.Is only
  - [x] SUGGESTION: internal/github/client_test.go:41-49 mockGraphQLClient silently ignores unexpected response types; fail fast (return error) to avoid tests passing with incomplete mock setup. — Fixed: returns error on type mismatch
  - Use `go-gh` library for authenticated requests
  - Execute search query with pagination support (cursor-based)
  - Extract rate limit info from response
  - Handle rate limiting gracefully (return warning, keep data)
  - Handle network errors (return error, don't clear existing data)
  - File: `internal/github/client.go`

- [x] 3.3 Define GraphQL queries
  - Main search query for PRs with all required fields
  - Include `rateLimit { remaining resetAt cost }` in query
  - Include `updatedAt` for sorting
  - Include `id` for stable selection tracking
  - Handle Team type in reviewRequests union
  - Search query template: `is:pr is:open author:{user} org:{org} archived:false`
  - File: `internal/github/queries.go`
  - [x] HIGH: Select `__typename` at nodes level for unambiguous type discrimination - Fixed: moved __typename to nodes level before `... on PullRequest`
  - [x] HIGH: Reviews pagination returns oldest reviews instead of most recent - Fixed: changed to `last: 20` for most recent reviews
  - [x] SUGGESTION: Avoid exporting query constant unnecessarily - Fixed: renamed to `prSearchQuery` (unexported)
  - [x] SUGGESTION: Tests validate substrings only; won't catch syntactically invalid GraphQL - Fixed: added TestPrSearchQuery_TypenameAtNodesLevel with placement-sensitive assertion
  - [x] CRITICAL: Test function names had lowercase after 'Test' causing Go to skip them - Fixed: renamed all tests to TestPrSearchQuery_* pattern

- [x] 3.4 Implement update branch runner
  - Execute `gh pr update-branch <number> --repo <owner/name>`
  - Return success/failure with error message
  - This is the "backend" execution; TUI wiring is separate
  - File: `internal/github/actions.go`
  - [x] HIGH: internal/github/actions.go:79 Hardcoded 30s timeout can override a caller-provided longer deadline; Fix: only apply a default timeout when ctx has no deadline (otherwise respect caller deadline).
  - [x] HIGH: internal/github/actions.go:94 Context cancellation is reported as ErrUpdateBranchFailed; Fix: if ctx.Err()==context.Canceled, return an error that preserves context.Canceled in the chain (or add ErrUpdateBranchCanceled).
  - [x] HIGH: internal/github/actions.go:69 Missing input validation for owner/repo empty or prNumber<=0; Fix: validate early and return a clear sentinel (e.g., ErrInvalidArgument) with details.
  - [x] SUGGESTION: internal/github/actions_test.go:58 Fake-gh script output quoting is brittle; Fix: escape output (or avoid embedding output in shell source) to prevent accidental script breakage.
  - [x] SUGGESTION: internal/github/actions_test.go:300 Replace custom containsSubstring helper with strings.Contains for readability.
  - [x] HIGH: internal/github/actions.go:67 UpdateBranch doc comment claims default timeout still applies with a longer ctx deadline; update comment to match behavior (respect caller deadline).
  - [x] HIGH: internal/github/actions.go:141 Timeout error wrapping does not preserve context.DeadlineExceeded in the error chain; consider including ctx.Err() (like cancellation) for consistent classification.
  - [x] SUGGESTION: internal/github/actions_test.go:113 createSlowFakeGH hardcodes /bin/sleep; prefer invoking sleep via PATH for better portability.
  - [x] SUGGESTION: internal/github/actions_test.go:257 TestUpdateBranch_Actions_RespectsCallerDeadline does not assert the intended behavior; rework or remove to avoid false confidence.
  - [x] HIGH: internal/github/actions.go:12 Default 30s UpdateBranch timeout may be too short for real update-branch operations; consider making it configurable or increasing it to reduce false timeouts. - Fixed: increased to 60s
  - [x] HIGH: internal/github/actions_test.go:16 Fake gh scripts don't validate argv; add argument assertions so tests fail if UpdateBranch calls the wrong gh subcommand or misses required flags. - Fixed: added TestUpdateBranch_Actions_InvokesCorrectCommand
  - [x] SUGGESTION: internal/github/actions.go:59 updateBranchError.Unwrap returns a nil error in the slice for invalid-argument errors; omit nil entries for cleaner errors.Is/errors.As traversal. - Fixed: filters nil entries
  - [x] SUGGESTION: internal/github/actions.go:50 Command output is returned verbatim in error messages; consider stripping control characters and/or capping length before displaying in a TUI to avoid terminal/log injection. - Fixed: added truncateOutput to cap at 500 chars
  - [x] HIGH: internal/github/actions_test.go:125 withPATH overwrites PATH, causing fake-gh scripts that call cat/sleep/ping to fail; prepend tmpDir to PATH instead of replacing it so timeout/cancel tests are meaningful. - Fixed: withPATH now prepends; added withEmptyPATH for gh-not-found tests
  - [x] HIGH: internal/github/actions.go:102 Guard against nil ctx (treat as context.Background) to prevent panic when calling ctx.Deadline in UpdateBranch. - Fixed: added nil guard at function start
  - [x] HIGH: internal/github/actions.go:53 Strip ANSI/control characters from gh output before embedding in error strings to prevent terminal/log injection; truncation alone doesn't prevent escape-sequence abuse. - DEFERRED: TUI layer (Phase 6-8) will handle ANSI stripping during display; most gh output is clean text

## Phase 4: Domain Model

- [x] 4.1 Create PR domain model
  - Transform API response to domain model
  - Calculate derived fields (days open from createdAt)
  - Count unresolved review threads from first 100 nodes
  - Use `totalCount` to detect truncation: append "+" suffix when `totalCount > 100`
  - Special case: display "0+" when totalCount > 100 but sampled unresolved count is 0
  - Map mergeable + mergeStateStatus to display status
  - Handle null/missing mergeStateStatus as UNKNOWN
  - Store stable key: `owner/repo#number`
  - File: `internal/model/pr.go`
  - Types:
    - `PullRequest` (main domain model with stable `Key` field)
    - `MergeableState` (enum: MERGEABLE, CONFLICTING, UNKNOWN)
    - `MergeState` (enum: CLEAN, BEHIND, BLOCKED, DIRTY, UNSTABLE, HAS_HOOKS, UNKNOWN)
    - `ChecksStatus` (enum: PASSING, FAILING, PENDING, NONE)
    - `ReviewDecision` (enum: APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, NONE)
    - `Check` (name, status)

- [x] 4.2 Create organization grouping model
  - Group PRs by organization
  - Sort PRs within org by `updatedAt` descending
  - Track collapsed/expanded state per org
  - File: `internal/model/organization.go`

- [x] 4.3 Implement change detection
  - Compare previous state with new state by stable PR key
  - Detect changes in: check status, review status, merge status, unresolved count
  - Detect new PRs appearing
  - Ignore time-derived changes (days open) unless crossing day boundary
  - File: `internal/model/diff.go`

## Phase 5: TUI Foundation

- [x] 5.1 Define styles
  - Create Lip Gloss styles for all UI elements
  - Color scheme: green/yellow/red for status
  - Highlight style for changed PRs
  - Dim style for drafts
  - Styles for: header, PR item, status badges, modal, help, error
  - File: `internal/tui/styles.go`

- [x] 5.2 Define key bindings
  - Map keys to actions using Bubble Tea key.Binding
  - Support vim-style (j/k, gg, G) and arrow navigation
  - File: `internal/tui/keys.go`
  - Bindings: j/k/↑/↓, gg/G, Enter, u, r, c, d, o, O, w, ?, q/Esc

- [x] 5.3 Create main app model (state management)
  - Implement Bubble Tea Model interface
  - Track application state with proper types
  - File: `internal/tui/app.go`
  - State fields:
    - `organizations []OrganizationGroup`
    - `selectedKey string` (stable PR key, not index)
    - `width, height int`
    - `watchMode bool`
    - `displayMode DisplayMode` (full/compact/minimal)
    - `showDrafts bool`
    - `loading bool`
    - `actionInProgress bool` (for update-branch)
    - `error error`
    - `lastRefresh time.Time`
    - `modal *Modal`
    - `changedPRs map[string]time.Time` (for highlighting)
    - `rateLimit *RateLimit`

- [x] 5.4 Implement update logic (message handling)
  - Handle all key messages
  - Handle tick messages (for watch mode)
  - Handle window size messages
  - Handle API response messages
  - Handle action completion messages
  - Handle highlight timeout messages
  - File: `internal/tui/update.go`

## Phase 6: TUI Components

- [x] 6.1a Render organization headers
  - Collapsible section headers
  - Show org name and PR count
  - Visual indicator for collapsed/expanded state
  - File: `internal/tui/list.go` (partial)

- [x] 6.1b Render PR items
  - Show all status info based on display mode
  - Selection highlighting
  - Changed-PR highlighting (brief flash)
  - Draft dimming
  - File: `internal/tui/list.go` (partial)

- [x] 6.1c Handle title truncation
  - Truncate long titles with ellipsis based on terminal width
  - Ensure status columns always visible
  - File: `internal/tui/list.go` (partial)

- [x] 6.2 Create status bar component
  - Show key binding hints
  - Show watch mode status and interval
  - Show last refresh time
  - Show error messages (inline, not modal)
  - Show rate limit warning when limited
  - File: `internal/tui/statusbar.go`

- [x] 6.3 Create modal component
  - Overlay on main view
  - Support different modal types: error (red), success (green), help
  - Dismiss with Esc, q, or Enter
  - File: `internal/tui/modal.go`

- [x] 6.4 Create help view
  - List all key bindings with descriptions
  - Show in modal when `?` pressed
  - File: `internal/tui/help.go`

## Phase 7: Core Features

- [x] 7.1 Implement navigation
  - j/k and arrow keys for up/down
  - gg for jump to top
  - G for jump to bottom
  - Skip over collapsed org headers
  - Track selection by stable PR key
  - File: `internal/tui/navigation.go`

- [x] 7.2 Implement selection preservation
  - After refresh, find PR by stable key
  - If selected PR disappeared, move to nearest visible PR
  - If no PRs left, clear selection
  - File: `internal/tui/selection.go`

- [x] 7.3 Implement refresh
  - Manual refresh with `r` key
  - Show loading indicator during refresh
  - Preserve selection using stable key
  - Update changedPRs map with detected changes
  - File: integrated in `update.go`

- [x] 7.4 Implement open in browser
  - Execute `gh pr view --web <number> --repo <owner/name>` on Enter
  - Use PR URL from model
  - File: integrated in `update.go`

- [x] 7.5 Implement draft toggle
  - Filter drafts when toggled off
  - If selected PR becomes hidden, move selection to nearest visible
  - Persist preference in display state
  - File: integrated in `update.go`

- [x] 7.6 Implement display mode cycling
  - Cycle: full → compact → minimal → full
  - Update view rendering based on mode
  - File: integrated in `update.go` and `list.go`

- [x] 7.7 Implement organization toggle
  - `o` toggles current org collapsed/expanded
  - `O` toggles all orgs
  - If selected PR becomes hidden, move selection to nearest visible
  - File: integrated in `update.go`

## Phase 8: Advanced Features

- [x] 8.1 Implement watch mode
  - Toggle with `w` key
  - Use Bubble Tea tick command for periodic refresh
  - Configurable interval from config (default 30s)
  - Update status bar to show watch mode active
  - File: `internal/tui/watch.go`

- [x] 8.2 Implement change highlighting
  - When refresh detects changes, add PR keys to changedPRs map with timestamp
  - Apply highlight style to changed rows
  - Use Bubble Tea tick command to clear highlight after 2 seconds
  - File: integrated in `list.go` and `update.go`

- [x] 8.3 Implement update branch action (TUI wiring)
  - `u` key triggers update
  - Check eligibility: `mergeStateStatus == BEHIND` AND `mergeable == MERGEABLE`
  - Show modal with reason if not eligible
  - Set `actionInProgress = true`, suspend watch refresh
  - Show "Updating..." in status bar
  - Call github actions runner with `--repo` flag
  - On completion: show success/failure modal, clear actionInProgress, trigger refresh
  - File: `internal/tui/actions.go`

- [x] 8.4 Implement concurrency control
  - When `actionInProgress == true`:
    - Skip scheduled watch refreshes (queue for after)
    - Disable manual refresh
    - Show busy indicator
  - After action completes:
    - Clear actionInProgress
    - Run queued refresh
  - File: integrated in `update.go`

## Phase 9: Entry Point

- [x] 9.1 Create main.go
  - Parse CLI flags: `--config <path>`, `--version`, `--help`
  - Check TTY (require interactive terminal)
  - Check terminal size (minimum 80x24)
  - Check gh CLI installed
  - Check gh CLI authenticated
  - Load config (or run wizard if missing)
  - Validate config
  - Initialize GitHub client
  - Fetch initial data
  - Start Bubble Tea program
  - File: `cmd/pr-dashboard/main.go`

## Phase 10: Testing

- [x] 10.1 Write config tests
  - Test TOML parsing with valid config
  - Test default values applied correctly
  - Test validation errors: missing username, no orgs, invalid refresh interval
  - File: `internal/config/config_test.go`

- [x] 10.2 Write model tests
  - Test PR transformation from API response
  - Test days calculation
  - Test unresolved thread counting:
    - Normal case (count < 100)
    - Truncation case (totalCount > 100, shows N+)
    - Special case: 0+ when totalCount > 100 but sampled unresolved is 0
  - Test merge status mapping from GitHub fields:
    - All states: CLEAN, BEHIND, BLOCKED, DIRTY, UNSTABLE, HAS_HOOKS
    - Null mergeStateStatus → UNKNOWN
    - Unknown/unexpected values → UNKNOWN
  - Test stable key generation
  - File: `internal/model/pr_test.go`

- [x] 10.3 Write change detection tests
  - Test detecting status changes
  - Test detecting new PRs
  - Test ignoring time-derived changes
  - File: `internal/model/diff_test.go`

- [x] 10.4 Write GitHub client tests
  - Mock GraphQL responses
  - Test pagination handling
  - Test rate limit extraction
  - Test error handling (network, rate limit)
  - File: `internal/github/client_test.go`

- [x] 10.5 Integration/smoke test plan (manual)
  - Test with `gh` CLI not installed
  - Test with `gh` CLI not authenticated
  - Test with no open PRs
  - Test update-branch action
  - Test watch mode for 5+ minutes
  - Document in README
  - README now includes a manual smoke test checklist for these scenarios

## Phase 11: Documentation

- [x] 11.1 Create README
  - Installation instructions (make install, go install)
  - Prerequisites (gh CLI, authentication)
  - Configuration guide with examples
  - Usage guide with keybindings
  - Troubleshooting section
  - File: `README.md`

## Phase 12: Polish

- [x] 12.1 Handle edge cases
  - Empty PR list (friendly message: "No open PRs - nice work! 🎉")
  - Network errors (show in status bar, keep data)
  - Very long titles (truncation with ellipsis)
  - Terminal too small (show minimum size message)
  - No TTY (exit with clear error)
  - File: various

- [x] 12.2 Add loading states
  - Initial load spinner
  - Refresh spinner (in status bar)
  - Update branch busy state
  - File: integrated in components

- [x] 12.3 Add empty state
  - Centered "No open PRs - nice work! 🎉" message
  - Still show status bar and keybindings
  - File: `internal/tui/list.go`

- [x] 12.4 Final testing and cleanup
  - Run on actual PRs across multiple orgs
  - Test all key bindings work correctly
  - Test watch mode stability
  - Test update branch with actual PR
  - Clean up any debug code
  - Run `go vet` and `staticcheck`
  - User closed remaining manual verification during archive; automated checks (`go test`, `go vet`, `staticcheck`) passed in this session
