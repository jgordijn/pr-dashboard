## Context
Visibility is currently derived from fetched PRs, draft filtering, grouping, collapse, and terminal clipping. Hide rules must precede every projection so rendering, navigation, and mouse targets cannot diverge.

## Goals / Non-Goals
- Goals: persistent repository and PR exclusions, reversible hide workflow, searchable selective restore manager, account scoping, atomic storage, exact cross-view filtering, clear feedback.
- Non-goals: hiding whole organizations, remote synchronization, bulk destructive restore, or changing GitHub queries.

## Decisions
- `H` hides the focused repository or PR. `Ctrl+h` remains avoided because terminals encode it as Backspace and plain `h` is tree navigation.
- `M` opens a full-screen Hidden Items manager. `z` undoes the latest successful hide in this process.
- Rules are independent: restoring a repository does not restore explicitly hidden PR rules beneath it.
- Stable matching is case-insensitive `owner/repository` and `owner/repository#number`; saved entries preserve display casing/title snapshots.
- State is account-scoped in `~/.config/pr-dashboard/hidden.json`, versioned JSON, written atomically with restrictive permissions.
- Persistence is injected. Hide/unhide operations commit only after successful save; failures preserve prior visibility and show an error.
- Filtering happens before draft projection, grouping, counts, layout, navigation, rollups, and mouse targets. Future PRs in hidden repositories are hidden automatically.
- The manager is a dedicated full-screen view with newest-first rows, All/Repositories/PR filters, case-insensitive search, keyboard selection, and exact-rule unhide.

## Risks / Trade-offs
- Additional view state increases update complexity. Dedicated pure manager/filter helpers and injected storage keep it testable.
- Corrupt state must not be overwritten silently. Startup continues with persistence disabled and a visible error until the file is repaired.

## Migration Plan
Missing state files load as empty. No existing config changes. Removing the feature leaves the separate JSON state file harmlessly unused.
