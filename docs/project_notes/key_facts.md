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
- **Modules**: context, usage, git, tools
- **Newline**: tools (starts on new line)
- **Icons**: `running=≡`, `completed=✓`, `error=✗`, `todo=▸`, `done=✓`, `dirty=*`
- **dot_warn_tokens**: 200000 (set to 0 to disable)

## ANSI Colors

- Colors live in `internal/ansi/ansi.go` (separate package to avoid render↔module import cycle)
- Icon colors only; module text colors stay in their respective Vars() methods
- `●` dot follows ContextColor (green/yellow/red) with bright yellow override at dot_warn_tokens threshold

## Debug

- **Enable**: `DEBUG=claude-statusline` or `DEBUG=*`
- **Output**: stderr
