package module

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/mritd/claude-statusline/internal/debug"
)

type EnvironmentModule struct {
	claudeMd  int
	rules     int
	mcps      int
	hooks     int
	collected bool
}

func NewEnvironmentModule() *EnvironmentModule { return &EnvironmentModule{} }
func (m *EnvironmentModule) Name() string      { return "environment" }

func (m *EnvironmentModule) Collect(ctx *Context) error {
	if m.collected {
		return nil
	}
	m.collected = true

	home, _ := os.UserHomeDir()
	claudeDir := filepath.Join(home, ".claude")

	for _, p := range []string{
		filepath.Join(claudeDir, "CLAUDE.md"),
		filepath.Join(ctx.CWD, "CLAUDE.md"),
		filepath.Join(ctx.CWD, "CLAUDE.local.md"),
		filepath.Join(ctx.CWD, ".claude", "CLAUDE.md"),
	} {
		if _, err := os.Stat(p); err == nil {
			m.claudeMd++
		}
	}

	m.rules += countGlob(filepath.Join(claudeDir, "rules", "*.md"))
	m.rules += countGlob(filepath.Join(ctx.CWD, ".claude", "rules", "*.md"))

	m.mcps, m.hooks = countSettings(filepath.Join(claudeDir, "settings.json"))
	mcps2, hooks2 := countSettings(filepath.Join(ctx.CWD, ".claude", "settings.json"))
	m.mcps += mcps2
	m.hooks += hooks2

	return nil
}

func (m *EnvironmentModule) Vars(ctx *Context) map[string]string {
	return map[string]string{
		"claude_md": fmt.Sprintf("%d", m.claudeMd),
		"rules":     fmt.Sprintf("%d", m.rules),
		"mcps":      fmt.Sprintf("%d", m.mcps),
		"hooks":     fmt.Sprintf("%d", m.hooks),
	}
}

func (m *EnvironmentModule) DefaultFormat() string {
	return "{claude_md} CLAUDE.md | {rules} rules | {mcps} MCPs | {hooks} hooks"
}

func countGlob(pattern string) int {
	matches, err := filepath.Glob(pattern)
	if err != nil {
		debug.Log("environment", "glob error: %v", err)
		return 0
	}
	return len(matches)
}

func countSettings(path string) (mcps, hooks int) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, 0
	}
	var settings struct {
		MCPServers map[string]json.RawMessage `json:"mcpServers"`
		Hooks      map[string]json.RawMessage `json:"hooks"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		return 0, 0
	}
	return len(settings.MCPServers), len(settings.Hooks)
}
