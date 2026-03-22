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
- **Notes**: Text colors unchanged; only icons get color. Fixed empty `()` bug in usage reset times. `dot_warn_tokens` later replaced by `context_limit` (see below).

### 2026-03-22 - Code review cleanup

- **Status**: Completed
- **Description**: Deleted dead `render/colors.go` (re-exports unused after ansi extraction). Moved color tests to `ansi/ansi_test.go`. Simplified dot warn constant. (`DefaultDotWarnTokens` later renamed to `DefaultContextLimit`; `TestDotWarnTokens` replaced by `TestContextLimit`.)

### 2026-03-22 - Tools per-turn reset and error display

- **Status**: Completed
- **Description**: Tool stats now reset on each user message (only current turn shown). Error tools display with red `✗` icon. Session tool names (top 6 by recency) persist across resets; unused tools show dimmed `×0`. Bash command newlines stripped from target display. Single-pass JSON parsing via `json.RawMessage`.
- **Notes**: See ADR-010/011/012

### 2026-03-22 - Agent module improvements

- **Status**: Completed
- **Description**: Fixed `"Agent"` tool name not recognized (only `"Task"` was handled). Agent display name now uses Description (50 char) -> Type -> "Agent" fallback. Running agents show full cyan with `...`, no elapsed time. Completed agents show no color, only latest 1 kept. Running agents hide completed ones. Error agents show full red.

### 2026-03-22 - Replace dot_warn_tokens with context_limit

- **Status**: Completed
- **Description**: Renamed `dot_warn_tokens` to `context_limit` (default 250k). Bar and dot now calculate percentage against this limit instead of full window. Exceeds limit caps at 100%. Removed dead `BRIGHT_YELLOW` and `LAVENDER` from ansi package. Default modules now include agents and environment.
- **Notes**: See ADR-013

### 2026-03-22 - Remove todos module and add tail scan

- **Status**: Completed
- **Description**: Removed todos module (Claude Code's own UI handles this; JSONL data unreliable). Added tail scan optimization: `max_tail_size` config (default `"10MB"`) makes transcript parser seek to file tail for large JSONL files. Fixed scanner buffer overflow (1MB -> 16MB max) that caused agents to not be detected after large tool_result entries. Removed `SessionStart` field.
- **Notes**: See ADR-014, bugs.md scanner buffer entry

### 2026-03-22 - Session-wide tool counts and running tool dedup

- **Status**: Completed
- **Description**: Tool counts changed from per-turn to session-wide cumulative (`SessionToolCounts`). Per-turn status used only for highlight vs dim styling. Fixed running tools appearing as duplicate dimmed entries by excluding them from session tool list. See ADR-010 update.
