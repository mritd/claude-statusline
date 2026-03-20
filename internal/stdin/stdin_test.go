package stdin

import (
	"strings"
	"testing"
)

func TestParseBasic(t *testing.T) {
	input := `{"model":{"display_name":"Claude Opus 4.6"},"context_window":{"context_window_size":200000,"current_usage":{"input_tokens":90000,"cache_creation_input_tokens":5000,"cache_read_input_tokens":5000}},"transcript_path":"/tmp/test.jsonl","cwd":"/home/user/project"}`
	data, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if data.Model.DisplayName != "Claude Opus 4.6" {
		t.Fatalf("unexpected model: %s", data.Model.DisplayName)
	}
	if data.CWD != "/home/user/project" {
		t.Fatalf("unexpected cwd: %s", data.CWD)
	}
}

func TestTotalTokens(t *testing.T) {
	data := &Data{
		ContextWindow: ContextWindow{
			Size: 200000,
			CurrentUsage: &TokenUsage{
				InputTokens:              90000,
				CacheCreationInputTokens: 5000,
				CacheReadInputTokens:     5000,
			},
		},
	}
	total := data.TotalTokens()
	if total != 100000 {
		t.Fatalf("expected 100000, got %d", total)
	}
}

func TestContextPercentNative(t *testing.T) {
	pct := 45.0
	data := &Data{
		ContextWindow: ContextWindow{
			Size:           200000,
			UsedPercentage: &pct,
		},
	}
	got := data.ContextPercent()
	if got != 45 {
		t.Fatalf("expected native 45, got %d", got)
	}
}

func TestContextPercentFallback(t *testing.T) {
	data := &Data{
		ContextWindow: ContextWindow{
			Size:         200000,
			CurrentUsage: &TokenUsage{InputTokens: 90000},
		},
	}
	got := data.ContextPercent()
	if got != 45 {
		t.Fatalf("expected fallback 45, got %d", got)
	}
}

func TestBufferedPercent(t *testing.T) {
	data := &Data{
		ContextWindow: ContextWindow{
			Size:         200000,
			CurrentUsage: &TokenUsage{InputTokens: 100000}, // 50% raw
		},
	}
	bp := data.BufferedPercent()
	// At 50% raw, scale = (0.5-0.05)/(0.50-0.05) = 1.0
	// buffer = 200000 * 0.165 * 1.0 = 33000
	// buffered = (100000+33000)/200000 * 100 = 66.5 -> 67
	if bp != 67 {
		t.Fatalf("expected buffered 67, got %d", bp)
	}
}

func TestContextPercentCap100(t *testing.T) {
	data := &Data{
		ContextWindow: ContextWindow{
			Size:         100000,
			CurrentUsage: &TokenUsage{InputTokens: 150000},
		},
	}
	if data.ContextPercent() != 100 {
		t.Fatalf("expected capped 100, got %d", data.ContextPercent())
	}
}

func TestEmptyStdin(t *testing.T) {
	data, err := Parse(strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if data.ContextPercent() != 0 {
		t.Fatalf("expected 0 for empty, got %d", data.ContextPercent())
	}
}
