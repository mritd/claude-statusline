package module

import (
	"strings"
	"testing"

	"github.com/mritd/claude-statusline/internal/ansi"
	"github.com/mritd/claude-statusline/internal/stdin"
)

func testBarFn(pct, width int) string {
	filled := pct * width / 100
	empty := width - filled
	result := ""
	for range filled {
		result += "█"
	}
	for range empty {
		result += "░"
	}
	return result
}

func TestContextVars(t *testing.T) {
	data := &stdin.Data{
		ContextWindow: stdin.ContextWindow{
			Size:         200_000,
			CurrentUsage: &stdin.TokenUsage{InputTokens: 90_000},
		},
	}
	// context_limit=0 means use full window size (200k)
	m := NewContextModule(10, 0, testBarFn)
	ctx := &Context{Stdin: data}
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)

	if vars["percent"] != "45%" {
		t.Fatalf("percent = %q", vars["percent"])
	}
	if vars["tokens"] != "90k" {
		t.Fatalf("tokens = %q", vars["tokens"])
	}
	if vars["remaining"] != "110k" {
		t.Fatalf("remaining = %q", vars["remaining"])
	}
	if !strings.Contains(vars["bar"], "█") {
		t.Fatal("bar should contain filled chars")
	}
}

func TestContextVarsCritical(t *testing.T) {
	data := &stdin.Data{
		ContextWindow: stdin.ContextWindow{
			Size: 200000,
			CurrentUsage: &stdin.TokenUsage{
				InputTokens:              150000,
				CacheCreationInputTokens: 20000,
				CacheReadInputTokens:     10000,
			},
		},
	}
	// context_limit=0 means use full window (200k)
	m := NewContextModule(10, 0, testBarFn)
	ctx := &Context{Stdin: data}
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)

	if vars["percent"] != "90%" {
		t.Fatalf("percent = %q", vars["percent"])
	}
	if vars["breakdown"] == "" {
		t.Fatal("breakdown should be populated at >=85%")
	}
}

func TestContextLimit(t *testing.T) {
	data := &stdin.Data{
		ContextWindow: stdin.ContextWindow{
			Size:         1_000_000,
			CurrentUsage: &stdin.TokenUsage{InputTokens: 150_000},
		},
	}

	// context_limit=300k, usage=150k -> 50%
	m := NewContextModule(10, 300_000, testBarFn)
	ctx := &Context{Stdin: data}
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)

	if vars["percent"] != "50%" {
		t.Fatalf("expected 50%%, got %s", vars["percent"])
	}
	if vars["remaining"] != "150k" {
		t.Fatalf("expected remaining 150k, got %s", vars["remaining"])
	}
	// 50% -> green dot
	if !strings.Contains(vars["dot"], ansi.GREEN) {
		t.Fatalf("dot should be green at 50%%, got %q", vars["dot"])
	}

	// usage=250k -> 83%, yellow zone
	data.ContextWindow.CurrentUsage.InputTokens = 250_000
	vars = m.Vars(ctx)
	if vars["percent"] != "83%" {
		t.Fatalf("expected 83%%, got %s", vars["percent"])
	}
	if !strings.Contains(vars["dot"], ansi.YELLOW) {
		t.Fatalf("dot should be yellow at 83%%, got %q", vars["dot"])
	}

	// usage=400k -> exceeds limit, show actual tokens instead of 100%
	data.ContextWindow.CurrentUsage.InputTokens = 400_000
	vars = m.Vars(ctx)
	if vars["percent"] != "400k" {
		t.Fatalf("expected 400k (over limit), got %s", vars["percent"])
	}
	if !strings.Contains(vars["dot"], ansi.RED) {
		t.Fatalf("dot should be red at 100%%, got %q", vars["dot"])
	}
	if vars["remaining"] != "0" {
		t.Fatalf("remaining should be 0 when over limit, got %s", vars["remaining"])
	}

	// context_limit=0 disables, falls back to full window
	m2 := NewContextModule(10, 0, testBarFn)
	data.ContextWindow.CurrentUsage.InputTokens = 150_000
	vars = m2.Vars(ctx)
	if vars["percent"] != "15%" {
		t.Fatalf("expected 15%% with no limit, got %s", vars["percent"])
	}
}

func TestFormatTokens(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{500, "500"},
		{1000, "1k"},
		{45000, "45k"},
		{1500000, "1.5M"},
		{2000000, "2.0M"},
	}
	for _, tt := range tests {
		got := formatTokens(tt.n)
		if got != tt.want {
			t.Errorf("formatTokens(%d) = %q, want %q", tt.n, got, tt.want)
		}
	}
}
