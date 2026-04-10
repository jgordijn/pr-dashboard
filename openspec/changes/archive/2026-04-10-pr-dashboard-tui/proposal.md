# PR Dashboard TUI

## Why

Managing multiple open PRs across repositories in a large organization is cumbersome. You have to:
1. Navigate to github.com/pulls manually
2. Click into each PR to see CI status, review status, and comments
3. Check if branches are up-to-date with their base branch
4. Manually update branches when behind
5. Constantly refresh to see changes

This leads to context-switching overhead and delayed responses to PR feedback. A terminal-based dashboard provides:
- At-a-glance status of all PRs
- Real-time updates via watch mode
- Quick actions (update branch, open in browser) without leaving the terminal
- Integration with existing `gh` CLI authentication

## What Changes

This is a **new application** - building from scratch.

### New Application: `pr-dashboard`
A Go-based TUI application using the Charm stack (Bubble Tea, Lip Gloss, Bubbles) that:
- Queries GitHub GraphQL API for open PRs authored by a specific user
- Displays PRs grouped by organization with collapsible sections
- Shows rich status information (CI checks, reviews, comments, merge status)
- Provides interactive actions (update branch, open in browser)
- Supports watch mode with auto-refresh and change highlighting
- Uses TOML configuration for customization

### Key Capabilities
1. **PR List View**: Grouped by organization, scrollable, with vim-style navigation
2. **Status Display**: CI checks (passing/failing/running), review status, unresolved review threads, days open
3. **Branch Status**: Clean, behind base branch, conflicts - visual indicators
4. **Actions**: Update branch (with modal feedback), open in browser, refresh
5. **Watch Mode**: Auto-refresh every N seconds, highlight changes
6. **First-Run Wizard**: Interactive setup when no config exists
7. **Display Mode Toggle**: Cycle between full/compact/minimal display modes
8. **Organization Toggle**: Collapse/expand org sections

## Impact

### New Dependencies
- `github.com/charmbracelet/bubbletea` - TUI framework
- `github.com/charmbracelet/lipgloss` - Styling
- `github.com/charmbracelet/bubbles` - Pre-built components
- `github.com/cli/go-gh/v2` - GitHub API access
- `github.com/BurntSushi/toml` - Config parsing

### Prerequisites
- Go 1.21+
- `gh` CLI installed and authenticated (github.com only, Enterprise hosts out of scope)
- `~/.local/bin` in PATH (for installation)
- Interactive terminal (TTY required)

### Files Created
```
~/projects/pr-management/
├── cmd/
│   └── pr-dashboard/
│       └── main.go                 # Entry point, CLI flags
├── internal/
│   ├── config/
│   │   ├── config.go               # TOML config loading & defaults
│   │   ├── wizard.go               # First-run setup wizard
│   │   └── auth.go                 # gh CLI authentication check
│   ├── github/
│   │   ├── client.go               # GitHub GraphQL client
│   │   ├── queries.go              # GraphQL query definitions
│   │   ├── types.go                # API response types
│   │   └── actions.go              # Branch update execution
│   ├── model/
│   │   ├── pr.go                   # Domain model for PRs
│   │   └── diff.go                 # Change detection between refreshes
│   └── tui/
│       ├── app.go                  # Main Bubble Tea model
│       ├── styles.go               # Lip Gloss color schemes
│       ├── list.go                 # PR list view component
│       ├── modal.go                # Modal dialog component
│       ├── statusbar.go            # Bottom status bar
│       ├── keys.go                 # Key bindings
│       └── update.go               # Update logic (Msg handling)
├── go.mod
├── go.sum
├── Makefile
├── README.md
└── config.example.toml
```

### Config File Location
`~/.config/pr-dashboard/config.toml`

## Design Decisions

### Decision 1: Charm Stack for TUI
**Choice:** Bubble Tea + Lip Gloss + Bubbles
**Rationale:** Modern, well-maintained, beautiful defaults, active community. The Elm-architecture (Model-View-Update) makes state management predictable. Lip Gloss provides excellent terminal styling with color support.
**Alternatives considered:** 
- `tview` - More widget-focused, less flexible for custom UIs
- `termui` - Less active development, fewer styling options

