# PR Dashboard TUI - Implementation Plan

This document tracks the implementation progress of the PR Dashboard TUI application.

---

## Latest Verification (v0.0.9)

- **All 238 tests passing** across 5 packages (83 config + 40 github + 41 model + 73 tui + 2 main)
- **Build successful** with go build
- **go vet clean** - no static analysis issues
- **All core phases (1-9) verified complete** with no TODOs, FIXMEs, or placeholder implementations
- **TUI architecture uses consolidated files** (update.go, view.go) rather than separate navigation.go, actions.go, etc.
- **ANSI stripping implemented** - gh CLI output is now cleaned of escape sequences before display in TUI modals and error messages
- **Loading spinners implemented** - animated spinners shown during initial load, manual refresh, and branch update operations using charmbracelet/bubbles spinner component
- **Error recovery implemented** - automatic retry with exponential backoff for transient network errors

---

## Executive Summary

| Phase | Status | Test Count | Notes |
|-------|--------|------------|-------|
| Phase 1: Project Foundation | COMPLETE | N/A | go.mod, Makefile, directory structure |
| Phase 2: Configuration | COMPLETE | 83 | Full wizard, validation, auth checks, config loading tests, ANSI stripping |
| Phase 3: GitHub API | COMPLETE | 40 | GraphQL client, pagination, UpdateBranch, ANSI stripping |
| Phase 4: Domain Model | COMPLETE | 41 | PR transformation, grouping, change detection |
| Phase 5: TUI Foundation | COMPLETE | 73 | Exceeds requirements - many Phase 6-8 features implemented early |
| Phase 6: PR List Component | COMPLETE | 0 | All display modes, empty state, team reviewers, truncation |
| Phase 7: Status Bar & Modals | COMPLETE | 0 | Status bar, help modal, success/error modals |
| Phase 8: TUI Features | COMPLETE | 0 | Navigation, watch mode, actions, change highlighting |
| Phase 9: Entry Point | COMPLETE | 2 | CLI, TTY validation, entry point |
| Phase 10-12: Testing/Docs/Polish | IN PROGRESS | 0 | Lower priority |

**Total Tests:** 238 (83 config + 40 github + 41 model + 73 tui + 2 main)

---

## Specification Mapping

| Spec File | Status | Implementation Location |
|-----------|--------|------------------------|
| configuration/spec.md | COMPLETE | internal/config/ |
| github-integration/spec.md | COMPLETE | internal/github/, internal/model/ |
| tui-navigation/spec.md | COMPLETE | internal/tui/ (Phase 5, 8) |
| watch-mode/spec.md | COMPLETE | internal/tui/ (Phase 8) |
| branch-actions/spec.md | COMPLETE | API: internal/github/actions.go, TUI: internal/tui/ |
| pr-display/spec.md | COMPLETE | internal/tui/ (Phase 6) |

---

## Phase 1: Project Foundation - COMPLETE

- [x] Initialize Go module (`go.mod`)
- [x] Create Makefile with build, install, clean, run, test targets
- [x] Set up directory structure (`cmd/pr-dashboard/`, `internal/{config,github,model,tui}/`)
- [x] Create `config.example.toml` with documented options
- [x] Use `install -m 0755` for deterministic permissions
- [x] Allow overriding Go tool via `GO ?= go`

**Files:**
- `/home/jgordijn/projects/gpr-management/go.mod`
- `/home/jgordijn/projects/gpr-management/Makefile`
- `/home/jgordijn/projects/gpr-management/config.example.toml`

---

## Phase 2: Configuration Module (`internal/config/`) - COMPLETE

- [x] Define `Config` struct matching TOML structure
- [x] Implement `Load()` and `LoadFromPath()` with `ErrConfigNotFound` sentinel
- [x] Implement `DefaultConfigPath()` for `~/.config/pr-dashboard/config.toml`
- [x] Apply defaults before TOML parsing (not after)
- [x] Reject unknown TOML keys using `meta.Undecoded()`
- [x] Implement `Validate()` with `ValidationError` type
- [x] Trim whitespace for `display.initial_mode` before validating
- [x] Handle empty `ValidationError.Errors` in `Error()` formatting
- [x] Implement `CheckGHCLI()` with `ErrGHCLINotFound` sentinel
- [x] Preserve underlying `exec.LookPath` error for diagnostics
- [x] Implement `CheckGHAuth()` with `ErrGHNotAuthenticated` sentinel
- [x] Add timeout (5s) for `gh auth status` command
- [x] Include `gh auth status` output in errors for troubleshooting
- [x] Implement setup wizard with atomic file writes (temp file + rename)
- [x] Use restrictive permissions (0700 for dir, 0600 for file)
- [x] Refuse to overwrite existing config files
- [x] Loop until valid input with error messages
- [x] Ensure generated TOML ends with newline
- [x] Write comprehensive unit tests (61 test functions)

