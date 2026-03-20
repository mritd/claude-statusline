package module

import (
	"fmt"
	"strings"

	"github.com/mritd/claude-statusline/internal/transcript"
)

type ToolsModule struct {
	tools []transcript.ToolEntry
}

func NewToolsModule() *ToolsModule  { return &ToolsModule{} }
func (m *ToolsModule) Name() string { return "tools" }

func (m *ToolsModule) Collect(ctx *Context) error {
	if ctx.Transcript == nil {
		return fmt.Errorf("no transcript")
	}
	m.tools = ctx.Transcript.Tools
	return nil
}

func (m *ToolsModule) Vars(ctx *Context) map[string]string {
	var running, completed []transcript.ToolEntry
	for i := range m.tools {
		switch m.tools[i].Status {
		case "running":
			running = append(running, m.tools[i])
		case "completed", "error":
			completed = append(completed, m.tools[i])
		}
	}

	var runParts []string
	start := len(running) - 2
	if start < 0 {
		start = 0
	}
	for _, t := range running[start:] {
		s := "◐ " + t.Name
		if t.Target != "" {
			s += ": " + t.Target
		}
		runParts = append(runParts, s)
	}

	groups := make(map[string]int)
	var order []string
	for _, t := range completed {
		if _, ok := groups[t.Name]; !ok {
			order = append(order, t.Name)
		}
		groups[t.Name]++
	}
	var compParts []string
	for i, name := range order {
		if i >= 4 {
			break
		}
		if groups[name] > 1 {
			compParts = append(compParts, fmt.Sprintf("✓ %s ×%d", name, groups[name]))
		} else {
			compParts = append(compParts, "✓ "+name)
		}
	}

	if len(runParts) == 0 && len(compParts) == 0 {
		return nil
	}

	runStr := strings.Join(runParts, " | ")
	compStr := strings.Join(compParts, " | ")
	var summaryParts []string
	if runStr != "" {
		summaryParts = append(summaryParts, runStr)
	}
	if compStr != "" {
		summaryParts = append(summaryParts, compStr)
	}

	return map[string]string{
		"running":   runStr,
		"completed": compStr,
		"summary":   strings.Join(summaryParts, " | "),
	}
}

func (m *ToolsModule) DefaultFormat() string { return "{summary}" }
