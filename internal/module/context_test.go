package module

import (
	"strings"
	"testing"

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
			Size:         200000,
			CurrentUsage: &stdin.TokenUsage{InputTokens: 90000},
		},
	}
	m := NewContextModule(10, testBarFn)
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
	m := NewContextModule(10, testBarFn)
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
