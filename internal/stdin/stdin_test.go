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

func TestParseRateLimits(t *testing.T) {
	input := `{"rate_limits":{"five_hour":{"used_percentage":12.5,"resets_at":1760000000},"seven_day":{"used_percentage":40,"resets_at":1760500000}}}`
	data, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	rl := data.RateLimits
	if rl == nil || rl.FiveHour == nil || rl.SevenDay == nil {
		t.Fatalf("expected both windows, got %+v", rl)
	}
	if rl.FiveHour.UsedPercentage != 12.5 || rl.FiveHour.ResetsAt != 1760000000 {
		t.Fatalf("unexpected five_hour: %+v", rl.FiveHour)
	}
	if rl.SevenDay.UsedPercentage != 40 || rl.SevenDay.ResetsAt != 1760500000 {
		t.Fatalf("unexpected seven_day: %+v", rl.SevenDay)
	}
}

func TestEmptyStdin(t *testing.T) {
	data, err := Parse(strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	if data.TotalTokens() != 0 || data.RateLimits != nil {
		t.Fatalf("expected empty data, got %+v", data)
	}
}

func TestParseToleratesMistypedField(t *testing.T) {
	input := `{"model":{"display_name":"Opus"},"rate_limits":{"five_hour":{"used_percentage":10,"resets_at":"soon"}},"cwd":"/p"}`
	data, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatalf("mistyped field should not fail parse: %v", err)
	}
	if data.Model.DisplayName != "Opus" || data.CWD != "/p" {
		t.Fatalf("expected other fields decoded, got %+v", data)
	}
}

func TestParseFractionalResetsAt(t *testing.T) {
	input := `{"rate_limits":{"five_hour":{"used_percentage":10,"resets_at":1760000000.5}}}`
	data, err := Parse(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if int64(data.RateLimits.FiveHour.ResetsAt) != 1760000000 {
		t.Fatalf("unexpected resets_at: %v", data.RateLimits.FiveHour.ResetsAt)
	}
}

func TestParseMalformed(t *testing.T) {
	if _, err := Parse(strings.NewReader(`{"model":`)); err == nil {
		t.Fatal("expected error for malformed JSON")
	}
}
