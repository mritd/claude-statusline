package render

import (
	"math"
	"strings"

	"github.com/mritd/claude-statusline/internal/ansi"
)

func Bar(pct, width int, filled, empty string, colorFn func(int) string) string {
	pct = max(0, min(100, pct))
	f := int(math.Round(float64(pct) / 100 * float64(width)))
	e := width - f
	color := colorFn(pct)
	return color + strings.Repeat(filled, f) + ansi.RESET + ansi.DIM + strings.Repeat(empty, e) + ansi.RESET
}

// NewContextBar returns a bar function bound to context colors and the given characters.
func NewContextBar(filled, empty string) func(pct, width int) string {
	return func(pct, width int) string {
		return Bar(pct, width, filled, empty, ansi.ContextColor)
	}
}

// NewQuotaBar returns a bar function bound to quota colors and the given characters.
func NewQuotaBar(filled, empty string) func(pct, width int) string {
	return func(pct, width int) string {
		return Bar(pct, width, filled, empty, ansi.QuotaColor)
	}
}