**Files:**
- `/home/jgordijn/projects/gpr-management/internal/config/config.go` - Config struct, TOML loading, defaults
- `/home/jgordijn/projects/gpr-management/internal/config/config_test.go` - 17 test functions for Load, LoadFromPath, DefaultConfigPath
- `/home/jgordijn/projects/gpr-management/internal/config/validation.go` - ValidationError, input validation
- `/home/jgordijn/projects/gpr-management/internal/config/validation_test.go` - 19 test functions
- `/home/jgordijn/projects/gpr-management/internal/config/auth.go` - gh CLI detection and authentication
- `/home/jgordijn/projects/gpr-management/internal/config/auth_test.go` - 9 test functions
- `/home/jgordijn/projects/gpr-management/internal/config/wizard.go` - Interactive setup wizard
- `/home/jgordijn/projects/gpr-management/internal/config/wizard_test.go` - 33 test functions

---

## Phase 3: GitHub API Module (`internal/github/`) - COMPLETE

- [x] Define GraphQL response types with union type handling
- [x] Implement `PullRequestNode.IsPullRequest()` for type discrimination
- [x] Implement `PullRequestNode.HasTypename()` for defensive validation
- [x] Handle `RequestedReviewer` union (User vs Team) with typed accessors
- [x] Handle `CheckStatusContext` union (CheckRun vs StatusContext) with typed accessors
- [x] Implement `Client` with `GraphQLClient` interface for testability
- [x] Add 30s HTTP timeout via `api.ClientOptions{Timeout}`
- [x] Implement `FetchPRs()` with cursor-based pagination
- [x] Use pageSize=50 for reasonable API cost tradeoff
- [x] Add `ErrPaginationLoop` guard for nil endCursor
- [x] Add `ErrMaxPagesExceeded` guard (maxPages=100)
- [x] Implement rate limit warning at <=100 remaining (shows minutes until reset)
- [x] Define `prSearchQuery` with `__typename` at nodes level
- [x] Use `last: 20` for reviews to get most recent
- [x] Use `first: 100` for reviewThreads with `totalCount`
- [x] Use `first: 20` for check contexts with aggregate `statusCheckRollup.state`
- [x] Validate username is non-empty (`ErrEmptyUsername`)
- [x] Trim whitespace from username and organization inputs
- [x] Implement `UpdateBranch()` with 60s default timeout
- [x] Respect caller-provided context deadline
- [x] Guard against nil context (treat as `context.Background()`)
- [x] Define sentinel errors: `ErrGHCLINotFound`, `ErrUpdateBranchTimeout`, `ErrUpdateBranchFailed`, `ErrUpdateBranchCancelled`, `ErrInvalidArgument`
- [x] Preserve `context.DeadlineExceeded` and `context.Canceled` in error chain
- [x] Truncate command output to 500 chars in error messages
- [x] Filter nil entries from `Unwrap()` error slice
- [x] Validate owner, repo non-empty and prNumber > 0
- [x] Write comprehensive unit tests (32 test functions)

**Deferred Items:**
- ANSI/control character stripping from gh output - deferred to TUI layer (Phase 6-8)

**Test Notes:**
- 4 tests with `t.Skip` in actions_test.go for timing-sensitive tests in short mode
  - These are legitimate skips for tests requiring sleep/timing behavior

**Files:**
- `/home/jgordijn/projects/gpr-management/internal/github/types.go` - GraphQL response types (339 lines)
- `/home/jgordijn/projects/gpr-management/internal/github/client.go` - GraphQL client with pagination (212 lines)
- `/home/jgordijn/projects/gpr-management/internal/github/client_test.go` - 15 test functions
- `/home/jgordijn/projects/gpr-management/internal/github/queries.go` - GraphQL query definition (92 lines)
- `/home/jgordijn/projects/gpr-management/internal/github/queries_test.go` - 6 test functions
- `/home/jgordijn/projects/gpr-management/internal/github/actions.go` - UpdateBranch implementation (187 lines)
- `/home/jgordijn/projects/gpr-management/internal/github/actions_test.go` - 11 test functions (4 with t.Skip for timing-sensitive tests)
  - Added retry with exponential backoff tests (6 new tests in v0.0.9)

---

## Phase 4: Domain Model (`internal/model/`) - COMPLETE

**Spec Coverage:** Implements requirements from `github-integration/spec.md` (PR Data Transformation) and supports `pr-display/spec.md`

### 4.1 Create PR Domain Model (`internal/model/pr.go`)

