package render

import (
	"math"
	"strings"
)

func Bar(pct, width int, filled, empty string, colorFn func(int) string) string {
	pct = max(0, min(100, pct))
	f := int(math.Round(float64(pct) / 100 * float64(width)))
	e := width - f
	color := colorFn(pct)
	return color + strings.Repeat(filled, f) + RESET + DIM + strings.Repeat(empty, e) + RESET
}

// NewContextBar returns a bar function bound to context colors and the given characters.
func NewContextBar(filled, empty string) func(pct, width int) string {
	return func(pct, width int) string {
		return Bar(pct, width, filled, empty, ContextColor)
	}
}

// NewQuotaBar returns a bar function bound to quota colors and the given characters.
func NewQuotaBar(filled, empty string) func(pct, width int) string {
	return func(pct, width int) string {
		return Bar(pct, width, filled, empty, QuotaColor)
	}
}
