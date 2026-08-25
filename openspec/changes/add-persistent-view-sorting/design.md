## Context
Runtime view state is spread across grouping, display, filtering, selection, and three collapse representations. PR ordering is currently fixed by fetch/group helpers. Hidden-item persistence provides an established atomic, account-scoped storage pattern.

## Goals / Non-Goals
- Goals: exact session restoration, account-scoped preferences, deterministic multi-field PR sorting, visible controls, safe fallback on stale focus/collapse keys, atomic persistence.
- Non-goals: changing repository/organization node ordering, server-side GitHub sorting, or persisting transient loading/error/modal/manager/search/watch state.

## Decisions
- Persist per account in versioned `~/.config/pr-dashboard/view-state.json`.
- Persist grouping, display mode, draft visibility, sort field/direction, selected key, organization-view collapse, tree-organization collapse, and repository collapse.
- Do not persist transient watch/action/loading/modal/hidden-manager state.
- Sort applies to PR leaves within their current organization/repository parent; organization and repository nodes keep deterministic name order.
- Sort fields: `name` compares title case-insensitively; `age` compares CreatedAt/DaysOpen; `state` compares a documented worst-first readiness severity. Stable PR key is the final tie-breaker.
- Accounts without saved state default to `age` ascending (youngest first), matching the dashboard's freshness-oriented behavior.
- `t` cycles `name → age → state`; `T` toggles ascending/descending. Status chrome always shows the active field and arrow.
- Restored stale focus falls back through existing nearest-visible rules. Stale collapse keys are harmless and retained because temporarily absent repositories may return.
- State mutations commit in memory and save atomically; save failures keep the new runtime view usable but surface a persistent warning, with retry on subsequent mutations.

## Risks / Trade-offs
- A failure after an in-memory view change can make disk lag until retry. Keeping the user's requested runtime action is less surprising than rolling back navigation/view state.
- Severity ordering combines CI/review/merge. A single pure rank function and exhaustive tests make it explainable and stable.

## Migration Plan
Missing state loads defaults from config. Corrupt/unsupported state is not overwritten automatically; defaults apply and a visible warning explains persistence is unavailable.
