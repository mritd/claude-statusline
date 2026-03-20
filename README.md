# claude-statusline

A fast, secure, modular statusline for [Claude Code](https://docs.anthropic.com/en/docs/claude-code).

Single Go binary. Zero external dependencies. Starts in <5ms.

```
Context ██░░░░░░░░ 21% | Usage ████░░░░░░ 37% (1h30m) | ██░░░░░░░░ 15% (4d16h) | ✓ All todos complete (17/17) | main*
```

## Install

### One-liner

```bash
go install github.com/mritd/claude-statusline@latest
```

Then add to `~/.claude/settings.json`:

```json
{
  "statusLine": {
    "type": "command",
    "command": "claude-statusline"
  }
}
```

> Make sure `$(go env GOPATH)/bin` is in your `PATH`.

### From source

```bash
git clone https://github.com/mritd/claude-statusline.git
cd claude-statusline
go build -ldflags="-s -w" -o claude-statusline .
mv claude-statusline /usr/local/bin/  # or anywhere in PATH
```

## Configuration

A default config is auto-generated on first run at `~/.claude/plugins/claude-statusline/config.json`:

```json
{
  "modules": ["context", "usage", "todos", "git"],
  "separator": " | ",
  "bar_style": "block",
  "bar_styles": {
    "block":      ["█", "░"],
    "square":     ["■", "□"],
    "small":      ["▪", "▫"],
    "bar":        ["▰", "▱"],
    "dot":        ["●", "○"],
    "diamond":    ["◆", "◇"],
    "line":       ["━", "─"],
    "braille":    ["⣿", "⣀"],
    "shade":      ["▓", "░"],
    "half":       ["▄", "▁"],
    "half-block": ["▌", "░"]
  }
}
```

### Bar styles

Change `bar_style` to switch presets, or add custom entries to `bar_styles`:

```json
{
  "bar_style": "bar",
  "bar_styles": {
    "bar":    ["▰", "▱"],
    "stars":  ["★", "☆"]
  }
}
```

### Modules

| Module | Default | Description |
|--------|---------|-------------|
| `context` | enabled | Context window usage bar |
| `usage` | enabled | 5h/7d API usage with reset times |
| `todos` | enabled | Task progress |
| `git` | enabled | Branch name + dirty indicator |
| `tools` | disabled | Active tool calls |
| `agents` | disabled | Subagent status |
| `project` | disabled | Model name + project path |
| `environment` | disabled | CLAUDE.md/rules/MCP counts |

### Custom formats

Each module supports a `format` override using `{var}` template syntax:

```json
{
  "context": {
    "format": "Ctx {bar} {percent} {breakdown}"
  },
  "usage": {
    "format": "5h:{5h_pct} 7d:{7d_pct}"
  },
  "git": {
    "format": "{branch}{dirty}{ahead}{behind}"
  }
}
```

### Module variables

**context**: `{bar}` `{percent}` `{tokens}` `{remaining}` `{breakdown}`

**usage**: `{5h_bar}` `{5h_pct}` `{5h_reset}` `{7d_bar}` `{7d_pct}` `{7d_reset}` `{syncing}` `{api_error}`

**git**: `{branch}` `{dirty}` `{ahead}` `{behind}`

**todos**: `{current}` `{progress}` `{summary}`

**tools**: `{running}` `{completed}` `{summary}`

**agents**: `{running}` `{completed}` `{summary}`

**project**: `{model}` `{plan}` `{path}`

**environment**: `{claude_md}` `{rules}` `{mcps}` `{hooks}`

### Multi-line output

Use `newline` to break modules onto separate lines:

```json
{
  "modules": ["context", "usage", "git", "todos"],
  "newline": ["todos"]
}
```

Output:
```
Context ██░░░░░░░░ 21% | Usage ████░░░░░░ 37% (1h30m) | ██░░░░░░░░ 15% (4d16h)
✓ All todos complete (17/17) | main*
```

### Cache tuning

Usage module caches API responses. Configurable per-module:

```json
{
  "usage": {
    "cache_ttl_seconds": 300,
    "failure_cache_ttl_seconds": 15
  }
}
```

## Color thresholds

**Context bar**: green (<70%) / yellow (70-85%) / red (>=85%)

**Usage bars**: blue (<75%) / magenta (75-90%) / red (>=90%)

**Git**: bold green branch, yellow dirty `*`

**Todos**: green `✓`, yellow `▸`

**API errors**: yellow `API 429* (34m)`

## Debug

```bash
DEBUG=claude-statusline echo '{}' | claude-statusline
```

Debug output goes to stderr.

## Platform

- **macOS**: Full support. OAuth token read from Keychain via `/usr/bin/security` CLI.
- **Linux/Windows**: Builds with `CGO_ENABLED=0`. Token read from `~/.claude/.credentials.json`.

## Credits

This project is inspired by [claude-hud](https://github.com/jarrodwatts/claude-hud) (TypeScript), reimplemented in Go for faster startup and zero dependencies.

## License

MIT
