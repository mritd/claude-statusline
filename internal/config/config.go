package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/mritd/claude-statusline/internal/debug"
)

type Config struct {
	Modules     []string                   `json:"modules"`
	Separator   string                     `json:"separator"`
	Newline     []string                   `json:"newline"`
	BarStyle    string                     `json:"bar_style"`
	BarStyles   map[string][2]string       `json:"bar_styles"`
	Icons       map[string]string          `json:"icons"`
	MaxTailSize string                     `json:"max_tail_size"`
	Raw         map[string]json.RawMessage `json:"-"`
}

type ModuleConf struct {
	Format                 string `json:"format"`
	BarWidth               int    `json:"bar_width"`
	CacheTTLSeconds        int    `json:"cache_ttl_seconds"`
	FailureCacheTTLSeconds int    `json:"failure_cache_ttl_seconds"`
	ContextLimit           *int   `json:"context_limit,omitempty"`
}

var defaultBarStyles = map[string][2]string{
	"block":      {"█", "░"},
	"square":     {"■", "□"},
	"small":      {"▪", "▫"},
	"bar":        {"▰", "▱"},
	"dot":        {"●", "○"},
	"diamond":    {"◆", "◇"},
	"line":       {"━", "─"},
	"braille":    {"⣿", "⣀"},
	"shade":      {"▓", "░"},
	"half":       {"▄", "▁"},
	"half-block": {"▌", "░"},
}

var defaultIcons = map[string]string{
	"running":   "≡",
	"completed": "✓",
	"error":     "✗",
	"dirty":     "*",
}

const DefaultMaxTailSize = "10MB"

func Default() *Config {
	return &Config{
		Modules:     []string{"context", "usage", "git", "tools", "agents", "environment"},
		Separator:   " | ",
		Newline:     []string{"tools", "agents", "environment"},
		BarStyle:    "diamond",
		BarStyles:   defaultBarStyles,
		Icons:       defaultIcons,
		MaxTailSize: DefaultMaxTailSize,
		Raw:         make(map[string]json.RawMessage),
	}
}

func DefaultPath() string {
	configDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if configDir == "" {
		home, _ := os.UserHomeDir()
		configDir = filepath.Join(home, ".claude")
	}
	return filepath.Join(configDir, "plugins", "claude-statusline", "config.json")
}

func Load(path string) *Config {
	cfg := Default()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			writeDefault(path, cfg)
		} else {
			debug.Log("config", "load failed (using defaults): %v", err)
		}
		return cfg
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		debug.Log("config", "parse failed (using defaults): %v", err)
		return cfg
	}

	if v, ok := raw["modules"]; ok {
		_ = json.Unmarshal(v, &cfg.Modules)
	}
	if v, ok := raw["separator"]; ok {
		_ = json.Unmarshal(v, &cfg.Separator)
	}
	if v, ok := raw["newline"]; ok {
		_ = json.Unmarshal(v, &cfg.Newline)
	}
	if v, ok := raw["bar_style"]; ok {
		_ = json.Unmarshal(v, &cfg.BarStyle)
	}
	if v, ok := raw["bar_styles"]; ok {
		var custom map[string][2]string
		if json.Unmarshal(v, &custom) == nil {
			for k, v := range custom {
				cfg.BarStyles[k] = v
			}
		}
	}
	if v, ok := raw["icons"]; ok {
		var custom map[string]string
		if json.Unmarshal(v, &custom) == nil {
			for k, v := range custom {
				cfg.Icons[k] = v
			}
		}
	}
	if v, ok := raw["max_tail_size"]; ok {
		var s string
		if json.Unmarshal(v, &s) == nil && s != "" {
			cfg.MaxTailSize = s
		}
	}

	cfg.Raw = raw
	return cfg
}

// BarChars returns the [filled, empty] character pair for the active bar style.
func (c *Config) BarChars() (string, string) {
	if pair, ok := c.BarStyles[c.BarStyle]; ok {
		return pair[0], pair[1]
	}
	return "█", "░"
}

// Icon returns the icon for the given key, falling back to the provided default.
func (c *Config) Icon(key string) string {
	if v, ok := c.Icons[key]; ok {
		return v
	}
	if v, ok := defaultIcons[key]; ok {
		return v
	}
	return ""
}

// MaxTailBytes returns the parsed max_tail_size in bytes.
// Returns 0 (full scan) on invalid input.
func (c *Config) MaxTailBytes() int64 {
	return ParseSize(c.MaxTailSize)
}

// ParseSize parses a human-readable size string like "10MB", "512KB", "1GB".
// Pure numeric strings are treated as bytes. Returns 0 for empty or invalid input.
func ParseSize(s string) int64 {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0
	}

	upper := strings.ToUpper(s)
	var suffix string
	var multiplier int64
	switch {
	case strings.HasSuffix(upper, "GB"):
		suffix = s[:len(s)-2]
		multiplier = 1024 * 1024 * 1024
	case strings.HasSuffix(upper, "MB"):
		suffix = s[:len(s)-2]
		multiplier = 1024 * 1024
	case strings.HasSuffix(upper, "KB"):
		suffix = s[:len(s)-2]
		multiplier = 1024
	default:
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil || n < 0 {
			return 0
		}
		return n
	}

	n, err := strconv.ParseFloat(strings.TrimSpace(suffix), 64)
	if err != nil || n < 0 {
		return 0
	}
	return int64(n * float64(multiplier))
}

func (c *Config) ModuleConfig(name string) ModuleConf {
	var mc ModuleConf
	if v, ok := c.Raw[name]; ok {
		_ = json.Unmarshal(v, &mc)
	}
	return mc
}

func (c *Config) IsEnabled(name string) bool {
	return slices.Contains(c.Modules, name)
}

func (c *Config) StartsNewline(name string) bool {
	return slices.Contains(c.Newline, name)
}

// writeDefault writes a default config file with all available options.
func writeDefault(path string, cfg *Config) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		debug.Log("config", "create config dir: %v", err)
		return
	}

	out := map[string]any{
		"modules":   cfg.Modules,
		"separator": cfg.Separator,
	}

	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		debug.Log("config", "marshal default config: %v", err)
		return
	}
	if err := os.WriteFile(path, append(data, '\n'), 0600); err != nil {
		debug.Log("config", "write default config: %v", err)
		return
	}
	debug.Log("config", "created default config: %s", path)
}
