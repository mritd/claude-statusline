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
			SessionToolNames: []string{"Edit", "Read"},
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
			SessionToolNames: []string{"Read", "Bash"},
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
			SessionToolNames: []string{"Read"},
		},
	}
	m := NewToolsModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	if !strings.Contains(vars["completed"], "✓") || !strings.Contains(vars["completed"], "Read ×3") {
		t.Fatalf("unexpected: %q", vars["completed"])
	}
}

func TestToolsVarsDimmedZeroCount(t *testing.T) {
	// Read was used in a previous turn but not this one; should show dimmed ×0
	ctx := &Context{
		Config: config.Default(),
		Transcript: &transcript.Data{
			Tools: []transcript.ToolEntry{
				{Name: "Bash", Status: "completed"},
			},
			SessionToolNames: []string{"Read", "Bash"},
		},
	}
	m := NewToolsModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	// Bash should have colored icon with ×1
	if !strings.Contains(vars["completed"], "Bash ×1") {
		t.Fatalf("expected Bash ×1, got: %q", vars["completed"])
	}
	// Read should show ×0 (dimmed)
	if !strings.Contains(vars["completed"], "Read ×0") {
		t.Fatalf("expected dimmed Read ×0, got: %q", vars["completed"])
	}
}
