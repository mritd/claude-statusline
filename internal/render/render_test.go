package render

import (
	"strings"
	"testing"

	"github.com/mritd/claude-statusline/internal/config"
	"github.com/mritd/claude-statusline/internal/module"
)

type fakeModule struct {
	name   string
	vars   map[string]string
	format string
}

func (f *fakeModule) Name() string                               { return f.name }
func (f *fakeModule) Collect(ctx *module.Context) error          { return nil }
func (f *fakeModule) Vars(ctx *module.Context) map[string]string { return f.vars }
func (f *fakeModule) DefaultFormat() string                      { return f.format }

func TestRenderSingleLine(t *testing.T) {
	reg := module.NewRegistry()
	reg.Register(&fakeModule{name: "a", vars: map[string]string{"x": "1"}, format: "A={x}"})
	reg.Register(&fakeModule{name: "b", vars: map[string]string{"y": "2"}, format: "B={y}"})

	cfg := &config.Config{
		Modules:   []string{"a", "b"},
		Separator: " | ",
	}
	ctx := &module.Context{}
	out := Render(ctx, reg, cfg)
	if out != "A=1 | B=2" {
		t.Fatalf("unexpected: %q", out)
	}
}

func TestRenderNewline(t *testing.T) {
	reg := module.NewRegistry()
	reg.Register(&fakeModule{name: "a", vars: map[string]string{"x": "1"}, format: "{x}"})
	reg.Register(&fakeModule{name: "b", vars: map[string]string{"y": "2"}, format: "{y}"})

	cfg := &config.Config{
		Modules:   []string{"a", "b"},
		Separator: " | ",
		Newline:   []string{"b"},
	}
	ctx := &module.Context{}
	out := Render(ctx, reg, cfg)
	parts := strings.Split(out, "\n")
	if len(parts) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(parts), out)
	}
}
