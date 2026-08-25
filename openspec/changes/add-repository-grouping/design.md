## Context
The current model stores organization groups and the TUI focuses PR keys only. Repository grouping adds a second tree projection over the same fetched PRs; it must not duplicate API state or weaken the aligned status language.

## Goals / Non-Goals
- Goals: optional repository tree, instant runtime switching, title-before-project rows, focusable/collapsible repository nodes, deterministic sorting, preserved focus/collapse, responsive Unicode and ASCII output.
- Non-goals: changing GitHub queries, persisting runtime toggles, or introducing organization nodes inside repository mode.

## Decisions
- `display.grouping` defaults to `organization`; `v` toggles the runtime projection.
- Repository mode derives organization roots and sorted repository children from organization PR data. Node keys use `org:<owner>` and `repo:<owner>/<repository>` and cannot collide with PR keys.
- Organization and repository nodes are focusable. PR actions are disabled naturally when no PR is focused.
- `j/k`, arrows, `gg/G` traverse visible organization nodes, repository nodes, and PR leaves. `h/l`, left/right, `o`, `O`, and Enter provide tree collapse/expand behavior.
- Repository rows use `gutter connector #number title repository [author] [age] triad [threads]`. One frame-wide measured layout aligns triads globally.
- Repository nodes are sorted case-insensitively by owner/repository with raw-key tie-breakers. Children sort by updated time descending with stable-key tie-breakers.
- Collapse state is stored independently by stable organization-tree and repository keys and survives refresh and view toggling. Organization-view collapse remains independent.
- Repository nodes with zero visible PRs after draft filtering are omitted.

## Risks / Trade-offs
- Focus can refer to a PR, repository node, or organization node. Collision-proof key prefixes and small pure visibility helpers contain this complexity.
- Repeating repository after the title is redundant with the parent node, but explicitly supports cross-row project scanning and the requested information order.
- The existing active status-row change overlaps the same capabilities. New requirements are expressed as additive repository-mode behavior rather than partial replacements.

## Migration Plan
Existing config omits `grouping` and remains in organization mode. Runtime toggling does not rewrite config. Rollback ignores the optional field only after removing it from user configs because unknown keys are rejected.
