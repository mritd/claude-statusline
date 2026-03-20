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

## Debug

- **Enable**: `DEBUG=claude-statusline` or `DEBUG=*`
- **Output**: stderr
