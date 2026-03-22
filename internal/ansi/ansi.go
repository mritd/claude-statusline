package ansi

const (
	RESET          = "\x1b[0m"
	RED            = "\x1b[31m"
	GREEN          = "\x1b[32m"
	YELLOW         = "\x1b[33m"
	MAGENTA        = "\x1b[35m"
	CYAN           = "\x1b[36m"
	BRIGHT_BLUE    = "\x1b[94m"
	BRIGHT_MAGENTA = "\x1b[95m"
	DIM            = "\x1b[2m"
	BOLD           = "\x1b[1m"
	LAVENDER       = "\x1b[38;5;146m"
	BRIGHT_YELLOW  = "\x1b[93m"
)

func Colored(color, text string) string {
	return color + text + RESET
}

func Dim(text string) string {
	return DIM + text + RESET
}

func Yellow(text string) string {
	return YELLOW + text + RESET
}

func ContextColor(pct int) string {
	switch {
	case pct >= 85:
		return RED
	case pct >= 70:
		return YELLOW
	default:
		return GREEN
	}
}

func QuotaColor(pct int) string {
	switch {
	case pct >= 90:
		return RED
	case pct >= 75:
		return BRIGHT_MAGENTA
	default:
		return BRIGHT_BLUE
	}
}
