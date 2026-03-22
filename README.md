# claude-statusline

A fast, secure, modular statusline for [Claude Code](https://docs.anthropic.com/en/docs/claude-code).

Single Go binary. Zero external dependencies. Starts in <5ms.

<img width="1088" height="147" alt="image" src="https://github.com/user-attachments/assets/f5c9d377-881d-4e0f-b097-480142331952" />


```
● Context ◆◇◇◇◇◇◇◇◇◇ 14% | Usage ◆◇◇◇◇◇◇◇◇◇ 4% (2h 22m) | ◆◆◆◇◇◇◇◇◇◇ 31% (2d 14h) | main*
✓ Agent ×21 | ✓ Write ×25 | ✓ Skill ×11 | ✓ ToolSearch ×10
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
  "modules": ["context", "usage", "git", "tools"],
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
| `completed` | `✓` | tools/agents: finished items |
| `error` | `✗` | tools: failed tool calls |
| `todo` | `▸` | todos: in-progress/pending task |
| `done` | `✓` | todos: all tasks complete |
| `dirty` | `*` | git: uncommitted changes |

### Modules

| Module | Default | Description |
|--------|---------|-------------|
| `context` | enabled | Context window usage bar with status dot |
| `usage` | enabled | 5h/7d API usage with reset times |
| `git` | enabled | Branch name + dirty indicator |
| `tools` | enabled | Active/completed tool calls |
| `todos` | disabled | Task progress |
| `agents` | disabled | Subagent status |
| `project` | disabled | Model name + project path |
| `environment` | disabled | CLAUDE.md/rules/MCP counts |

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

**todos**: `{current}` `{progress}` `{summary}`

**tools**: `{running}` `{completed}` `{summary}`

**agents**: `{running}` `{completed}` `{summary}`

**project**: `{model}` `{plan}` `{path}`

**environment**: `{claude_md}` `{rules}` `{mcps}` `{hooks}`

### Multi-line output

Use `newline` to break modules onto separate lines. Default: tools starts on a new line.

```json
{
  "modules": ["context", "usage", "git", "tools"],
  "newline": ["tools"]
}
```

Output:
```
● Context ◆◇◇◇◇◇◇◇◇◇ 14% | Usage ◇◇◇◇◇◇◇◇◇◇ 4% (2h 22m) | ◆◆◆◇◇◇◇◇◇◇ 31% (2d 14h) | main*
✓ Agent ×21 | ✓ Write ×25 | ✓ Skill ×11
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

## Colors

### Status dot (●)

The dot before `Context` changes color based on token usage:

| Condition | Color |
|-----------|-------|
| Normal | Green (matches context bar) |
| Token usage >= 200k | Bright yellow (performance warning) |
| Context >= 70% | Yellow (follows bar color) |
| Context >= 85% | Red (follows bar color) |

The 200k threshold warns that model performance may degrade at high token counts, even if the percentage is low (e.g., 20% of 1M context).

Configure or disable:

```json
{
  "context": {
    "dot_warn_tokens": 200000
  }
}
```

Set to `0` to disable the early warning (dot will only follow bar color).

### Bar colors

**Context bar**: green (<70%) → yellow (70-85%) → red (>=85%)

**Usage bars**: blue (<75%) → magenta (75-90%) → red (>=90%)

### Icon colors

| Element | Color |
|---------|-------|
| `≡` (running tool/agent) | Cyan |
| `✓` (completed tool/agent) | Green |
| `▸` (in-progress todo) | Yellow |
| `✓` (all todos done) | Green |
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
