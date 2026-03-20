package module

import "testing"

func TestExpandFormat(t *testing.T) {
	vars := map[string]string{
		"bar":     "█████░░░░░",
		"percent": "45%",
	}
	got := ExpandFormat("Context {bar} {percent}", vars)
	if got != "Context █████░░░░░ 45%" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestExpandFormatEscapedBrace(t *testing.T) {
	vars := map[string]string{"x": "1"}
	got := ExpandFormat("val={{x}} is {x}", vars)
	if got != "val={x} is 1" {
		t.Fatalf("unexpected: %q", got)
	}
}

func TestExpandFormatUnknownVar(t *testing.T) {
	got := ExpandFormat("{unknown}", map[string]string{})
	if got != "" {
		t.Fatalf("unknown var should expand to empty, got %q", got)
	}
}

func TestRegistryOrder(t *testing.T) {
	r := NewRegistry()
	r.Register(&stubModule{name: "a"})
	r.Register(&stubModule{name: "b"})
	r.Register(&stubModule{name: "c"})

	modules := r.Enabled([]string{"c", "a"})
	if len(modules) != 2 {
		t.Fatalf("expected 2, got %d", len(modules))
	}
	if modules[0].Name() != "c" || modules[1].Name() != "a" {
		t.Fatalf("wrong order: %s, %s", modules[0].Name(), modules[1].Name())
	}
}

type stubModule struct{ name string }

func (s *stubModule) Name() string                        { return s.name }
func (s *stubModule) Collect(ctx *Context) error          { return nil }
func (s *stubModule) Vars(ctx *Context) map[string]string { return nil }
func (s *stubModule) DefaultFormat() string               { return "" }