- [x] Define `PullRequest` domain struct:
  - `Key` - Stable identifier: `owner/repo#number` (per spec: stable PR key)
  - `Title` - PR title
  - `Author` - GitHub username
  - `DaysOpen` - Calculated from `createdAt` (per spec: days since creation)
  - `ReviewStatus` - Aggregated review state enum
  - `CheckStatus` - Aggregated CI status enum (per spec: use statusCheckRollup.state)
  - `MergeStatus` - Computed merge readiness enum
  - `UnresolvedThreads` - Count string with "+" suffix for >100 (per spec: totalCount truncation)
  - `Reviewers` - Flattened list with "team:" prefix for teams (per spec)
  - `IsDraft` - Draft PR indicator
  - `URL` - Full GitHub URL
  - `Organization` - Owner login for grouping
  - `Repository` - Repo name
  - `Number` - PR number
  - `UpdatedAt` - For sorting

- [x] Define `ReviewStatus` enum with `String()` method:
  - `ReviewStatusApproved` - At least one approval, no changes requested
  - `ReviewStatusChangesRequested` - Changes requested by reviewer
  - `ReviewStatusReviewRequired` - Awaiting review
  - `ReviewStatusNone` - No review activity

- [x] Define `CheckStatus` enum with `String()` method:
  - `CheckStatusPassing` - All checks successful (state=SUCCESS)
  - `CheckStatusFailing` - Any check failed (state=FAILURE or ERROR)
  - `CheckStatusPending` - Checks in progress (state=PENDING)
  - `CheckStatusNone` - No checks configured (null status)

- [x] Define `MergeStatus` enum with `String()` method (per pr-display/spec.md):
  - `MergeStatusReady` - mergeable=MERGEABLE && mergeStateStatus=CLEAN
  - `MergeStatusBehind` - mergeable=MERGEABLE && mergeStateStatus=BEHIND
  - `MergeStatusBlocked` - mergeable=MERGEABLE && mergeStateStatus=BLOCKED
  - `MergeStatusConflicts` - mergeable=CONFLICTING
  - `MergeStatusDirty` - mergeStateStatus=DIRTY
  - `MergeStatusUnstable` - mergeStateStatus=UNSTABLE
  - `MergeStatusHasHooks` - mergeStateStatus=HAS_HOOKS
  - `MergeStatusUnknown` - mergeable=UNKNOWN or null, or unexpected values
  - `MergeStatusDraft` - PR is a draft (overrides other states)

- [x] Define `DisplayMode` enum:
  - `DisplayModeFull` - All columns visible
  - `DisplayModeCompact` - Reduced column set
  - `DisplayModeMinimal` - Essential info only

### 4.2 Create Organization Grouping Model (`internal/model/organization.go`)

- [x] Define `PRGroup` struct:
  - `Organization` - GitHub org login
  - `PRs` - Slice of PullRequest
  - `Collapsed` - UI collapse state

- [x] Define `PRList` struct:
  - `Groups` - Slice of PRGroup
  - `TotalCount` - Total PR count across all groups

- [x] Implement `GroupByOrganization(prs []PullRequest) []PRGroup`
- [x] Implement sort by `updatedAt` descending within each group (per pr-display/spec.md)

### 4.3 Create Transformer Service (`internal/model/transformer.go`)

- [x] Implement `TransformPR(apiPR github.PullRequestNode) PullRequest`:
  - Calculate `DaysOpen` from `createdAt` (round to nearest day)
  - Generate `Key` as `owner/repo#number`
  - Extract `Organization` from repository owner login
  - Map `reviewDecision` to `ReviewStatus`
  - Map `statusCheckRollup.state` to `CheckStatus` (SUCCESS->PASSING, FAILURE/ERROR->FAILING, PENDING->PENDING, null->NONE)
  - Derive `MergeStatus` from `mergeable` + `mergeStateStatus` + `isDraft`
  - Count unresolved threads from nodes
  - Format `UnresolvedThreads` with "+" when `totalCount > len(nodes)`
  - Special case: display "0+" when totalCount > 100 but sampled unresolved = 0
  - Flatten reviewers with "team:" prefix for teams

- [x] Implement `TransformPRs(apiPRs []github.PullRequestNode) []PullRequest`

### 4.4 Create Change Detection Service (`internal/model/diff.go`)

Per `watch-mode/spec.md`:

- [x] Define `Change` struct:
  - `Key` - PR key that changed
  - `Type` - ChangeType enum
  - `OldValue` - Previous value (for display)
  - `NewValue` - New value (for display)

- [x] Define `ChangeType` enum:
  - `ChangeTypeNew` - PR appeared in list
  - `ChangeTypeRemoved` - PR no longer in list
  - `ChangeTypeReviewStatus` - Review status changed
  - `ChangeTypeCheckStatus` - Check status changed
  - `ChangeTypeMergeStatus` - Merge status changed
  - `ChangeTypeThreadCount` - Unresolved thread count changed

- [x] Implement `DetectChanges(old, new []PullRequest) []Change`:
  - Track new PRs (key not in old list)
  - Track removed PRs (key not in new list)
  - Track status changes by comparing fields
  - Ignore `DaysOpen` changes unless crossing day boundary (per spec)

### 4.5 Write Unit Tests (`internal/model/*_test.go`)

