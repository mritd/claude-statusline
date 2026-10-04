# Key Facts

## API

- **Usage endpoint**: `https://api.anthropic.com/api/oauth/usage`
- **Required headers**: `anthropic-beta: oauth-2025-04-20`, `User-Agent: claude-code/2.1`
- **Auth**: Bearer token from macOS Keychain or `~/.claude/.credentials.json`
- **Only used as fallback**: stdin `rate_limits` is preferred (ADR-015)
- **Utilization values**: 0-100 percentage (not 0-1 ratio)

## Keychain (macOS)

- **Default service name**: `Claude Code-credentials`
- **Custom config service name**: `Claude Code-credentials-<8 hex chars SHA256(configDir)>`
- **CLI**: `/usr/bin/security find-generic-password -s <service> [-a <account>] -w`
- **Timeout**: 3 seconds

## Paths

All paths are under `$CLAUDE_CONFIG_DIR` (default `~/.claude`):

- **Config**: `plugins/claude-statusline/config.json`
- **Cache**: `plugins/claude-statusline/.usage-cache.json`
- **Lock**: `plugins/claude-statusline/.usage-cache.lock`

## Defaults

- **Bar style**: diamond (◆/◇)
- **Modules**: context, usage, git, tools, agents, environment
- **Newline**: tools, agents, environment (each starts on new line)
- **Icons**: `running=≡`, `completed=✓`, `error=✗`, `dirty=*`
- **context_limit**: 250000 (bar/dot treat this as 100%; set to 0 for full window)
- **max_tail_size**: `"10MB"` (human-readable; `"0"` for full scan; supports KB/MB/GB)
- **Session tools limit**: 6 (most recently used, `maxSessionTools` constant)

## Transcript Parsing

- Scanner buffer: 64KB initial, 16MB max (some JSONL lines can reach 3-4MB from large tool_result content)
- JSONL entry `type` field: `user`, `assistant`, `progress`, `custom-title`, `agent-name`, etc.
- Typed user prompts have `content` as string; assistant turns and tool_result carriers use arrays -- `content` is kept as `json.RawMessage` and decoded only when it starts with `[`
- Agent tool names: both `"Task"` and `"Agent"` map to agents module
- Tail scan: seeks to last `max_tail_size` bytes of file; discards first partial line after seek
- `isAgentTool`: Task, Agent (excluded from tool stats)
- Tool results are recorded as `type == "user"` entries with `tool_result` blocks; meta entries (skill bodies, caveats) carry `isMeta: true`
- Tool entries reset on user prompts only (`isUserPrompt`); running tools preserved, completed/error cleared
- `SessionToolCounts` tracks cumulative call counts per tool name (never reset); used for display
- Per-turn status (completed/error) used only for highlight vs dim styling

## Claude Code stdin / Auto-compact

- stdin `context_window.context_window_size` is the full model window, not `autoCompactWindow`; no compaction threshold field exists
- `rate_limits.{five_hour,seven_day}`: `used_percentage` (0-100), `resets_at` (Unix seconds); only for subscribers after the first API response
- Auto-compact threshold (Claude Code 2.1.289): `min(window, autoCompactWindow) - min(maxOutputTokens, 20000) - 13000`; internal and may change

## ANSI Colors

- Colors live in `internal/ansi/ansi.go` (separate package to avoid render↔module import cycle)
- Icon colors only; module text colors stay in their respective Vars() methods
- `●` dot follows ContextColor (green/yellow/red), same as bar color

## Debug

- **Enable**: `DEBUG=claude-statusline` or `DEBUG=*`
- **Output**: stderr
