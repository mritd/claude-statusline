# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

```bash
go build -ldflags="-s -w" -o claude-statusline .   # Build binary
go install github.com/mritd/claude-statusline@latest # Install from remote
go test ./...                                        # Run all tests
go test -v ./internal/module/                        # Run single package tests
go test -run TestContextModule ./internal/module/    # Run single test
DEBUG=claude-statusline go build && echo '{}' | ./claude-statusline  # Debug run
```

No Makefile; standard Go toolchain only. Zero external dependencies (stdlib only). All platforms work with `CGO_ENABLED=0`.

## Architecture

**Pipeline:** stdin JSON -> parse -> modules collect data -> render formatted output -> stdout

The binary is invoked by Claude Code's `statusLine` hook. It receives a JSON blob on stdin containing model info, context window stats, and transcript path, then outputs a formatted statusline string.

### Key packages under `internal/`:

| Package | Description |
|---------|-------------|
| stdin | Parses JSON input from Claude Code (model, context window, transcript path) |
| config | Loads `~/.claude/plugins/claude-statusline/config.json`; auto-generates defaults on first run; provides per-module config and bar style resolution |
| module | Pluggable module system. Each module implements `Module` interface (`Name`, `Collect`, `Vars`, `DefaultFormat`). Registry renders modules in config order |
| render | Progress bar rendering via closure injection (`NewContextBar`/`NewQuotaBar`), ANSI color output, format expansion (`{var}` substitution) |
| transcript | Streams JSONL transcript file to extract active tools, subagents, and todos |
| keychain | Platform-specific credential retrieval (macOS: `/usr/bin/security` CLI, other: `~/.claude/.credentials.json`) |
| debug | Conditional stderr logging enabled by `DEBUG=claude-statusline` or `DEBUG=*` |

### Modules

| Module | Default | Data Source | Description |
|--------|---------|-------------|-------------|
| context | enabled | stdin JSON | Context window usage bar |
| usage | enabled | Claude API + cache | 5h/7d API quota bars with caching |
| todos | enabled | transcript JSONL | Todo progress from TaskCreate/Update; auto-completes stale in_progress, filters deleted |
| git | enabled | git CLI | Branch, dirty, ahead/behind status |
| tools | disabled | transcript JSONL | Active tool call tracking |
| agents | disabled | transcript JSONL | Subagent status tracking |
| project | disabled | stdin JSON | Model name and project path |
| environment | disabled | filesystem | CLAUDE.md, rules, MCPs, hooks counts |

### Design Principles

- **Dependency injection over globals**: Bar characters, color functions, and dim functions are injected via closures/function parameters. No mutable package-level state
- **Single cache read**: UsageModule reads the cache file once per Collect cycle and passes the parsed result through all code paths
- **Rune-aware truncation**: All string truncation uses `[]rune` to avoid splitting multi-byte UTF-8 characters
- **Consistent nil returns**: Module `Vars()` returns nil when there is no data to display, skipping format expansion entirely

### Usage Module Caching

The usage module fetches API quotas via HTTP and caches responses:

- **Success TTL**: 300s (5 min), matches Anthropic usage API rate limit window
- **Failure TTL**: 15s for non-429 errors, distinguished by `IsFailure` flag on cache entries
- **429 backoff**: Exponential 60s->300s cap, gated by `RetryAfterUntil` timestamp
- **File locking**: Prevents concurrent fetches from multiple statusline processes
- **Last good data**: On failure, displays previous successful data with `syncing` indicator

## Project Notes

Design decisions and key configuration facts are maintained in `docs/project_notes/` for consistency across sessions:

- **decisions.md** - Architectural Decision Records (ADRs)
- **key_facts.md** - API endpoints, keychain details, cache config
