package debug

import (
	"fmt"
	"os"
)

var enabled bool

func init() {
	v := os.Getenv("DEBUG")
	enabled = v == "claude-statusline" || v == "*"
}

func Log(module, format string, args ...any) {
	if !enabled {
		return
	}
	msg := fmt.Sprintf(format, args...)
	fmt.Fprintf(os.Stderr, "[claude-statusline:%s] %s\n", module, msg)
}

func Enabled() bool {
	return enabled
}
