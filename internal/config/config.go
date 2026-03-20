package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"

	"github.com/mritd/claude-statusline/internal/debug"
)

type Config struct {
	Modules   []string                   `json:"modules"`
	Separator string                     `json:"separator"`
	Newline   []string                   `json:"newline"`
	BarStyle  string                     `json:"bar_style"`
	BarStyles map[string][2]string       `json:"bar_styles"`
	Raw       map[string]json.RawMessage `json:"-"`
}

type ModuleConf struct {
	Format                 string `json:"format"`
	BarWidth               int    `json:"bar_width"`
	CacheTTLSeconds        int    `json:"cache_ttl_seconds"`
	FailureCacheTTLSeconds int    `json:"failure_cache_ttl_seconds"`
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

func Default() *Config {
	return &Config{
		Modules:   []string{"context", "usage", "todos", "git"},
		Separator: " | ",
		BarStyle:  "block",
		BarStyles: defaultBarStyles,
		Raw:       make(map[string]json.RawMessage),
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
			// Merge custom styles into defaults (custom wins on conflict)
			for k, v := range custom {
				cfg.BarStyles[k] = v
			}
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
		"modules":    cfg.Modules,
		"separator":  cfg.Separator,
		"bar_style":  cfg.BarStyle,
		"bar_styles": cfg.BarStyles,
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
