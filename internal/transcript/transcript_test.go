package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseToolUseAndResult(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","message":{"content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/tmp/foo.go"}}]}}
{"timestamp":"2025-03-19T12:00:01Z","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":false}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, err := Parse(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(data.Tools))
	}
	if data.Tools[0].Name != "Read" || data.Tools[0].Status != "completed" {
		t.Fatalf("unexpected tool: %+v", data.Tools[0])
	}
	if data.Tools[0].Target != "/tmp/foo.go" {
		t.Fatalf("unexpected target: %s", data.Tools[0].Target)
	}
}

func TestParseAgent(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","message":{"content":[{"type":"tool_use","id":"a1","name":"Task","input":{"subagent_type":"explore","model":"haiku","description":"Finding auth code"}}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, _ := Parse(path)
	if len(data.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(data.Agents))
	}
	if data.Agents[0].Type != "explore" || data.Agents[0].Model != "haiku" {
		t.Fatalf("unexpected agent: %+v", data.Agents[0])
	}
}

func TestParseTodoWrite(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","message":{"content":[{"type":"tool_use","id":"tw1","name":"TodoWrite","input":{"todos":[{"content":"Fix bug","status":"in_progress"},{"content":"Add tests","status":"pending"}]}}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, _ := Parse(path)
	if len(data.Todos) != 2 {
		t.Fatalf("expected 2 todos, got %d", len(data.Todos))
	}
	if data.Todos[0].Status != "in_progress" {
		t.Fatalf("unexpected status: %s", data.Todos[0].Status)
	}
}

func TestParseTaskCreateUpdate(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","message":{"content":[{"type":"tool_use","id":"tc1","name":"TaskCreate","input":{"subject":"Fix auth"}}]}}
{"timestamp":"2025-03-19T12:00:01Z","message":{"content":[{"type":"tool_use","id":"tu1","name":"TaskUpdate","input":{"taskId":"1","status":"completed"}}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, _ := Parse(path)
	if len(data.Todos) != 1 {
		t.Fatalf("expected 1 todo, got %d", len(data.Todos))
	}
	if data.Todos[0].Status != "completed" {
		t.Fatalf("expected completed, got %s", data.Todos[0].Status)
	}
}

func TestParseSessionStart(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","message":{"content":[]}}
{"timestamp":"2025-03-19T12:00:05Z","message":{"content":[]}}
`
	path := writeTempJSONL(t, jsonl)
	data, _ := Parse(path)
	if data.SessionStart.IsZero() {
		t.Fatal("session start should be set")
	}
}

func TestParseMissingFile(t *testing.T) {
	data, err := Parse("/nonexistent/file.jsonl")
	if err != nil {
		t.Fatal("should not error on missing file")
	}
	if len(data.Tools) != 0 {
		t.Fatal("should return empty data")
	}
}

func writeTempJSONL(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "transcript.jsonl")
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		t.Fatal(err)
	}
	return path
}