### Decision 2: GraphQL over REST API
**Choice:** Use GitHub GraphQL API
**Rationale:** Single request fetches all needed data (PR details, checks, reviews, comments). REST would require N+1 requests. GraphQL provides exactly the fields we need.
**Query approach:** Use search query `is:pr is:open author:{user} org:{org} archived:false`

### Decision 3: TOML for Configuration
**Choice:** TOML format at `~/.config/pr-dashboard/config.toml`
**Rationale:** Modern CLI tools favor TOML. More explicit than YAML, no whitespace sensitivity issues. Well-supported in Go.
**Alternatives considered:**
- YAML - More error-prone due to whitespace
- JSON - No comments, less human-friendly

### Decision 4: go-gh Library
**Choice:** Use `github.com/cli/go-gh/v2` for GitHub API access
**Rationale:** Official GitHub CLI library, automatically uses `gh` CLI authentication, handles token refresh, provides GraphQL client.
**Alternatives considered:**
- Direct HTTP with personal token - Would require separate auth management
- `google/go-github` - REST-focused, would need separate GraphQL setup

### Decision 5: Organization Grouping with Collapsible Sections
**Choice:** Group PRs by organization with toggle to collapse/expand
**Rationale:** Users often work across multiple orgs. Grouping provides visual separation. Collapse saves space when not interested in certain orgs.

### Decision 6: First-Run Interactive Setup
**Choice:** Interactive wizard prompts for username, orgs, refresh interval when no config exists
**Rationale:** Better UX than requiring manual config file creation. Gets users started quickly.
**Detection:** Also check if `gh` CLI is authenticated and installed, show helpful error if not.

### Decision 7: Modal for Action Feedback
**Choice:** Use modal popups for update branch success/failure
**Rationale:** Actions like branch updates can fail. Modal ensures user sees the result without scrolling or missing inline messages.

### Decision 8: Watch Mode with In-Place Updates
**Choice:** Auto-refresh preserves selection by stable PR identifier, highlights changed rows
**Rationale:** Jumping around on refresh would be disorienting. Users want to monitor while staying in context. Selection tracked by `owner/repo#number` key, not list index.

### Decision 9: Selection Tracking by Stable Identifier
**Choice:** Track selected PR by `owner/repo#number` composite key
**Rationale:** List indices change when PRs are added/removed or sort order changes. Using a stable identifier ensures selection is preserved across refreshes. If the selected PR disappears (merged/closed), select the nearest visible PR.

### Decision 10: Unresolved Thread Count Display
**Choice:** Display exact unresolved count from sampled threads, with "+" suffix when truncated
**Rationale:** Query fetches first 100 thread nodes plus `totalCount`. Count unresolved from returned nodes. When `totalCount > 100`, append "+" suffix (e.g., "15+") to indicate there may be more unresolved threads not sampled. This provides accuracy for most PRs while clearly indicating when sampling occurred.

### Decision 11: Update Branch Command Form
**Choice:** Execute `gh pr update-branch <number> --repo <owner/name>`
**Rationale:** Explicit `--repo` flag avoids CWD dependency and works regardless of where the dashboard is run from.

### Decision 12: Concurrency Control for Actions
**Choice:** While an update-branch action is in flight, suspend watch-mode refresh and show busy state
**Rationale:** Prevents race conditions where refresh overwrites pending operation state. Queue refresh to run after action completes.

## Edge Cases & Error Handling

