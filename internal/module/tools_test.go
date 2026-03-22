package module

import (
	"strings"
	"testing"
	"time"

	"github.com/mritd/claude-statusline/internal/config"
	"github.com/mritd/claude-statusline/internal/transcript"
)

func TestToolsVarsRunning(t *testing.T) {
	ctx := &Context{
		Config: config.Default(),
		Transcript: &transcript.Data{
			Tools: []transcript.ToolEntry{
				{Name: "Edit", Target: "auth.ts", Status: "running", StartTime: time.Now()},
				{Name: "Read", Target: "foo.go", Status: "completed"},
			},
			SessionToolNames:  []string{"Edit", "Read"},
			SessionToolCounts: map[string]int{"Edit": 3, "Read": 5},
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

func TestToolsVarsErrors(t *testing.T) {
	ctx := &Context{
		Config: config.Default(),
		Transcript: &transcript.Data{
			Tools: []transcript.ToolEntry{
				{Name: "Read", Status: "completed"},
				{Name: "Bash", Status: "error"},
			},
			SessionToolNames:  []string{"Read", "Bash"},
			SessionToolCounts: map[string]int{"Read": 7, "Bash": 2},
		},
	}
	m := NewToolsModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	if !strings.Contains(vars["completed"], "✓") || !strings.Contains(vars["completed"], "Read") {
		t.Fatalf("unexpected completed: %q", vars["completed"])
	}
	if !strings.Contains(vars["errors"], "✗") || !strings.Contains(vars["errors"], "Bash") {
		t.Fatalf("unexpected errors: %q", vars["errors"])
	}
	if !strings.Contains(vars["summary"], "✗") {
		t.Fatalf("summary should contain error icon: %q", vars["summary"])
	}
}

func TestToolsVarsGrouped(t *testing.T) {
	ctx := &Context{
		Config: config.Default(),
		Transcript: &transcript.Data{
			Tools: []transcript.ToolEntry{
				{Name: "Read", Status: "completed"},
				{Name: "Read", Status: "completed"},
				{Name: "Read", Status: "completed"},
			},
			SessionToolNames:  []string{"Read"},
			SessionToolCounts: map[string]int{"Read": 15},
		},
	}
	m := NewToolsModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	if !strings.Contains(vars["completed"], "✓") || !strings.Contains(vars["completed"], "Read ×15") {
		t.Fatalf("unexpected: %q", vars["completed"])
	}
}

func TestToolsVarsDimmedSessionCount(t *testing.T) {
	// Read was used in a previous turn but not this one; should show dimmed with session count
	ctx := &Context{
		Config: config.Default(),
		Transcript: &transcript.Data{
			Tools: []transcript.ToolEntry{
				{Name: "Bash", Status: "completed"},
			},
			SessionToolNames:  []string{"Read", "Bash"},
			SessionToolCounts: map[string]int{"Read": 12, "Bash": 5},
		},
	}
	m := NewToolsModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	// Bash active this turn: colored icon with session count
	if !strings.Contains(vars["completed"], "Bash ×5") {
		t.Fatalf("expected Bash ×5, got: %q", vars["completed"])
	}
	// Read inactive this turn: dimmed with session count
	if !strings.Contains(vars["completed"], "Read ×12") {
		t.Fatalf("expected dimmed Read ×12, got: %q", vars["completed"])
	}
}
