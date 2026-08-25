# pr-dashboard

A terminal UI for monitoring your GitHub pull requests across one or more organizations.

`pr-dashboard` shows your open PRs in a single full-screen dashboard with status, review state, unresolved threads, merge readiness, watch mode, branch update actions, and GitHub account switching.

## Features

- Project identity on every pull request, with title-first placement in repository view
- A compact CI/review/merge symbol triad with a selected-row decoder
- Switch between organization grouping and an organization-rooted repository tree
- Repository-tree rows place project identity after the PR title
- Left-click a PR to open it; click organization/repository nodes to toggle them
- Persistently hide individual repositories or PRs, with undo and a searchable restore manager
- Sort PRs by name, age, or readiness state in either direction
- Restore each account's last grouping, display, drafts, sort, focus, and collapse setup
- Vim-style, arrow-key, and mouse navigation
- Responsive full, compact, and minimal display modes
- Draft visibility toggle
- Watch mode with refresh + change highlighting
- Open the selected PR in your browser
- Update a behind branch with `gh pr update-branch`
- Switch between authenticated `gh` accounts from inside the TUI
- First-run setup wizard when no config file exists

## Prerequisites

Before running `pr-dashboard`, make sure you have:

- Go 1.21+
- [GitHub CLI](https://cli.github.com/) installed
- A valid GitHub CLI login via `gh auth login`
- An interactive terminal (TTY)
- A terminal size of at least 80x24

## Installation

### Option 1: Install with Make

```bash
make install
```

This builds the binary and installs it to `~/.local/bin/pr-dashboard`.

### Option 2: Install with Go

```bash
go install ./cmd/pr-dashboard
```

### Build and run locally

```bash
make build
./pr-dashboard
```

## Configuration

Default config path:

```text
~/.config/pr-dashboard/config.toml
```

Hidden repository/PR rules and the last account-scoped dashboard setup are stored separately and atomically in:

```text
~/.config/pr-dashboard/hidden.json
~/.config/pr-dashboard/view-state.json
```

The restored setup includes grouping, display density, draft visibility, sort field/direction, focus, and all collapse states.

If the config file does not exist, `pr-dashboard` starts an interactive setup wizard.

You can also copy the example config:

```bash
mkdir -p ~/.config/pr-dashboard
cp config.example.toml ~/.config/pr-dashboard/config.toml
```

Example configuration:

```toml
[general]
username = "your-github-username"
refresh_interval = 30

[[organizations]]
login = "your-org"

[display]
show_drafts = true
initial_mode = "full"
grouping = "organization" # or "repository"
ascii = false

[notifications]
highlight_changes = true
```

### Configuration fields

- `general.username`: required GitHub login
- `general.refresh_interval`: refresh interval in seconds, valid range `10-300`, default `30`
- `organizations[].login`: one or more GitHub organizations to monitor
- `display.show_drafts`: show draft PRs, default `true`
- `display.initial_mode`: `full`, `compact`, or `minimal`, default `full`
- `display.grouping`: initial grouping projection, `organization` or `repository`, default `organization`
- `display.ascii`: use a pure-ASCII status vocabulary for terminals with ambiguous Unicode widths, default `false`
- `notifications.highlight_changes`: briefly highlight changed PRs after refresh, default `true`

## Status language

Each row keeps the repository and a fixed three-slot status triad visible:

| Slot | Good | Attention | Bad | None/unknown |
|---|---|---|---|---|
| CI | `✓` passing | `◐` pending | `✗` failing | `·` none |
| Review | `✓` approved | `?` required | `!` changes | `·` none |
| Merge | `✓` ready | `↓` behind, `⊘` blocked, `~` unstable | `≠` conflicts | `?` unknown, `○` draft |

The gutter uses `▶` for the selected row and `●` for recently changed data. `◈n` is the unresolved-thread count. The status bar spells out the selected row's symbols, and `?` opens the complete legend. Set `display.ascii = true` for the equivalent ASCII vocabulary.

Repository grouping retains the organization as the tree root:

```text
▾ RoyalAholdDelhaize 3
  ▾ agentic-demo 1
    └─ #1 OpenSpecs: AH Groceries Agent  agentic-demo  ✓ ? ✓
  ▾ ahold-a2a 2
    ├─ #34 Add A2A MCP bridge           ahold-a2a     ✓ ✓ ✓
    └─ #33 Add deployment mappings      ahold-a2a     ✓ ✓ ✓
```

## Usage

Run the dashboard:

```bash
pr-dashboard
```

Use a custom config file:

```bash
pr-dashboard --config /path/to/config.toml
```

Show help or version:

```bash
pr-dashboard --help
pr-dashboard --version
```

## Keybindings

### Navigation

- Left-click a PR row: focus and open that exact PR in the browser
- Left-click an organization or repository node: focus and toggle it
- `j` / `↓`: move down
- `k` / `↑`: move up
- `gg`: go to top
- `G`: go to bottom
- `o`: toggle the current organization, or the current repository in repository view
- `O`: toggle all organizations, or all repositories in repository view
- `h` / `←`: focus the parent repository or collapse it
- `l` / `→`: expand a repository or focus its first PR

### Actions

- `Enter`: open selected PR in browser
- `r`: refresh
- `u`: update branch when the PR is behind and updateable
- `s`: switch active GitHub account

### Display

- `c`: cycle display mode (`full -> compact -> minimal`)
- `v`: toggle organization/repository grouping without refetching
- `t`: cycle PR sorting through `name → age → state`
- `T`: toggle ascending/descending sort direction
- `H`: persistently hide the focused repository or PR
- `z`: undo the latest successful hide from this session
- `M`: open the Hidden Items manager; use `/` search, `Tab` type filter, and `u`/Enter to restore
- `d`: toggle draft visibility
- `w`: toggle watch mode

### Sorting

Sorting affects PR siblings without reordering organization or repository nodes:

- `name ↑/↓`: title A→Z or Z→A
- `age ↑/↓`: youngest→oldest or oldest→youngest
- `state ↑/↓`: healthy→critical or critical→healthy

State severity considers conflicts/dirty state, failing CI, requested changes, blocked/behind status, pending review/checks, unresolved threads, unknown state, and drafts. Equal rows always use deterministic identity tie-breakers.

### Other

- `?`: show help
- `q` / `Esc`: quit

## Troubleshooting

### `gh CLI not found`

Install GitHub CLI:

```bash
https://cli.github.com/
```

### `gh CLI not authenticated`

Authenticate with GitHub CLI:

```bash
gh auth login
```

### `Interactive terminal required`

Run `pr-dashboard` in a real terminal instead of piping input or running in a non-interactive environment.

### `Terminal too small`

Resize your terminal to at least `80x24`.

### Configuration validation errors

Check that:

- `username` is set
- at least one `[[organizations]]` entry exists
- `refresh_interval` is between `10` and `300`
- `initial_mode` is one of `full`, `compact`, or `minimal`

### No pull requests shown

Verify that:

- the configured username matches the GitHub account you want to query
- the configured organizations are correct
- you have open PRs authored by that user in those organizations
- the active `gh` account has access to the relevant repositories

### Watch mode or refresh errors

Transient GitHub API or network errors are shown in the status bar. Existing data stays visible so you can retry with `r`.

## Development

Run tests:

```bash
make test
```

Build the binary:

```bash
make build
```

Run locally:

```bash
make run
```

## Manual smoke test plan

Use this checklist when validating the dashboard manually:

- [ ] Start with `gh` missing and verify the install error is shown
- [ ] Start with `gh` unauthenticated and verify the login guidance is shown
- [ ] Start with no matching open PRs and verify the empty state appears
- [ ] Open a PR in the browser with `Enter`
- [ ] Trigger `u` on an eligible PR and verify success/failure feedback
- [ ] Toggle watch mode with `w` and observe stable refresh behavior for 5+ minutes
- [ ] Toggle drafts, display modes, and organization collapse/expand
- [ ] Open the account picker with `s` and switch accounts if multiple accounts are configured
