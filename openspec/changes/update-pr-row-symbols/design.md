## Context
Rows repeat long labels such as `checks:passing`, `approved`, and `conflicts`, while omitting the repository. The model already exposes repository, review, CI, merge, draft, thread, author, and time data.

## Goals / Non-Goals
- Goals: project-first identity, shape-readable status, deterministic width handling, one visual language across modes, keyboard discoverability, ASCII fallback.
- Non-goals: vertical viewport/scrolling and new GitHub API data.

## Decisions
- Every row begins with a two-cell gutter, then `repository#number`.
- Every mode retains a positional CI/review/merge triad.
- Unicode mappings are CI `✓ ✗ ◐ ·`, review `✓ ! ? ·`, merge `✓ ↓ ⊘ ≠ ~ ? ○`; ASCII mappings are CI `+ x ~ -`, review `+ ! ? -`, merge `= v # X ~ ? o`.
- Conflict outranks draft in the merge slot; dim styling remains the draft channel.
- Plain-language status appears only for the selected row in the status bar and in help.
- Rows are built unstyled, measured in display cells, truncated by grapheme, then styled.
- Fields use one-cell separators. At narrower widths optional fields drop before the triad or identity.
- Collapsed organization headers alone show failing/behind/thread rollups.

## Risks / Trade-offs
- Unicode symbols have East Asian ambiguous width in some terminal configurations. Measured layout uses the same default width ruler as Lip Gloss; `display.ascii=true` is the escape hatch.
- Symbols require learning. The selected-row decoder and help legend make the mapping discoverable.

## Migration Plan
Existing configuration remains valid. Unicode is the default; ASCII is opt-in.
