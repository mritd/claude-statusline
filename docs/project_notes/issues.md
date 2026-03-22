# Work Log

## Format

- Date, brief description, status

### 2026-03-21 - Fix todo sync bugs

- **Status**: Completed
- **Description**: Fixed three transcript parsing bugs: deleted status not filtered, batch detection failing across TaskUpdate gaps, missing completion events
- **Notes**: See bugs.md and ADR-005/007/008

### 2026-03-22 - Add statusline colors

- **Status**: Completed
- **Description**: Added ANSI colors to icons (● dot, ≡ running, ✓ completed). Extracted `internal/ansi` package to avoid render-module import cycle. Updated defaults: bar_style=diamond, modules include tools, newline on tools. Added dot_warn_tokens (200k) for context performance warning.
- **Notes**: Text colors unchanged; only icons get color. Fixed empty `()` bug in usage reset times.

### 2026-03-22 - Code review cleanup

- **Status**: Completed
- **Description**: Deleted dead `render/colors.go` (re-exports unused after ansi extraction). Moved color tests to `ansi/ansi_test.go`. Simplified `DefaultDotWarnTokens` from function to exported constant. Added `TestDotWarnTokens` covering threshold, below-threshold, and disabled cases.