- [x] Test all enum derivations with edge cases
- [x] Test transformer with various API response shapes
- [x] Test change detection scenarios
- [x] Test "0+" special case for threads (totalCount > 100 but sampled unresolved = 0)
- [x] Test merge status derivation for all combinations
- [x] Test unexpected/unknown merge status values map to UNKNOWN

### Files

- `/home/jgordijn/projects/gpr-management/internal/model/pr.go` - Domain types, enums, PullRequest struct
- `/home/jgordijn/projects/gpr-management/internal/model/organization.go` - PRGroup, PRList, grouping functions
- `/home/jgordijn/projects/gpr-management/internal/model/transformer.go` - API to domain type conversion
- `/home/jgordijn/projects/gpr-management/internal/model/diff.go` - Change detection
- `/home/jgordijn/projects/gpr-management/internal/model/pr_test.go` - Enum and method tests
- `/home/jgordijn/projects/gpr-management/internal/model/organization_test.go` - Grouping tests
- `/home/jgordijn/projects/gpr-management/internal/model/transformer_test.go` - Transformer tests
- `/home/jgordijn/projects/gpr-management/internal/model/diff_test.go` - Change detection tests

---

## Phase 5: TUI Foundation (`internal/tui/`) - COMPLETE

**Status:** COMPLETE - 32 tests passing

**Depends on:** Phase 4 (Domain Model)

**Note:** Phase 5 substantially exceeds requirements. Many features from Phase 6-8 were implemented early, including navigation, display mode cycling, organization collapse/expand, draft visibility toggle, and change highlighting. The TUI module contains partial implementations for PR list rendering, status bar, modals, and core TUI features.

### 5.0 Add Charm Dependencies to `go.mod`

- [x] Add `github.com/charmbracelet/bubbletea` (TUI framework)
- [x] Add `github.com/charmbracelet/lipgloss` (styling)
- [x] Add `github.com/charmbracelet/bubbles` (pre-built components)
- [x] Run `go mod tidy` to fetch dependencies

### 5.1 Define Lip Gloss Styles (`internal/tui/styles.go`)

Per `pr-display/spec.md`:

- [x] Define color constants:
  - Green: `#10B981` (PASSING, APPROVED, READY, CLEAN)
  - Yellow: `#F59E0B` (PENDING, BEHIND, BLOCKED, UNSTABLE, HAS_HOOKS)
  - Red: `#EF4444` (FAILING, CHANGES_REQUESTED, CONFLICTS, DIRTY)
  - Gray: `#6B7280` (NONE, UNKNOWN, DRAFT)
  - Highlight: `#3B82F6` (selection, changes)

- [x] Define styles:
  - `HeaderStyle` - Organization header (bold)
  - `SelectedStyle` - Selected row (reverse video)
  - `DraftStyle` - Draft PR (dimmed)
  - `ChangedStyle` - Recently changed PR (highlight background)
  - `StatusPassingStyle`, `StatusFailingStyle`, `StatusPendingStyle`, `StatusNoneStyle`
  - `ModalStyle` - Modal border and background
  - `ModalTitleStyle` - Modal title (bold)
  - `StatusBarStyle` - Bottom status bar

### 5.2 Define Key Bindings (`internal/tui/keys.go`)

Per `tui-navigation/spec.md`:

- [x] Define `KeyMap` struct using `key.Binding`:
  - `Up` - k, up arrow
  - `Down` - j, down arrow
  - `Top` - gg (vim-style sequence)
  - `Bottom` - G (vim-style)
  - `ToggleOrg` - o
  - `ToggleAllOrgs` - O (shift+o)
  - `ToggleDrafts` - d
  - `CycleDisplayMode` - c
  - `ToggleWatch` - w
  - `UpdateBranch` - u
  - `Refresh` - r
  - `OpenBrowser` - Enter
  - `Help` - ?
  - `Quit` - q, Escape

- [x] Implement `ShortHelp()` and `FullHelp()` for help modal

### 5.3 Create Main App Model (`internal/tui/app.go`)

- [x] Define `Model` struct:
  - `Config` - Configuration reference
  - `Client` - GitHub client reference
  - `PRList` - Current PR data
  - `SelectedKey` - Track by key, not index (per spec: stable key)
  - `DisplayMode` - full/compact/minimal
  - `WatchMode` - Auto-refresh enabled
  - `ShowDrafts` - Draft visibility
  - `ShowHelp` - Help modal visible
  - `ActionModal` - Success/error modal
  - `LastRefresh` - For status bar
  - `RateLimit` - For status bar
  - `IsLoading` - Loading state
  - `ActionInProgress` - Block concurrent actions
  - `RefreshQueued` - Queue refresh during action
  - `ChangedKeys` - Map for highlighting
  - `Error` - Current error message
  - `Width`, `Height` - Terminal dimensions
  - `KeyMap` - Key bindings

