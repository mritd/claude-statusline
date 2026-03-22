# Architectural Decisions

## ADR-001: Keychain via /usr/bin/security CLI (not cgo)

macOS keychain access uses `exec.CommandContext` with 3s timeout calling `/usr/bin/security find-generic-password -w`. This avoids the authorization popups that cgo's `SecKeychainFindGenericPassword` triggers, and enables `CGO_ENABLED=0` builds. Service name lookup order: resolved config dir, env var, legacy fallback (`Claude Code-credentials`).

## ADR-002: Cache TTL = API Rate Limit Window (300s)

Default cache TTL is 300s (5 min), matching Anthropic's usage API rate limit window. Non-429 failures use a shorter 15s TTL via `IsFailure` flag on cache entries. 429 backoff is exponential 60s->300s cap.

## ADR-003: Zero External Dependencies

The binary uses Go stdlib only. No external modules. This ensures fast builds, <5ms startup, and zero dependency maintenance. Trade-off: some features require more code (manual JSON parsing, no structured logging).

## ADR-004: Bar Chars via Closure Injection

`NewContextBar(filled, empty)` and `NewQuotaBar(filled, empty)` return closures that capture bar characters. No mutable package-level state in the render package. This is parallel-test safe and consistent with how other render functions (dim, color) are injected.

## ADR-005: Todo Batch Detection via Consecutive TaskCreate

Transcript JSONL accumulates all TaskCreate events across a session. When a new plan is created (2+ consecutive TaskCreate calls), old todos are discarded. Detection: track `consecutiveCreates` counter, confirm batch on 2nd consecutive create, record start index. Both non-task tool calls AND TaskUpdate calls reset the counter (TaskUpdate signals the end of the create phase), but don't clear the confirmed index. Single TaskCreate insertions (adding one task to existing plan) are preserved.

## ADR-007: Auto-Complete In-Progress Tasks on New In-Progress

Claude Code often skips explicit `TaskUpdate(status=completed)` events in the transcript JSONL, even though its internal task UI shows tasks as completed. When a new task enters `in_progress`, all previously `in_progress` tasks are auto-completed. This heuristic bridges the gap between Claude Code's in-memory task state and the transcript JSONL record.

## ADR-008: Filter Deleted Todos from Transcript

Tasks marked as `deleted` via TaskUpdate are filtered out after parsing. This handles the pattern where Claude Code creates a brainstorming batch, deletes it, then creates the real implementation batch. Without filtering, deleted tasks inflate the total count and may appear as pending.

## ADR-006: Configurable Icons via `icons` Map

All statusline icons (running, completed, error, todo, done, dirty) are configurable via `icons` map in config.json. Modules read icons through `ctx.Config.Icon(key)` instead of hardcoding. This allows users to fix terminal font alignment issues by swapping characters.

## ADR-009: ANSI Colors in Separate Package (2026-03-22)

Color constants, helpers (`Colored`, `Dim`, `Yellow`), and color threshold functions (`ContextColor`, `QuotaColor`) live in `internal/ansi/`. This breaks the render↔module import cycle: modules need colors for icon styling, render needs module types for orchestration. The render package no longer owns color logic; it imports ansi for bar rendering only.
