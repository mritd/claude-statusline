package module

import (
	"fmt"
	"strings"
	"time"

	"github.com/mritd/claude-statusline/internal/ansi"
	"github.com/mritd/claude-statusline/internal/transcript"
)

type AgentsModule struct {
	agents []transcript.AgentEntry
}

func NewAgentsModule() *AgentsModule { return &AgentsModule{} }
func (m *AgentsModule) Name() string { return "agents" }

func (m *AgentsModule) Collect(ctx *Context) error {
	if ctx.Transcript == nil {
		return fmt.Errorf("no transcript")
	}
	m.agents = ctx.Transcript.Agents
	return nil
}

func (m *AgentsModule) Vars(ctx *Context) map[string]string {
	var runParts, compParts []string

	for _, a := range m.agents {
		name := agentDisplayName(a)
		switch a.Status {
		case "running":
			s := ctx.Config.Icon("running") + " " + name
			if a.Model != "" {
				s += fmt.Sprintf(" [%s]", a.Model)
			}
			runParts = append(runParts, ansi.Colored(ansi.CYAN, s+"..."))
		case "completed":
			elapsed := formatElapsed(a.EndTime.Sub(a.StartTime))
			compParts = append(compParts, fmt.Sprintf("%s %s (%s)", ctx.Config.Icon("running"), name, elapsed))
		case "error":
			elapsed := formatElapsed(a.EndTime.Sub(a.StartTime))
			s := fmt.Sprintf("%s %s (%s)", ctx.Config.Icon("running"), name, elapsed)
			compParts = append(compParts, ansi.Colored(ansi.RED, s))
		}
	}

	// When agents are running, hide completed; otherwise show only the latest.
	if len(runParts) > 0 {
		compParts = nil
	} else if len(compParts) > 1 {
		compParts = compParts[len(compParts)-1:]
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

func (m *AgentsModule) DefaultFormat() string { return "{summary}" }

func agentDisplayName(a transcript.AgentEntry) string {
	if a.Description != "" {
		return truncateRunes(a.Description, 50)
	}
	if a.Type != "" {
		return a.Type
	}
	return "Agent"
}

func truncateRunes(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "..."
}

func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Second:
		return "<1s"
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	default:
		m := int(d.Minutes())
		s := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", m, s)
	}
}
