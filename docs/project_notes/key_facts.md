# Key Facts

## API

- **Usage endpoint**: `https://api.anthropic.com/api/oauth/usage`
- **Required headers**: `anthropic-beta: oauth-2025-04-20`, `User-Agent: claude-code/2.1`
- **Auth**: Bearer token from macOS Keychain or `~/.claude/.credentials.json`
- **Utilization values**: 0-100 percentage (not 0-1 ratio)

## Keychain (macOS)

- **Default service name**: `Claude Code-credentials`
- **Custom config service name**: `Claude Code-credentials-<8 hex chars SHA256(configDir)>`
- **CLI**: `/usr/bin/security find-generic-password -s <service> [-a <account>] -w`
- **Timeout**: 3 seconds

## Paths

- **Config**: `~/.claude/plugins/claude-statusline/config.json`
- **Cache**: `~/.claude/plugins/claude-statusline/.usage-cache.json`
- **Lock**: `~/.claude/plugins/claude-statusline/.usage-cache.lock`

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
- User messages have `content` as string (not array) -- requires `json.RawMessage` deferred decode
- Agent tool names: both `"Task"` and `"Agent"` map to agents module
- Tail scan: seeks to last `max_tail_size` bytes of file; discards first partial line after seek
- `managementTools` set: Task, Agent (excluded from tool stats)
- Tool entries reset on `type == "user"` entries; running tools preserved, completed/error cleared
- `SessionToolCounts` tracks cumulative call counts per tool name (never reset); used for display
- Per-turn status (completed/error) used only for highlight vs dim styling

## ANSI Colors

- Colors live in `internal/ansi/ansi.go` (separate package to avoid render↔module import cycle)
- Icon colors only; module text colors stay in their respective Vars() methods
- `●` dot follows ContextColor (green/yellow/red), same as bar color

## Debug

- **Enable**: `DEBUG=claude-statusline` or `DEBUG=*`
- **Output**: stderr
