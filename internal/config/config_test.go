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