| Scenario | Expected Behavior | Notes |
|----------|-------------------|-------|
| No config file exists | Launch first-run wizard | Creates config interactively |
| `gh` CLI not installed | Show error: "gh CLI not found. Install from https://cli.github.com" | Check `exec.LookPath("gh")` |
| `gh` CLI not authenticated | Show error with instructions | "Run `gh auth login` first" |
| Network failure on refresh | Show error in status bar, keep old data | Don't clear existing view |
| No open PRs | Show friendly message | "No open PRs - nice work! 🎉" |
| PR has conflicts | Show ✗ Conflicts, disable update | Update button shows why disabled |
| PR mergeable status UNKNOWN | Show "?" indicator, disable update | GitHub hasn't computed merge status yet |
| Update branch fails | Show modal with error details | Include API error message |
| GraphQL rate limit | Show warning when remaining <= 100, display reset time | Continue with cached data, allow manual refresh |
| Organization has no PRs | Still show org header (collapsed) | Indicates org is being watched |
| Very long PR title | Truncate with ellipsis | Max width based on terminal |
| Terminal resize | Re-render with new dimensions | Bubble Tea handles this |
| Terminal too small | Show message "Terminal too small (min 80x24)" | Disable complex rendering |
| No TTY (non-interactive) | Exit with error: "Interactive terminal required" | Detect `os.Stdin.Fd()` |
| Watch mode with no changes | Update "Last refresh" timestamp only | No visual flash if unchanged |
| Selected PR disappears | Move selection to nearest visible PR | Merged/closed PR removed |
| Draft toggle hides selected | Move selection to nearest visible PR | Keep UX predictable |
| More than 100 review threads | Show count with "+" suffix (e.g., "15+") | Use totalCount to detect truncation |
| More than 20 check contexts | Use aggregate `statusCheckRollup.state` | Don't paginate contexts |
| Review requests include Teams | Display as "team:<name>" | Handle GraphQL union |
| Update during watch refresh | Suspend refresh, queue for after action | Prevent race conditions |

## Technical Details

### GitHub GraphQL Query
```graphql
query($searchQuery: String!, $cursor: String) {
  search(query: $searchQuery, type: ISSUE, first: 50, after: $cursor) {
    nodes {
      ... on PullRequest {
        id
        number
        title
        url
        isDraft
        createdAt
        updatedAt
        repository { nameWithOwner }
        baseRefName
        headRefName
        mergeable
        mergeStateStatus
        reviewDecision
        reviews(last: 10) {
          nodes { author { login } state }
        }
        reviewRequests(first: 10) {
          nodes { 
            requestedReviewer { 
              ... on User { login }
              ... on Team { name slug }
            } 
          }
        }
        comments { totalCount }
        reviewThreads(first: 100) {
          totalCount
          nodes { isResolved }
        }
        commits(last: 1) {
          nodes {
            commit {
              statusCheckRollup {
                state
                contexts(first: 20) {
                  nodes {
                    ... on CheckRun { name conclusion status }
                    ... on StatusContext { context state }
                  }
                }
              }
            }
          }
        }
      }
    }
    pageInfo { hasNextPage endCursor }
  }
  rateLimit {
    remaining
    resetAt
    cost
  }
}
```

### Merge Status Mapping
| GitHub Fields | Display Status | Update Enabled |
|---------------|----------------|----------------|
| `mergeable=MERGEABLE`, `mergeStateStatus=CLEAN` | ✓ Clean | No (already current) |
| `mergeable=MERGEABLE`, `mergeStateStatus=BEHIND` | ⚠ Behind | **Yes** |
| `mergeable=MERGEABLE`, `mergeStateStatus=BLOCKED` | ⚠ Blocked | No |
| `mergeable=CONFLICTING` | ✗ Conflicts | No |
| `mergeable=UNKNOWN` | ? Unknown | No (wait for GitHub) |
| `mergeStateStatus=DIRTY` | ✗ Dirty | No |
| `mergeStateStatus=UNSTABLE` | ⚠ Unstable | No |
| `mergeStateStatus=HAS_HOOKS` | ⚠ Has Hooks | No |
| `mergeStateStatus=null or UNKNOWN` | ? Unknown | No |
| Any other/unknown value | ? Unknown | No |

### Configuration Schema
```toml
[general]
username = "jgordijn"              # Required: GitHub username (login)
refresh_interval = 30              # Optional: seconds, default 30, min 10, max 300

[[organizations]]
login = "organisation"             # Required: GitHub org login (used in search query)

[[organizations]]
login = "another-org"              # Can have multiple

[display]
show_drafts = true                 # Optional: default true
initial_mode = "full"              # Optional: "full", "compact", "minimal", default "full"

[notifications]
highlight_changes = true           # Optional: default true
```

