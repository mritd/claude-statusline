package render

import (
	"fmt"
	"strings"

	"github.com/mritd/claude-statusline/internal/ansi"
	"github.com/mritd/claude-statusline/internal/config"
	"github.com/mritd/claude-statusline/internal/debug"
	"github.com/mritd/claude-statusline/internal/module"
)

func Render(ctx *module.Context, registry *module.Registry, cfg *config.Config) string {
	modules := registry.Enabled(cfg.Modules)
	if len(modules) == 0 {
		return ""
	}

	var lines [][]string
	current := []string{}

	for _, m := range modules {
		if err := m.Collect(ctx); err != nil {
			debug.Log(m.Name(), "collect error: %v", err)
			continue
		}

		vars := m.Vars(ctx)
		if vars == nil {
			continue
		}

		mc := cfg.ModuleConfig(m.Name())
		format := mc.Format
		if format == "" {
			format = m.DefaultFormat()
		}

		output := module.ExpandFormat(format, vars)
		output = strings.TrimSpace(output)
		if output == "" {
			continue
		}

		if cfg.StartsNewline(m.Name()) && len(current) > 0 {
			lines = append(lines, current)
			current = []string{}
		}
		current = append(current, output)
	}

	if len(current) > 0 {
		lines = append(lines, current)
	}

	var result []string
	for _, line := range lines {
		result = append(result, strings.Join(line, cfg.Separator))
	}

	return strings.Join(result, "\n")
}

func Print(output string) {
	for _, line := range strings.Split(output, "\n") {
		fmt.Println(ansi.RESET + line)
	}
}
