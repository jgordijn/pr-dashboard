# Change: Replace verbose PR rows with a project-first status language

## Why
PR rows currently omit the repository despite already carrying it in the model, and spend dozens of terminal cells on repeated status words. Users cannot quickly identify the project or scan review, CI, freshness, conflict, and draft state.

## What Changes
- Put `repository#number` on every PR row.
- Replace repeated status words with a fixed CI/review/merge glyph triad and a two-cell selection/change gutter.
- Add width-aware, grapheme-safe rows and a contextual plain-language status decoder.
- Add a compact legend to help and a pure-ASCII symbol option.
- Preserve conflicts on draft PRs and make collapsed organization headers summarize hidden risk.

## Impact
- Affected specs: `pr-display`, `tui-navigation`, `configuration`
- Affected code: `internal/tui`, `internal/model`, `internal/config`, example configuration and README
