package module

import (
	"testing"
	"time"

	"github.com/mritd/claude-statusline/internal/transcript"
)

func TestToolsVarsRunning(t *testing.T) {
	ctx := &Context{
		Transcript: &transcript.Data{
			Tools: []transcript.ToolEntry{
				{Name: "Edit", Target: "auth.ts", Status: "running", StartTime: time.Now()},
				{Name: "Read", Target: "foo.go", Status: "completed"},
			},
		},
	}
	m := NewToolsModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	if vars["running"] == "" {
		t.Fatal("running should not be empty")
	}
	if vars["completed"] == "" {
		t.Fatal("completed should not be empty")
	}
}

func TestToolsVarsGrouped(t *testing.T) {
	ctx := &Context{
		Transcript: &transcript.Data{
			Tools: []transcript.ToolEntry{
				{Name: "Read", Status: "completed"},
				{Name: "Read", Status: "completed"},
				{Name: "Read", Status: "completed"},
			},
		},
	}
	m := NewToolsModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	if vars["completed"] != "✓ Read ×3" {
		t.Fatalf("unexpected: %q", vars["completed"])
	}
}
