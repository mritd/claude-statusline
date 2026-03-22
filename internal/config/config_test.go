package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if len(cfg.Modules) != 4 {
		t.Fatalf("expected 4 default modules, got %d", len(cfg.Modules))
	}
	expected := []string{"context", "usage", "git", "tools"}
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
	if len(cfg.Modules) != 4 {
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
