package module

import (
	"testing"
	"time"

	"github.com/mritd/claude-statusline/internal/transcript"
)

func TestAgentsVars(t *testing.T) {
	ctx := &Context{
		Transcript: &transcript.Data{
			Agents: []transcript.AgentEntry{
				{Type: "explore", Model: "haiku", Description: "Finding auth code", Status: "running", StartTime: time.Now().Add(-65 * time.Second)},
			},
		},
	}
	m := NewAgentsModule()
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)
	if vars["running"] == "" {
		t.Fatal("running should not be empty")
	}
}

func TestFormatElapsed(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{500 * time.Millisecond, "<1s"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m 30s"},
	}
	for _, tt := range tests {
		got := formatElapsed(tt.d)
		if got != tt.want {
			t.Errorf("formatElapsed(%v) = %q, want %q", tt.d, got, tt.want)
		}
	}
}
