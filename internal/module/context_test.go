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
			Size:         200000,
			CurrentUsage: &stdin.TokenUsage{InputTokens: 90000},
		},
	}
	m := NewContextModule(10, DefaultDotWarnTokens, testBarFn)
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
	m := NewContextModule(10, DefaultDotWarnTokens, testBarFn)
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

func TestDotWarnTokens(t *testing.T) {
	// 200k tokens in 1M window = 20%, green zone but exceeds warn threshold
	data := &stdin.Data{
		ContextWindow: stdin.ContextWindow{
			Size:         1_000_000,
			CurrentUsage: &stdin.TokenUsage{InputTokens: 200_000},
		},
	}
	m := NewContextModule(10, DefaultDotWarnTokens, testBarFn)
	ctx := &Context{Stdin: data}
	_ = m.Collect(ctx)
	vars := m.Vars(ctx)

	if !strings.Contains(vars["dot"], ansi.BRIGHT_YELLOW) {
		t.Fatalf("dot should be bright yellow at 200k tokens, got %q", vars["dot"])
	}

	// 100k tokens — below threshold, should be green
	data.ContextWindow.CurrentUsage.InputTokens = 100_000
	vars = m.Vars(ctx)
	if !strings.Contains(vars["dot"], ansi.GREEN) {
		t.Fatalf("dot should be green below threshold, got %q", vars["dot"])
	}

	// dot_warn_tokens=0 disables the feature
	m2 := NewContextModule(10, 0, testBarFn)
	data.ContextWindow.CurrentUsage.InputTokens = 200_000
	vars = m2.Vars(ctx)
	if !strings.Contains(vars["dot"], ansi.GREEN) {
		t.Fatalf("dot should be green when warn disabled, got %q", vars["dot"])
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