- [x] Define `ActionModal` struct for success/error display
- [x] Implement `NewModel(cfg *config.Config, client *github.Client) Model`

### 5.4 Define Custom Messages (`internal/tui/messages.go`)

- [x] Define messages:
  - `PRsLoadedMsg` - PR data fetched successfully
  - `PRsErrorMsg` - Error fetching PRs
  - `ActionStartMsg` - Action initiated
  - `ActionResultMsg` - Action completed (success/error)
  - `RefreshTickMsg` - Watch mode timer tick
  - `ClearHighlightMsg` - Clear change highlighting for a key
  - `WindowSizeMsg` - Terminal resized

### 5.5 Implement Core Bubble Tea Methods (`internal/tui/update.go`)

- [x] Implement `Init() tea.Cmd`:
  - Return command to fetch initial PR data
  - Return command to get terminal size

- [x] Implement `Update(msg tea.Msg) (tea.Model, tea.Cmd)`:
  - Handle `tea.KeyMsg` - dispatch to key handlers
  - Handle `tea.WindowSizeMsg` - update dimensions
  - Handle `PRsLoadedMsg` - update PR list, detect changes
  - Handle `PRsErrorMsg` - set error state
  - Handle `ActionResultMsg` - show modal, clear action state
  - Handle `RefreshTickMsg` - trigger refresh if watch mode
  - Handle `ClearHighlightMsg` - remove key from changed set

- [x] Implement `View() string`:
  - Compose header, PR list, and status bar
  - Overlay modal if visible
  - Handle loading state
  - Handle error state

### 5.6 Write Unit Tests

- [x] Test key binding dispatch
- [x] Test state transitions
- [x] Test message handling

### Files

- `/home/jgordijn/projects/gpr-management/internal/tui/app.go` - Main application model and NewModel
- `/home/jgordijn/projects/gpr-management/internal/tui/app_test.go` - Model tests
- `/home/jgordijn/projects/gpr-management/internal/tui/keys.go` - Key bindings with KeyMap struct
- `/home/jgordijn/projects/gpr-management/internal/tui/keys_test.go` - Key binding tests
- `/home/jgordijn/projects/gpr-management/internal/tui/messages.go` - Custom Bubble Tea messages
- `/home/jgordijn/projects/gpr-management/internal/tui/styles.go` - Lip Gloss styles and colors
- `/home/jgordijn/projects/gpr-management/internal/tui/styles_test.go` - Style tests
- `/home/jgordijn/projects/gpr-management/internal/tui/update.go` - Update and Init implementations
- `/home/jgordijn/projects/gpr-management/internal/tui/update_test.go` - Update handler tests
- `/home/jgordijn/projects/gpr-management/internal/tui/view.go` - View rendering

---

## Phase 6: TUI PR List Component - COMPLETE

**Depends on:** Phase 4, Phase 5

**Spec Coverage:** Implements `pr-display/spec.md`

### 6.1 Implement PR List Rendering (`internal/tui/list.go`)

- [x] Implement `RenderPRList(m *Model) string`:
  - Iterate through PRGroups
  - Render organization headers
  - Render PR rows (skip collapsed orgs)
  - Apply selection highlighting
  - Apply change highlighting
  - Handle scrolling/viewport

- [x] Implement organization header rendering (per pr-display/spec.md):
  - Format: `>` OrgName (5 PRs)` collapsed, `v OrgName (5 PRs)` expanded
  - Apply `HeaderStyle`
  - Show org name and PR count

- [x] Implement PR row rendering by display mode (per pr-display/spec.md):
  - Full: `#123 Title [Draft] Author 3d | Checks | Reviews | Merge | Threads` (with reviewer names)
  - Compact: `#123 Title Author | Icons only` (summary icons with counts)
  - Minimal: `#123 Title Author` (title, status badge, age only)

- [x] Implement status column formatting (per pr-display/spec.md):
  - Review: icon + reviewer names (full) or count (compact)
  - Checks: icon + state text (full) or icon only (compact)
  - Merge: icon + state text with color (per spec color mapping)
  - Threads: count with "+" suffix if truncated

- [x] Implement Draft PR styling (per pr-display/spec.md):
  - Display "DRAFT" badge
  - Visually dimmed compared to ready PRs

### 6.2 Implement Title Truncation (per pr-display/spec.md)

- [x] Calculate available width based on terminal and column widths
- [x] Truncate with ellipsis ("...") preserving readability
- [x] Ensure status columns always visible

### 6.3 Implement Empty State (per pr-display/spec.md)

- [x] Render "No open PRs - nice work! :tada:" message centered
- [x] Show refresh hint

### 6.4 Write Unit Tests

- [x] Test all display modes (view_test.go)
- [x] Test truncation logic (view_test.go)
- [x] Test status formatting (view_test.go)
- [x] Test team reviewer display with "team:" prefix (view_test.go)

