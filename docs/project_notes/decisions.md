# Architectural Decisions

## ADR-001: Keychain via /usr/bin/security CLI (not cgo)

macOS keychain access uses `exec.CommandContext` with 3s timeout calling `/usr/bin/security find-generic-password -w`. This avoids the authorization popups that cgo's `SecKeychainFindGenericPassword` triggers, and enables `CGO_ENABLED=0` builds. Service name lookup order: resolved config dir, env var, legacy fallback (`Claude Code-credentials`).

## ADR-002: Cache TTL = API Rate Limit Window (300s)

Default cache TTL is 300s (5 min), matching Anthropic's usage API rate limit window. Non-429 failures use a shorter 15s TTL via `IsFailure` flag on cache entries. 429 backoff is exponential 60s->300s cap.

## ADR-003: Zero External Dependencies

The binary uses Go stdlib only. No external modules. This ensures fast builds, <5ms startup, and zero dependency maintenance. Trade-off: some features require more code (manual JSON parsing, no structured logging).

## ADR-004: Bar Chars via Closure Injection

`NewContextBar(filled, empty)` and `NewQuotaBar(filled, empty)` return closures that capture bar characters. No mutable package-level state in the render package. This is parallel-test safe and consistent with how other render functions (dim, color) are injected.
