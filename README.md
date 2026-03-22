# claude-statusline

A fast, secure, modular statusline for [Claude Code](https://docs.anthropic.com/en/docs/claude-code).

Single Go binary. Zero external dependencies. Starts in <5ms.

<img width="1290" height="173" alt="image" src="https://github.com/user-attachments/assets/6618cbb4-0769-4d3d-a26c-bb59feba9149" />

```
● Context ◆◇◇◇◇◇◇◇◇◇ 14% | Usage ◆◇◇◇◇◇◇◇◇◇ 4% (2h 22m) | ◆◆◆◇◇◇◇◇◇◇ 31% (2d 14h) | main*
≡ Read: config.go | ✓ Bash ×5 | ✓ Edit ×3 | ✓ Read ×2 | ✓ Write ×0 | ✗ Bash ×1
≡ Review all uncommitted changes
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
  "modules": ["context", "usage", "git", "tools", "agents", "environment"],
  "separator": " | "
}
```

All other settings (bar styles, icons, etc.) have built-in defaults and only need to be added when you want to override them.

### Bar styles

Default style is `diamond` (◆/◇). Change `bar_style` to switch presets, or add custom entries to `bar_styles`:

```json
{
  "bar_style": "block",
  "bar_styles": {
    "stars": ["★", "☆"]
  }
}
```

Built-in styles: `block` (█░), `square` (■□), `small` (▪▫), `bar` (▰▱), `dot` (●○), `diamond` (◆◇), `line` (━─), `braille` (⣿⣀), `shade` (▓░), `half` (▄▁), `half-block` (▌░)

### Icons

Override any icon used in the statusline:

```json
{
  "icons": {
    "running": "▸",
    "dirty": "●"
  }
}
```

| Key | Default | Used in |
|-----|---------|---------|
| `running` | `≡` | tools/agents: active tool or subagent |
| `completed` | `✓` | tools: finished items |
| `error` | `✗` | tools: failed tool calls |
| `dirty` | `*` | git: uncommitted changes |

### Modules

| Module | Default | Description |
|--------|---------|-------------|
| `context` | enabled | Context window usage bar with status dot |
| `usage` | enabled | 5h/7d API usage with reset times |
| `git` | enabled | Branch name + dirty indicator |
| `tools` | enabled | Per-turn tool calls with session history |
| `agents` | enabled | Subagent status (running/completed) |
| `project` | disabled | Model name + project path |
| `environment` | enabled | CLAUDE.md/rules/MCP counts |

### Custom formats

Each module supports a `format` override using `{var}` template syntax:

```json
{
  "context": {
    "format": "{dot} Ctx {bar} {percent} {breakdown}"
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

**context**: `{dot}` `{bar}` `{percent}` `{tokens}` `{remaining}` `{breakdown}`

**usage**: `{plan}` `{5h_bar}` `{5h_pct}` `{5h_reset}` `{7d_bar}` `{7d_pct}` `{7d_reset}` `{syncing}` `{api_error}`

**git**: `{branch}` `{dirty}` `{ahead}` `{behind}`

**tools**: `{running}` `{completed}` `{errors}` `{summary}`

**agents**: `{running}` `{completed}` `{summary}`

**project**: `{model}` `{plan}` `{path}`

**environment**: `{claude_md}` `{rules}` `{mcps}` `{hooks}`

### Multi-line output

Use `newline` to break modules onto separate lines. Default: tools starts on a new line.

```json
{
  "modules": ["context", "usage", "git", "tools", "agents", "environment"],
  "newline": ["tools", "agents", "environment"]
}
```

Output:
```
● Context ◆◇◇◇◇◇◇◇◇◇ 14% | Usage ◇◇◇◇◇◇◇◇◇◇ 4% (2h 22m) | ◆◆◆◇◇◇◇◇◇◇ 31% (2d 14h) | main*
✓ Bash ×5 | ✓ Edit ×3 | ✓ Read ×0 | ✓ Grep ×0 | ✓ Write ×0 | ✓ Glob ×0
≡ Review all uncommitted changes
2 CLAUDE.md | 7 rules | 0 MCPs | 0 hooks
```

### Transcript tail scan

For long sessions, the JSONL transcript can grow large. The parser only reads the tail of the file by default:

```json
{
  "max_tail_size": "10MB"
}
```

Accepts human-readable sizes: `"512KB"`, `"10MB"`, `"1GB"`. Set to `"0"` for full scan.

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

## Tools module

The tools module tracks tool calls per turn (resets on each user message). It maintains a session-wide list of the 6 most recently used tools, sorted by last usage time.

- **Running**: shows tool name + target (file path or command), cyan
- **Completed**: shows count (`✓ Read ×3`), green icon + dim text
- **Error**: shows count (`✗ Bash ×1`), red icon + dim text
- **Inactive**: tools used in previous turns but not the current one show as dimmed `×0`

## Agents module

The agents module tracks subagent (Agent tool) activity:

- **Running**: full line in cyan, no elapsed time (`≡ Review all uncommitted changes`)
- **Completed**: no color, shows elapsed time (`≡ Review all uncommitted changes (1m 58s)`)
- **Error**: full line in red, shows elapsed time
- Only the latest completed agent is shown; running agents hide completed ones
- Display name: Description (50 chars) → Type → "Agent" fallback

## Colors

### Context limit

By default, the context bar and dot treat 250k tokens as 100% instead of the full context window (e.g., 1M). This reflects that model performance degrades at high token counts. The dot color follows the bar color.

| Percentage | Color |
|------------|-------|
| < 70% | Green |
| 70-85% | Yellow |
| >= 85% | Red |

Configure or disable:

```json
{
  "context": {
    "context_limit": 250000
  }
}
```

Set to `0` to use the full context window size.

### Bar colors

**Context bar**: green (<70%) → yellow (70-85%) → red (>=85%)

**Usage bars**: blue (<75%) → magenta (75-90%) → red (>=90%)

### Icon colors

| Element | Color |
|---------|-------|
| `≡` running tool | Cyan (icon + name + target) |
| `✓` completed tool | Green (icon only), dim text |
| `✗` error tool | Red (icon only), dim text |
| `≡` running agent | Cyan (entire line) |
| `≡` error agent | Red (entire line) |
| Git branch | Bold green |
| Git dirty `*` | Yellow |
| API errors `API 429*` | Yellow |
| Syncing `⟳ syncing...` | Dim |

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
