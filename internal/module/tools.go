package module

import (
	"fmt"
	"strings"

	"github.com/mritd/claude-statusline/internal/ansi"
	"github.com/mritd/claude-statusline/internal/transcript"
)

type ToolsModule struct {
	tools             []transcript.ToolEntry
	sessionToolNames  []string
	sessionToolCounts map[string]int
}

func NewToolsModule() *ToolsModule  { return &ToolsModule{} }
func (m *ToolsModule) Name() string { return "tools" }

func (m *ToolsModule) Collect(ctx *Context) error {
	if ctx.Transcript == nil {
		return fmt.Errorf("no transcript")
	}
	m.tools = ctx.Transcript.Tools
	m.sessionToolNames = ctx.Transcript.SessionToolNames
	m.sessionToolCounts = ctx.Transcript.SessionToolCounts
	return nil
}

func (m *ToolsModule) Vars(ctx *Context) map[string]string {
	// Count current-turn tools by status.
	var running []transcript.ToolEntry
	compCounts := make(map[string]int)
	errCounts := make(map[string]int)
	for i := range m.tools {
		switch m.tools[i].Status {
		case "running":
			running = append(running, m.tools[i])
		case "completed":
			compCounts[m.tools[i].Name]++
		case "error":
			errCounts[m.tools[i].Name]++
		}
	}

	// Running tools (last 2, with target details).
	var runParts []string
	start := len(running) - 2
	if start < 0 {
		start = 0
	}
	for _, t := range running[start:] {
		s := ansi.Colored(ansi.CYAN, ctx.Config.Icon("running")+" "+t.Name)
		if t.Target != "" {
			s += ansi.Colored(ansi.CYAN, ": "+t.Target)
		}
		runParts = append(runParts, s)
	}

	// Collect names of currently running tools to exclude from session list.
	runningNames := make(map[string]bool)
	for _, t := range running {
		runningNames[t.Name] = true
	}

	// Build completed/error parts from session tool names.
	// Tools active in current turn get colored icons; inactive tools get dimmed.
	// All counts are session-wide cumulative. Skip running tools (already shown).
	compIcon := ctx.Config.Icon("completed")
	errIcon := ctx.Config.Icon("error")
	var compParts, errParts []string
	for _, name := range m.sessionToolNames {
		if runningNames[name] {
			continue
		}
		sc := m.sessionToolCounts[name]
		ec := errCounts[name]
		cc := compCounts[name]
		if ec > 0 {
			errParts = append(errParts, fmt.Sprintf("%s %s",
				ansi.Colored(ansi.RED, errIcon),
				ansi.Dim(fmt.Sprintf("%s ×%d", name, sc))))
		} else if cc > 0 {
			compParts = append(compParts, fmt.Sprintf("%s %s",
				ansi.Colored(ansi.GREEN, compIcon),
				ansi.Dim(fmt.Sprintf("%s ×%d", name, sc))))
		} else {
			// Tool seen in session but not this turn: dimmed.
			compParts = append(compParts, ansi.Dim(fmt.Sprintf("%s %s ×%d", compIcon, name, sc)))
		}
	}

	if len(runParts) == 0 && len(compParts) == 0 && len(errParts) == 0 {
		return nil
	}

	runStr := strings.Join(runParts, " | ")
	compStr := strings.Join(compParts, " | ")
	errStr := strings.Join(errParts, " | ")
	var summaryParts []string
	if runStr != "" {
		summaryParts = append(summaryParts, runStr)
	}
	if compStr != "" {
		summaryParts = append(summaryParts, compStr)
	}
	if errStr != "" {
		summaryParts = append(summaryParts, errStr)
	}

	return map[string]string{
		"running":   runStr,
		"completed": compStr,
		"errors":    errStr,
		"summary":   strings.Join(summaryParts, " | "),
	}
}

func (m *ToolsModule) DefaultFormat() string { return "{summary}" }