**Test File:** `/home/jgordijn/projects/gpr-management/internal/tui/view_test.go` - 36 view rendering tests

---

## Phase 7: TUI Status Bar & Modals - COMPLETE

**Depends on:** Phase 5

**Spec Coverage:** Implements parts of `watch-mode/spec.md`, `branch-actions/spec.md`, `tui-navigation/spec.md`

### 7.1 Implement Status Bar (`internal/tui/statusbar.go`)

Per `watch-mode/spec.md` and `github-integration/spec.md`:

- [x] Left section: Mode indicators (watch icon with interval, display mode)
- [x] Center section: "Last refresh: HH:MM"
- [x] Right section: Rate limit warning (if <=100 remaining, show reset time)
- [x] Keyboard hint: "Press ? for help"

### 7.2 Implement Help Modal (`internal/tui/help.go`)

Per `tui-navigation/spec.md`:

- [x] List all key bindings grouped by category:
  - Navigation: j/k, gg/G, o/O
  - Actions: u, r, Enter
  - Display: c, d, w
  - Other: ?, q/Esc
- [x] Show dismiss hint at bottom
- [x] Dismiss with Enter, q, or Escape

### 7.3 Implement Action Modals (`internal/tui/modal.go`)

Per `branch-actions/spec.md`:

- [x] Success modal with green styling
- [x] Error modal with red styling
- [x] Dismiss with Enter, q, or Escape
- [x] Center in terminal with border

### 7.4 Write Unit Tests

- [x] Test status bar content variations (view_test.go)
- [x] Test modal rendering (view_test.go)
- [x] Test dismissal handling (view_test.go)

**Test File:** Tests included in `/home/jgordijn/projects/gpr-management/internal/tui/view_test.go`

---

## Phase 8: TUI Features - COMPLETE

**Depends on:** Phases 5, 6, 7

**Spec Coverage:** Implements `tui-navigation/spec.md`, `watch-mode/spec.md`, `branch-actions/spec.md`, `pr-display/spec.md`

### 8.1 Implement Navigation (`internal/tui/navigation.go`)

Per `tui-navigation/spec.md`:

- [x] Single step up/down with j/k and arrows
- [x] Jump to first with `gg`, last with `G`
- [x] Skip collapsed organizations
- [x] Track selection by stable key (not index)

### 8.2 Implement Selection Preservation

Per `tui-navigation/spec.md`:

- [x] After refresh, find PR by stable key
- [x] If selected PR removed (merged/closed), select nearest visible
- [x] If selected PR hidden by filter (draft toggle, org collapse), move to nearest visible
- [x] If no PRs, clear selection

### 8.3 Implement Organization Collapse/Expand

Per `tui-navigation/spec.md`:

- [x] Toggle single org with `o`
- [x] Toggle all orgs with `O`
- [x] Adjust selection if in collapsed group

### 8.4 Implement Display Mode Cycling

Per `pr-display/spec.md`:

- [x] Cycle: full -> compact -> minimal -> full
- [x] Persist during session

### 8.5 Implement Draft Visibility Toggle

Per `pr-display/spec.md`:

- [x] Toggle with `d`
- [x] Update counts
- [x] Adjust selection if selected PR becomes hidden

### 8.6 Implement Manual Refresh

Per `tui-navigation/spec.md`:

- [x] Trigger with `r`
- [x] Show loading indicator
- [x] Queue if action in progress
- [x] Preserve selection by stable key

### 8.7 Implement Open in Browser

Per `tui-navigation/spec.md`:

- [x] Execute `gh pr view --web <number> --repo <owner/name>` on Enter
- [x] Handle failures gracefully

### 8.8 Implement Watch Mode (`internal/tui/watch.go`)

Per `watch-mode/spec.md`:

- [x] Toggle with `w`
- [x] Auto-refresh at configured interval (default 30s)
- [x] Suspend during actions with queued refresh
- [x] Show indicator in status bar with refresh interval

### 8.9 Implement Change Highlighting

Per `watch-mode/spec.md`:

- [x] On refresh, add changed PR keys to map with timestamp
- [x] Detect changes in: check status, review status, merge status, unresolved count
- [x] Detect new PRs appearing
- [x] Apply highlight style to changed rows
- [x] Clear after 2 seconds using tick command

### 8.10 Implement Update Branch Action (`internal/tui/actions.go`)

Per `branch-actions/spec.md`:

- [x] Pre-validate: mergeable=MERGEABLE && mergeStateStatus=BEHIND
- [x] Show modal for invalid states:
  - CONFLICTS: "Cannot update: PR has merge conflicts"
  - UNKNOWN: "Cannot update: merge status unknown (GitHub is still computing)"
  - CLEAN: "Branch is already up to date"
  - BLOCKED: "Cannot update: merge is blocked"
  - DRAFT: "Cannot update: PR is a draft"
