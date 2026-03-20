package module

import (
	"fmt"
	"strings"

	"github.com/mritd/claude-statusline/internal/transcript"
)

type TodosModule struct {
	todos []transcript.TodoItem
}

func NewTodosModule() *TodosModule  { return &TodosModule{} }
func (m *TodosModule) Name() string { return "todos" }

func (m *TodosModule) Collect(ctx *Context) error {
	if ctx.Transcript == nil {
		return fmt.Errorf("no transcript")
	}
	m.todos = ctx.Transcript.Todos
	return nil
}

func (m *TodosModule) Vars(ctx *Context) map[string]string {
	if len(m.todos) == 0 {
		return nil
	}

	var current, pending string
	completed := 0
	total := len(m.todos)

	for _, t := range m.todos {
		if t.Status == "completed" {
			completed++
			continue
		}
		c := t.Content
		if len(c) > 50 {
			c = c[:50] + "..."
		}
		if t.Status == "in_progress" && current == "" {
			current = c
		} else if pending == "" {
			pending = c
		}
	}

	// Fall back to first pending/open todo if nothing is in_progress
	if current == "" {
		current = pending
	}

	progress := fmt.Sprintf("%d/%d", completed, total)

	const (
		green  = "\x1b[32m"
		yellow = "\x1b[33m"
		dim    = "\x1b[2m"
		reset  = "\x1b[0m"
	)

	var summary string
	if completed == total {
		doneIcon := ctx.Config.Icon("done")
		summary = fmt.Sprintf("%s%s%s All todos complete %s(%s)%s", green, doneIcon, reset, dim, progress, reset)
	} else if current != "" {
		todoIcon := ctx.Config.Icon("todo")
		summary = fmt.Sprintf("%s%s%s %s %s(%s)%s", yellow, todoIcon, reset, current, dim, progress, reset)
	} else {
		todoIcon := ctx.Config.Icon("todo")
		summary = fmt.Sprintf("%s%s%s %s(%s)%s", yellow, todoIcon, reset, dim, progress, reset)
	}

	return map[string]string{
		"current":  current,
		"progress": progress,
		"summary":  strings.TrimSpace(summary),
	}
}

func (m *TodosModule) DefaultFormat() string { return "{summary}" }
