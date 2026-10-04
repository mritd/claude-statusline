package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if len(cfg.Modules) != 6 {
		t.Fatalf("expected 6 default modules, got %d", len(cfg.Modules))
	}
	expected := []string{"context", "usage", "git", "tools", "agents", "environment"}
	for i, e := range expected {
		if cfg.Modules[i] != e {
			t.Fatalf("unexpected default modules: %v", cfg.Modules)
		}
	}
	if cfg.Separator != " | " {
		t.Fatalf("unexpected separator: %q", cfg.Separator)
	}
}

func TestLoadMissingFile(t *testing.T) {
	cfg := Load("/nonexistent/path/config.json")
	if len(cfg.Modules) != 6 {
		t.Fatalf("missing file should return defaults, got %d modules", len(cfg.Modules))
	}
}

func TestLoadPartialOverride(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	err := os.WriteFile(path, []byte(`{"modules":["context","git"],"separator":" | "}`), 0600)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Load(path)
	if len(cfg.Modules) != 2 {
		t.Fatalf("expected 2 modules, got %d", len(cfg.Modules))
	}
	if cfg.Separator != " | " {
		t.Fatalf("expected custom separator, got %q", cfg.Separator)
	}
}

func TestParseSize(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"10MB", 10 * 1024 * 1024},
		{"10mb", 10 * 1024 * 1024},
		{"512KB", 512 * 1024},
		{"1GB", 1024 * 1024 * 1024},
		{"1024", 1024},
		{"0", 0},
		{"", 0},
		{"invalid", 0},
		{"  5MB  ", 5 * 1024 * 1024},
	}
	for _, tt := range tests {
		got := ParseSize(tt.input)
		if got != tt.want {
			t.Errorf("ParseSize(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestMaxTailSizeConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	_ = os.WriteFile(path, []byte(`{"max_tail_size":"50MB"}`), 0600)
	cfg := Load(path)
	if cfg.MaxTailSize != "50MB" {
		t.Fatalf("expected 50MB, got %s", cfg.MaxTailSize)
	}
	if cfg.MaxTailBytes() != 50*1024*1024 {
		t.Fatalf("expected %d bytes, got %d", 50*1024*1024, cfg.MaxTailBytes())
	}
}

func TestModuleConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{"modules":["context"],"context":{"bar_width":20,"format":"Ctx {bar} {percent}"}}`
	_ = os.WriteFile(path, []byte(data), 0600)
	cfg := Load(path)
	mc := cfg.ModuleConfig("context")
	if mc.Format != "Ctx {bar} {percent}" {
		t.Fatalf("expected custom format, got %q", mc.Format)
	}
	if mc.BarWidth != 20 {
		t.Fatalf("expected barWidth 20, got %d", mc.BarWidth)
	}
}

func TestLoadMergesMapsWithoutMutatingDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	_ = os.WriteFile(path, []byte(`{"icons":{"dirty":"!"},"bar_styles":{"custom":["#","-"]}}`), 0600)
	cfg := Load(path)
	if cfg.Icon("dirty") != "!" || cfg.Icon("running") != "≡" {
		t.Fatalf("expected merged icons, got %v", cfg.Icons)
	}
	if _, ok := cfg.BarStyles["custom"]; !ok {
		t.Fatalf("expected custom bar style, got %v", cfg.BarStyles)
	}
	if _, ok := cfg.BarStyles["diamond"]; !ok {
		t.Fatalf("expected default bar styles kept, got %v", cfg.BarStyles)
	}
	if defaultIcons["dirty"] != "*" {
		t.Fatalf("Load mutated package defaults: %v", defaultIcons)
	}
	if _, ok := defaultBarStyles["custom"]; ok {
		t.Fatal("Load mutated package default bar styles")
	}
}

func TestLoadSkipsMistypedField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	_ = os.WriteFile(path, []byte(`{"separator":123,"modules":["git"],"git":{"format":"{branch}"}}`), 0600)
	cfg := Load(path)
	if cfg.Separator != " | " {
		t.Fatalf("expected default separator for mistyped field, got %q", cfg.Separator)
	}
	if len(cfg.Modules) != 1 || cfg.Modules[0] != "git" {
		t.Fatalf("expected other fields applied, got %v", cfg.Modules)
	}
	if cfg.ModuleConfig("git").Format != "{branch}" {
		t.Fatalf("expected module config parsed, got %+v", cfg.ModuleConfig("git"))
	}
}

func TestLoadMalformedFallsBackToDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	_ = os.WriteFile(path, []byte(`{"modules":["git"`), 0600)
	cfg := Load(path)
	if len(cfg.Modules) != 6 {
		t.Fatalf("expected defaults on malformed JSON, got %v", cfg.Modules)
	}
}

func TestLoadIgnoresBadMapsAndKeepsModuleFields(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := `{"icons":{"dirty":1,"running":"R"},"bar_styles":null,"usage":{"format":"X","bar_width":"10"}}`
	_ = os.WriteFile(path, []byte(data), 0600)
	cfg := Load(path)
	if cfg.Icon("dirty") != "*" || cfg.Icon("running") != "≡" {
		t.Fatalf("bad icons map should be ignored, got %v", cfg.Icons)
	}
	if f, e := cfg.BarChars(); f != "◆" || e != "◇" {
		t.Fatalf("null bar_styles should keep defaults, got %q %q", f, e)
	}
	if mc := cfg.ModuleConfig("usage"); mc.Format != "X" {
		t.Fatalf("mistyped bar_width should not drop format, got %+v", mc)
	}
}