- [x] Set `actionInProgress = true`
- [x] Show loading indicator during operation
- [x] Suspend watch mode refreshes until completion
- [x] Call `github.UpdateBranch()` with `--repo` flag
- [x] Show success/error modal with appropriate styling
- [x] Clear action state, trigger refresh

### 8.11 Implement Concurrency Control

Per `branch-actions/spec.md`:

- [x] Block actions while `actionInProgress` (pressing 'u' ignored)
- [x] Queue refresh during action
- [x] Execute queued refresh after action completes

### 8.12 Write Integration Tests

- [x] Test navigation scenarios
- [x] Test action flows
- [x] Test watch mode behavior

---

## Phase 9: Entry Point (`cmd/pr-dashboard/main.go`) - COMPLETE

**Status:** COMPLETE - Entry point implemented with TTY validation, terminal size checks, and graceful shutdown

**Spec Coverage:** Implements `configuration/spec.md` (Terminal Requirements, startup sequence)

**Note:** Can partially develop in parallel with Phases 5-8

### 9.1 Implement CLI Flags

- [x] `--config <path>` - Custom config file path
- [x] `--version` - Show version
- [x] `--help` - Show usage

### 9.2 Implement TTY Validation

Per `configuration/spec.md`:

- [x] Check if stdin is TTY using `term.IsTerminal()`
- [x] Exit with error: "Interactive terminal required. Run in a terminal window."

### 9.3 Implement Terminal Size Validation

Per `configuration/spec.md`:

- [x] Get terminal dimensions
- [x] Require minimum 80x24
- [x] Exit with error showing current vs required dimensions: "Terminal too small (minimum 80x24)"

### 9.4 Implement Startup Sequence

Per `configuration/spec.md`:

- [x] Parse CLI flags
- [x] Validate TTY
- [x] Validate terminal size
- [x] Check gh CLI installed (`config.CheckGHCLI`) - show "gh CLI not found. Install from https://cli.github.com"
- [x] Check gh CLI authenticated (`config.CheckGHAuth`) - instruct to run `gh auth login`
- [x] Load config (or run wizard if `ErrConfigNotFound`)
- [x] Validate config (`config.Validate`)
- [x] Create GitHub client (`github.NewClient`)
- [x] Create TUI model
- [x] Start Bubble Tea program with alt screen

### 9.5 Implement Graceful Shutdown

- [x] Handle SIGINT/SIGTERM
- [x] Clean up resources
- [x] Restore terminal
- [x] Exit with appropriate code

### 9.6 Write Tests

- [x] Test flag parsing
- [x] Test error paths

### Files

- `/home/jgordijn/projects/gpr-management/cmd/pr-dashboard/main.go` - Entry point (180 lines)
- `/home/jgordijn/projects/gpr-management/cmd/pr-dashboard/main_test.go` - Basic tests (2 tests)

---

## Phase 10: Integration Testing - LOWER PRIORITY

**Depends on:** Phases 4-9

**Note:** View rendering tests have been added in `view_test.go` (36 tests), significantly reducing the testing gap for TUI components. These tests cover display modes, status bar, modals, PR list rendering, and edge cases.

- [ ] Create test fixtures (sample PR data, mock responses)
- [ ] Write end-to-end tests
- [ ] Write edge case tests:
  - Empty PR list
  - Single PR
  - Rate limit exceeded
  - Network errors (show in status bar, keep data per github-integration/spec.md)
  - Invalid API responses
- [x] Add missing unit tests for config loading functions (Load, LoadFromPath, DefaultConfigPath) - **DONE in v0.0.6**

---

## Phase 11: Documentation - LOWER PRIORITY

- [ ] Create README.md:
  - Project description and features
  - Installation instructions (go install, make install)
  - Prerequisites (gh CLI, authentication)
  - Quick start guide
  - Configuration reference
  - Key bindings reference
- [ ] Add screenshots/GIFs

---

## Phase 12: Polish & Release - LOWER PRIORITY

- [x] Add loading spinners (initial, refresh, action)
- [x] Implement error recovery (retry failed requests) - **DONE in v0.0.9**
- [ ] Build for multiple platforms (Linux, macOS, Windows)
- [ ] Create GitHub release workflow
- [ ] Final manual testing on all platforms
- [x] Run `go vet` and `staticcheck` - **DONE in v0.0.7**
- [x] Strip ANSI/control characters from gh output in TUI display - **DONE in v0.0.7** (deferred from Phase 3)

---

## Priority Order

1. **Phase 4 (Domain Model)** - CRITICAL: Foundation for all TUI work, blocking dependency
2. **Phase 5.0 (Add Charm dependencies)** - Required before any TUI code can compile
3. **Phase 5 (TUI Foundation)** - Core application structure
4. **Phase 9 (Entry Point)** - Make application runnable (can partially parallel with 5-8)
5. **Phases 6-8 (TUI Components & Features)** - Build out functionality
6. **Phases 10-12 (Testing, Docs, Polish)** - Production readiness