**Config Validation Rules:**
- `username` is required, must be non-empty
- At least one `[[organizations]]` block required
- `organizations.login` is the GitHub org login (used in `org:` search filter)
- `refresh_interval` must be between 10 and 300 seconds
- `initial_mode` must be one of: "full", "compact", "minimal"

### Key Bindings
| Key | Action |
|-----|--------|
| `↑/k` | Move selection up |
| `↓/j` | Move selection down |
| `gg` | Jump to top |
| `G` | Jump to bottom |
| `Enter` | Open PR in browser |
| `u` | Update branch (if behind base and mergeable) |
| `r` | Manual refresh |
| `c` | Cycle display mode (full → compact → minimal) |
| `d` | Toggle show/hide drafts |
| `o` | Toggle current org visibility |
| `O` | Toggle all orgs visibility |
| `w` | Toggle watch mode |
| `?` | Show help modal |
| `q/Esc` | Quit (or close modal) |

### Display Modes
1. **Full**: All columns - checks detail, reviews with names, unresolved threads, age, merge status, repo
2. **Compact**: Summary icons only - ✓/⏳/✗ for checks, review count, thread count
3. **Minimal**: PR title, status badge (Draft/Ready), age only

### Sort Order
PRs sorted within each organization by `updatedAt` descending (most recently updated first).

### Color Scheme
- **Green**: Approved, passing, clean, ready to merge
- **Yellow**: Pending review, checks running, behind base branch
- **Red**: Changes requested, checks failing, conflicts
- **Dim/Gray**: Draft indicator, secondary info, disabled items

## Open Questions

1. **Repository filtering**: Should we add per-org `repositories` filtering in the future?
   - **Decision**: Defer to v2. Not in current schema. Filtering can be added later without breaking changes.

2. **Sorting customization**: Should users be able to change sort order (e.g., by age, by status)?
   - **Decision**: Default to `updatedAt` desc for v1. Add config option in v2 if requested.

3. **Enterprise GitHub hosts**: Should we support GitHub Enterprise?
   - **Decision**: Out of scope for v1. Document as github.com only. Can add `host` config later.

## Acceptance Criteria

- [ ] Application compiles and runs on Linux
- [ ] Detects missing `gh` CLI and shows installation instructions
- [ ] First-run wizard creates valid config file
- [ ] Shows error if `gh` CLI not authenticated
- [ ] Lists all open PRs for configured user/orgs
- [ ] PRs grouped by organization with collapse/expand
- [ ] PRs sorted by updatedAt within each org
- [ ] Shows CI check status (passing/failing/running) using aggregate state
- [ ] Shows review status (approved/pending/changes requested)
- [ ] Shows unresolved review thread count (with "+" suffix when truncated)
- [ ] Shows days since PR opened
- [ ] Shows merge status (clean/behind/blocked/conflicts/dirty/unstable/has_hooks/unknown)
- [ ] Draft PRs clearly marked, can be toggled
- [ ] Enter opens PR in browser
- [ ] `u` updates branch when behind base and mergeable (using `--repo` flag)
- [ ] Update disabled with explanation when not possible
- [ ] Update success/failure shown in modal
- [ ] Watch mode auto-refreshes at configured interval
- [ ] Watch mode suspends during branch update action
- [ ] Watch mode highlights changed PRs (clears after 2 seconds)
- [ ] Selection preserved by stable PR key across refreshes
- [ ] Selection moves to nearest visible when current disappears
- [ ] Display modes cycle correctly (full/compact/minimal)
- [ ] Help screen shows all keybindings
- [ ] Graceful handling of network errors (show in status bar, keep data)
- [ ] Shows rate limit warning with remaining quota when limited
- [ ] Config file supports multiple organizations
- [ ] Config validation with clear error messages
- [ ] Detects non-TTY and exits with error
- [ ] Shows minimum terminal size message when too small
- [ ] `make install` installs to `~/.local/bin`
