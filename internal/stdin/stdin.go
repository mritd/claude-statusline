package stdin

import (
	"encoding/json"
	"errors"
	"io"

	"github.com/mritd/claude-statusline/internal/debug"
)

type Data struct {
	Model          Model         `json:"model"`
	ContextWindow  ContextWindow `json:"context_window"`
	RateLimits     *RateLimits   `json:"rate_limits"`
	TranscriptPath string        `json:"transcript_path"`
	CWD            string        `json:"cwd"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type ContextWindow struct {
	Size         int         `json:"context_window_size"`
	CurrentUsage *TokenUsage `json:"current_usage"`
}

type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

// RateLimits mirrors the subscription quota windows Claude Code reports.
// Present only for subscribers after the first API response; a window is
// omitted once its resets_at has passed.
type RateLimits struct {
	FiveHour *RateWindow `json:"five_hour"`
	SevenDay *RateWindow `json:"seven_day"`
}

type RateWindow struct {
	UsedPercentage float64 `json:"used_percentage"`
	ResetsAt       float64 `json:"resets_at"` // Unix epoch seconds
}

// Parse decodes the statusline JSON. A field with an unexpected type is left
// at its zero value so one schema drift cannot blank the whole statusline;
// malformed JSON is still an error.
func Parse(r io.Reader) (*Data, error) {
	var data Data
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		var typeErr *json.UnmarshalTypeError
		if !errors.As(err, &typeErr) {
			return nil, err
		}
		debug.Log("stdin", "ignoring mistyped field: %v", err)
	}
	return &data, nil
}

func (d *Data) TotalTokens() int {
	if d.ContextWindow.CurrentUsage == nil {
		return 0
	}
	u := d.ContextWindow.CurrentUsage
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
}

func (d *Data) TokenBreakdown() (input, cache int) {
	if d.ContextWindow.CurrentUsage == nil {
		return 0, 0
	}
	u := d.ContextWindow.CurrentUsage
	return u.InputTokens, u.CacheCreationInputTokens + u.CacheReadInputTokens
}