---

## Dependencies Graph

```
Phase 4 (Domain Model) --------------------------------+
                                                       |
                                                       v
Phase 5.0 (Add Charm deps) --> Phase 5 (TUI) --> Phase 6 (List) --> Phase 8 (Features)
                                    |
                                    +--> Phase 7 (Modals) --> Phase 8 (Features)
                                    |
                                    +--> Phase 9 (Entry Point - partial)

Phase 8 (Features) --> Phase 10 (Integration Tests)
Phase 9 (Entry Point) --> Phase 10 (Integration Tests)
Phase 10 --> Phase 11 (Documentation)
Phase 11 --> Phase 12 (Polish & Release)
```

---

## Current go.mod Dependencies

**Present:**
- `github.com/BurntSushi/toml v1.4.0`
- `github.com/cli/go-gh/v2 v2.11.1`

**Required for TUI (must add in Phase 5.0):**
- `github.com/charmbracelet/bubbletea`
- `github.com/charmbracelet/lipgloss`
- `github.com/charmbracelet/bubbles`

---

## Code Quality Checklist (Per Phase)

- [ ] No TODOs, FIXMEs, or panic("not implemented")
- [ ] All public functions have doc comments
- [ ] Unit tests with reasonable coverage
- [ ] No hardcoded strings (use constants)
- [ ] Error messages are actionable
- [ ] Follows existing code patterns in internal/config and internal/github

---

## Code Quality Status

Current state (Phases 1-3):
- No TODOs, FIXMEs, or placeholders in implemented code
- No panic("not implemented") calls
- 4 legitimate t.Skip calls in actions_test.go (timing-sensitive tests in short mode)
- Comprehensive test coverage: 61 config tests + 32 github tests = 93 total

---

## Acceptance Criteria (from Specifications)

### Configuration (configuration/spec.md) - COMPLETE
- [x] Application loads config from `~/.config/pr-dashboard/config.toml`
- [x] Apply default values for optional fields
- [x] Validate required fields (username, organizations)
- [x] Validate refresh_interval between 10-300
- [x] Validate initial_mode is full/compact/minimal
- [x] First-run wizard creates valid config
- [x] Detects missing `gh` CLI and shows installation instructions
- [x] Shows error if `gh` CLI not authenticated
- [x] Config file supports multiple organizations

### GitHub Integration (github-integration/spec.md) - COMPLETE
- [x] GraphQL search query with pagination
- [x] Rate limit tracking and warning at <=100 remaining
- [x] Network error handling (keep data, show error)

### TUI Navigation (tui-navigation/spec.md) - COMPLETE
- [x] j/k and arrow key navigation
- [x] gg/G for jump to top/bottom
- [x] Skip collapsed organizations during navigation
- [x] Selection preserved by stable key across refreshes
- [x] Selection moves to nearest visible when current disappears
- [x] o/O for org collapse toggle
- [x] Enter opens PR in browser via gh pr view --web
- [x] r for manual refresh with loading indicator
- [x] ? shows help modal
- [x] q/Escape exits application

### Watch Mode (watch-mode/spec.md) - COMPLETE
- [x] w toggles watch mode
- [x] Auto-refresh at configured interval (default 30s)
- [x] Watch mode suspended during branch update action
- [x] Change detection (status changes, new PRs)
- [x] Ignore time-derived changes unless crossing day boundary
- [x] Highlight changed PRs (clears after 2 seconds)
- [x] Last refresh time displayed in status bar

### Branch Actions (branch-actions/spec.md) - COMPLETE
- [x] UpdateBranch API implementation with `--repo` flag
- [x] TUI: u key triggers update when eligible (BEHIND + MERGEABLE)
- [x] TUI: Modal feedback for ineligible states (including draft PR check)
- [x] TUI: Success/failure modal display
- [x] TUI: Block concurrent actions
- [x] TUI: Queue refresh during action

### PR Display (pr-display/spec.md) - COMPLETE
- [x] PRs grouped by organization with collapse/expand
- [x] PRs sorted by updatedAt within each org
- [x] Shows CI check status using aggregate state
- [x] Shows review status with reviewer names
- [x] Shows unresolved thread count with "+" suffix when truncated
- [x] Shows days since PR opened
- [x] Shows merge status (all states with correct colors)
- [x] Draft PRs clearly marked with badge and dimmed
- [x] Display modes cycle correctly (full/compact/minimal)
- [x] Title truncation with ellipsis
- [x] Empty state: "No open PRs - nice work! :tada:"
- [x] Team reviewers shown with "team:" prefix

### Terminal Requirements (configuration/spec.md) - COMPLETE
- [x] Detects non-TTY and exits with error
- [x] Shows minimum terminal size message when too small (80x24)
- [x] `make install` installs to `~/.local/bin`
