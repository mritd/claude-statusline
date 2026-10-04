package config

import (
	"encoding/json"
	"errors"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/mritd/claude-statusline/internal/debug"
)

type Config struct {
	Modules     []string             `json:"modules"`
	Separator   string               `json:"separator"`
	Newline     []string             `json:"newline"`
	BarStyle    string               `json:"bar_style"`
	BarStyles   map[string][2]string `json:"-"` // merged from "bar_styles" in Load
	Icons       map[string]string    `json:"-"` // merged from "icons" in Load
	MaxTailSize string               `json:"max_tail_size"`

	moduleConfs map[string]ModuleConf // per-module sections, parsed once in Load
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
		BarStyles:   maps.Clone(defaultBarStyles),
		Icons:       maps.Clone(defaultIcons),
		MaxTailSize: DefaultMaxTailSize,
	}
}

// ClaudeDir returns the Claude Code config directory: $CLAUDE_CONFIG_DIR,
// or ~/.claude when unset.
func ClaudeDir() string {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".claude")
}

// PluginDir returns the directory holding this plugin's config and cache.
func PluginDir() string {
	return filepath.Join(ClaudeDir(), "plugins", "claude-statusline")
}

func DefaultPath() string {
	return filepath.Join(PluginDir(), "config.json")
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

	// Top-level fields overlay the defaults. A mistyped field is skipped and
	// the rest still apply; malformed JSON falls back to defaults.
	if err := json.Unmarshal(data, cfg); err != nil {
		var typeErr *json.UnmarshalTypeError
		if !errors.As(err, &typeErr) {
			debug.Log("config", "parse failed (using defaults): %v", err)
			return Default()
		}
		debug.Log("config", "ignoring mistyped field: %v", err)
	}
	if cfg.MaxTailSize == "" {
		cfg.MaxTailSize = DefaultMaxTailSize
	}

	var raw map[string]json.RawMessage
	_ = json.Unmarshal(data, &raw)

	// bar_styles and icons merge key by key; a map with any bad value is
	// ignored as a whole so defaults are never replaced by zero values.
	var styles map[string][2]string
	if v, ok := raw["bar_styles"]; ok && json.Unmarshal(v, &styles) == nil {
		maps.Copy(cfg.BarStyles, styles)
	}
	var icons map[string]string
	if v, ok := raw["icons"]; ok && json.Unmarshal(v, &icons) == nil {
		maps.Copy(cfg.Icons, icons)
	}

	// Module sections keep whatever decoded; a mistyped field stays zero.
	cfg.moduleConfs = make(map[string]ModuleConf, len(raw))
	for name, v := range raw {
		var mc ModuleConf
		_ = json.Unmarshal(v, &mc)
		cfg.moduleConfs[name] = mc
	}
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

// ModuleConfig returns the per-module section for name, or the zero value
// when the config file has none.
func (c *Config) ModuleConfig(name string) ModuleConf {
	return c.moduleConfs[name]
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
