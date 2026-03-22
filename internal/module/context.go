package module

import (
	"fmt"

	"github.com/mritd/claude-statusline/internal/ansi"
)

const defaultBarWidth = 10

// BarFunc renders a progress bar for a given percentage and width.
type BarFunc func(pct, width int) string

const DefaultDotWarnTokens = 200_000

type ContextModule struct {
	barWidth      int
	barFn         BarFunc
	dotWarnTokens int // token threshold for dot early warning; 0 disables
}

func NewContextModule(barWidth, dotWarnTokens int, barFn BarFunc) *ContextModule {
	if barWidth <= 0 {
		barWidth = defaultBarWidth
	}
	if dotWarnTokens < 0 {
		dotWarnTokens = 0
	}
	return &ContextModule{barWidth: barWidth, dotWarnTokens: dotWarnTokens, barFn: barFn}
}

func (m *ContextModule) Name() string { return "context" }

func (m *ContextModule) Collect(ctx *Context) error {
	return nil
}

func (m *ContextModule) Vars(ctx *Context) map[string]string {
	if ctx.Stdin == nil {
		return nil
	}

	barWidth := m.barWidth
	if barWidth <= 0 {
		barWidth = defaultBarWidth
	}

	pct := ctx.Stdin.ContextPercent()
	total := ctx.Stdin.TotalTokens()
	size := ctx.Stdin.ContextWindow.Size
	remaining := size - total
	if remaining < 0 {
		remaining = 0
	}

	barStr := ""
	if m.barFn != nil {
		barStr = m.barFn(pct, barWidth)
	}

	dotColor := ansi.ContextColor(pct)
	if m.dotWarnTokens > 0 && total >= m.dotWarnTokens && dotColor == ansi.GREEN {
		dotColor = ansi.BRIGHT_YELLOW
	}

	vars := map[string]string{
		"dot":       ansi.Colored(dotColor, "●"),
		"bar":       barStr,
		"percent":   fmt.Sprintf("%d%%", pct),
		"tokens":    formatTokens(total),
		"remaining": formatTokens(remaining),
	}

	if pct >= 85 {
		input, cache := ctx.Stdin.TokenBreakdown()
		vars["breakdown"] = fmt.Sprintf("(in:%s cache:%s)", formatTokens(input), formatTokens(cache))
	} else {
		vars["breakdown"] = ""
	}

	return vars
}

func (m *ContextModule) DefaultFormat() string {
	return "{dot} Context {bar} {percent}"
}

func formatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1000:
		return fmt.Sprintf("%dk", n/1000)
	default:
		return fmt.Sprintf("%d", n)
	}
}
