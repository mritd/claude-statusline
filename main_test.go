package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func TestMainIntegration(t *testing.T) {
	input := `{"model":{"display_name":"Claude Opus 4.6"},"context_window":{"context_window_size":200000,"current_usage":{"input_tokens":90000,"cache_creation_input_tokens":5000,"cache_read_input_tokens":5000}},"cwd":"/tmp/test-project"}`

	// Use temp dir so test gets default config, not user's real config
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	oldIn := os.Stdin
	os.Stdin, _ = makeTempStdin(t, input)

	run()

	os.Stdin = oldIn
	_ = w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "Ctx") {
		t.Fatalf("expected Ctx in output, got: %q", output)
	}
	if !strings.Contains(output, "◆") {
		t.Fatalf("expected bar chars in output, got: %q", output)
	}
	if !strings.Contains(output, "40%") {
		t.Fatalf("expected 40%% in output, got: %q", output)
	}
}

func makeTempStdin(t *testing.T, content string) (*os.File, error) {
	t.Helper()
	f, err := os.CreateTemp("", "stdin-*")
	if err != nil {
		return nil, err
	}
	_, _ = f.WriteString(content)
	_, _ = f.Seek(0, 0)
	t.Cleanup(func() { _ = os.Remove(f.Name()) })
	return f, nil
}
