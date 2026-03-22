# Architectural Decisions

## ADR-001: Keychain via /usr/bin/security CLI (not cgo)

macOS keychain access uses `exec.CommandContext` with 3s timeout calling `/usr/bin/security find-generic-password -w`. This avoids the authorization popups that cgo's `SecKeychainFindGenericPassword` triggers, and enables `CGO_ENABLED=0` builds. Service name lookup order: resolved config dir, env var, legacy fallback (`Claude Code-credentials`).

## ADR-002: Cache TTL = API Rate Limit Window (300s)

Default cache TTL is 300s (5 min), matching Anthropic's usage API rate limit window. Non-429 failures use a shorter 15s TTL via `IsFailure` flag on cache entries. 429 backoff is exponential 60s->300s cap.

## ADR-003: Zero External Dependencies

The binary uses Go stdlib only. No external modules. This ensures fast builds, <5ms startup, and zero dependency maintenance. Trade-off: some features require more code (manual JSON parsing, no structured logging).

## ADR-004: Bar Chars via Closure Injection

`NewContextBar(filled, empty)` and `NewQuotaBar(filled, empty)` return closures that capture bar characters. No mutable package-level state in the render package. This is parallel-test safe and consistent with how other render functions (dim, color) are injected.

## ~~ADR-005/007/008: Todos Module~~ (Removed 2026-03-22)

Todos module removed. Claude Code's own task UI already displays todo progress, and TaskCreate/TaskUpdate events in JSONL are unreliable (model often skips completion events). The complexity of batch detection, auto-completion heuristics, and deleted-task filtering wasn't justified for data that was fundamentally incomplete.

## ADR-006: Configurable Icons via `icons` Map

All statusline icons (running, completed, error, todo, done, dirty) are configurable via `icons` map in config.json. Modules read icons through `ctx.Config.Icon(key)` instead of hardcoding. This allows users to fix terminal font alignment issues by swapping characters.

## ADR-009: ANSI Colors in Separate Package (2026-03-22)

Color constants, helpers (`Colored`, `Dim`, `Yellow`), and color threshold functions (`ContextColor`, `QuotaColor`) live in `internal/ansi/`. This breaks the render↔module import cycle: modules need colors for icon styling, render needs module types for orchestration. The render package no longer owns color logic; it imports ansi for bar rendering only.

## ADR-010: Per-Turn Tool Stats Reset (2026-03-22)

Tool statistics reset on each user message in the transcript JSONL (`type == "user"`). Running tools are preserved across resets; completed/error tools are cleared. This makes the tools module show only the current turn's activity instead of cumulative session totals. A separate `SessionToolNames` list (never reset) tracks all tools seen across the session, sorted by most recent usage, limited to 6. Tools with zero count in the current turn show as dimmed `×0`.

## ADR-011: RawMessage Deferred Parsing (2026-03-22)

Transcript JSONL entries use `json.RawMessage` for the `message` field. User messages have `content` as a string; assistant messages have `content` as `[]contentBlock`. A single unmarshal populates header fields (type, timestamp, slug), then message content is decoded only for non-user entries. This avoids double-parsing and gracefully handles the polymorphic content field.

## ADR-012: Agent Tool Name "Agent" (2026-03-22)

Claude Code uses both `"Task"` and `"Agent"` as tool names for subagent dispatch in transcript JSONL. The parser handles both via `case "Task", "Agent"`. A shared `managementTools` map excludes these from regular tool tracking and session tool names.

## ADR-013: Context Limit Replaces Dot Warn Tokens (2026-03-22)

## ADR-014: Tail Scan for Large JSONL Files (2026-03-22)

JSONL transcript files grow unbounded (append-only, never truncated within a session). Long sessions with large tool_result entries can produce files of tens to hundreds of MB. Since no data item requires full session history (todos module removed), the parser seeks to the last `max_tail_size` bytes (default `"10MB"`, configurable as human-readable string like `"50MB"`, `"512KB"`). Set to `"0"` for full scan. `SessionStart` field removed as no module depended on it.

## ADR-013: Context Limit Replaces Dot Warn Tokens (2026-03-22)

`dot_warn_tokens` (which only changed the dot to bright yellow) replaced by `context_limit` (default 250k). When set, both the progress bar and dot calculate percentage against this limit instead of the full context window. Exceeding the limit caps at 100%. Set to 0 to use full window size. This reflects that model performance degrades at high token counts -- the bar naturally turns yellow/red as usage approaches the limit, making the old special-case dot color unnecessary. `BRIGHT_YELLOW` and `LAVENDER` removed from ansi package as dead code.
