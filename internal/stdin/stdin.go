package stdin

import (
	"encoding/json"
	"io"
	"math"
)

const autocompactBufferPercent = 0.165

type Data struct {
	Model          Model         `json:"model"`
	ContextWindow  ContextWindow `json:"context_window"`
	TranscriptPath string        `json:"transcript_path"`
	CWD            string        `json:"cwd"`
}

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type ContextWindow struct {
	Size                int         `json:"context_window_size"`
	CurrentUsage        *TokenUsage `json:"current_usage"`
	UsedPercentage      *float64    `json:"used_percentage"`
	RemainingPercentage *float64    `json:"remaining_percentage"`
}

type TokenUsage struct {
	InputTokens              int `json:"input_tokens"`
	OutputTokens             int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens"`
}

func Parse(r io.Reader) (*Data, error) {
	var data Data
	if err := json.NewDecoder(r).Decode(&data); err != nil {
		return nil, err
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

func (d *Data) ContextPercent() int {
	if d.ContextWindow.UsedPercentage != nil {
		return int(math.Round(*d.ContextWindow.UsedPercentage))
	}
	if d.ContextWindow.Size <= 0 {
		return 0
	}
	pct := float64(d.TotalTokens()) / float64(d.ContextWindow.Size) * 100
	return min(100, int(math.Round(pct)))
}

func (d *Data) BufferedPercent() int {
	if d.ContextWindow.Size <= 0 {
		return 0
	}
	total := float64(d.TotalTokens())
	size := float64(d.ContextWindow.Size)
	rawRatio := total / size

	const low, high = 0.05, 0.50
	scale := (rawRatio - low) / (high - low)
	scale = max(0, min(1, scale))

	buffer := size * autocompactBufferPercent * scale
	pct := (total + buffer) / size * 100
	return min(100, int(math.Round(pct)))
}

func (d *Data) TokenBreakdown() (input, cache int) {
	if d.ContextWindow.CurrentUsage == nil {
		return 0, 0
	}
	u := d.ContextWindow.CurrentUsage
	return u.InputTokens, u.CacheCreationInputTokens + u.CacheReadInputTokens
}
