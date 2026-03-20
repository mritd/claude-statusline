package module

import (
	"testing"

	"github.com/mritd/claude-statusline/internal/config"
	"github.com/mritd/claude-statusline/internal/transcript"
)

func TestTodosVarsInProgress(t *testing.T) {
	ctx := &Context{
		Config: config.Default(),
		Transcript: &transcript.Data{
			Todos: []transcript.TodoItem{
				{Content: "Fix auth bug", Status: "in_progress"},
				{Content: "Add tests", Status: "pending"},
				{Content: "Deploy", Status: "completed"},
			},
		},
	}
	m := NewTodosModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	if vars["current"] != "Fix auth bug" {
		t.Fatalf("current = %q", vars["current"])
	}
	if vars["progress"] != "1/3" {
		t.Fatalf("progress = %q", vars["progress"])
	}
}

func TestTodosVarsAllComplete(t *testing.T) {
	ctx := &Context{
		Config: config.Default(),
		Transcript: &transcript.Data{
			Todos: []transcript.TodoItem{
				{Content: "A", Status: "completed"},
				{Content: "B", Status: "completed"},
			},
		},
	}
	m := NewTodosModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	if vars["progress"] != "2/2" {
		t.Fatalf("progress = %q", vars["progress"])
	}
}
