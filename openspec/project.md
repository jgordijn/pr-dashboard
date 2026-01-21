# Project Context

## Purpose
A TUI (Terminal User Interface) application for monitoring and managing GitHub Pull Requests across organizations. The tool provides a real-time dashboard showing PR status, approvals, CI checks, and enables actions like updating branches directly from the terminal.

## Tech Stack
- **Language**: Go 1.21+
- **TUI Framework**: Bubble Tea (Charm)
- **Styling**: Lip Gloss (Charm)
- **Components**: Bubbles (Charm pre-built components)
- **GitHub API**: go-gh (official GitHub CLI library)
- **Config**: BurntSushi/toml

## Project Conventions

### Code Style
- Standard Go formatting (`gofmt`)
- Follow Effective Go guidelines
- Package names: lowercase, single word
- Interface names: verb-er pattern (e.g., `Renderer`, `Fetcher`)
- Exported functions/types: PascalCase
- Unexported: camelCase

### Architecture Patterns
- **cmd/** - Entry points (main packages)
- **internal/** - Private application code
  - **config/** - Configuration loading
  - **github/** - GitHub API interactions
  - **model/** - Domain models
  - **tui/** - Terminal UI components
- Bubble Tea Model-View-Update pattern for TUI
- Dependency injection via struct composition

### Testing Strategy
- Table-driven tests
- Test files alongside source (`*_test.go`)
- Mock GitHub API for unit tests
- Integration tests for TUI components using Bubble Tea's test utilities

### Git Workflow
- Branch naming: `feature/<name>` or `<ticket>-<description>`
- Conventional commits encouraged
- PR-based workflow

## Domain Context
- **PR States**: Draft vs Ready, Review status, Merge status
- **GitHub API**: Uses GraphQL for efficient data fetching
- **Organizations**: Supports multiple GitHub organizations
- **Watch Mode**: Auto-refresh capability for real-time monitoring

## Important Constraints
- Must work with existing `gh` CLI authentication
- Config file location: `~/.config/pr-dashboard/config.toml`
- Binary installed to `~/.local/bin/pr-dashboard`
- Must handle network failures gracefully
- Should work offline with cached data where possible

## External Dependencies
- GitHub GraphQL API (requires `gh` CLI authentication)
- Local filesystem for config storage
