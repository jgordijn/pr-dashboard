# pr-dashboard

A terminal UI for monitoring your GitHub pull requests across one or more organizations.

`pr-dashboard` shows your open PRs in a single full-screen dashboard with status, review state, unresolved threads, merge readiness, watch mode, branch update actions, and GitHub account switching.

## Features

- Group pull requests by organization
- Vim-style and arrow-key navigation
- Full, compact, and minimal display modes
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

[notifications]
highlight_changes = true
```

### Configuration fields

- `general.username`: required GitHub login
- `general.refresh_interval`: refresh interval in seconds, valid range `10-300`, default `30`
- `organizations[].login`: one or more GitHub organizations to monitor
- `display.show_drafts`: show draft PRs, default `true`
- `display.initial_mode`: `full`, `compact`, or `minimal`, default `full`
- `notifications.highlight_changes`: briefly highlight changed PRs after refresh, default `true`

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

- `j` / `↓`: move down
- `k` / `↑`: move up
- `gg`: go to top
- `G`: go to bottom
- `o`: toggle current organization
- `O`: toggle all organizations

### Actions

- `Enter`: open selected PR in browser
- `r`: refresh
- `u`: update branch when the PR is behind and updateable
- `s`: switch active GitHub account

### Display

- `c`: cycle display mode (`full -> compact -> minimal`)
- `d`: toggle draft visibility
- `w`: toggle watch mode

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
