package module

import (
	"fmt"
	"strings"
	"time"

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
		switch a.Status {
		case "running":
			elapsed := formatElapsed(time.Since(a.StartTime))
			desc := a.Description
			if len(desc) > 30 {
				desc = desc[:30] + "..."
			}
			s := fmt.Sprintf("%s %s", ctx.Config.Icon("running"), a.Type)
			if a.Model != "" {
				s += fmt.Sprintf(" [%s]", a.Model)
			}
			if desc != "" {
				s += ": " + desc
			}
			s += fmt.Sprintf(" (%s)", elapsed)
			runParts = append(runParts, s)
		case "completed":
			elapsed := formatElapsed(a.EndTime.Sub(a.StartTime))
			compParts = append(compParts, fmt.Sprintf("%s %s (%s)", ctx.Config.Icon("completed"), a.Type, elapsed))
		}
	}

	if len(compParts) > 2 {
		compParts = compParts[len(compParts)-2:]
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
