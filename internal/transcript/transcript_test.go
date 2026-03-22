package transcript

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestParseToolUseAndResult(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","message":{"content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/tmp/foo.go"}}]}}
{"timestamp":"2025-03-19T12:00:01Z","message":{"content":[{"type":"tool_result","tool_use_id":"t1","is_error":false}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, err := Parse(path, 0)
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
	data, _ := Parse(path, 0)
	if len(data.Agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(data.Agents))
	}
	if data.Agents[0].Type != "explore" || data.Agents[0].Model != "haiku" {
		t.Fatalf("unexpected agent: %+v", data.Agents[0])
	}
}

func TestParseMissingFile(t *testing.T) {
	data, err := Parse("/nonexistent/file.jsonl", 0)
	if err != nil {
		t.Fatal("should not error on missing file")
	}
	if len(data.Tools) != 0 {
		t.Fatal("should return empty data")
	}
}

func TestUserMessageResetsCompletedTools(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/tmp/a.go"}}]}}
{"timestamp":"2025-03-19T12:00:01Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_result","tool_use_id":"t1","is_error":false}]}}
{"timestamp":"2025-03-19T12:00:02Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t2","name":"Edit","input":{"file_path":"/tmp/b.go"}}]}}
{"timestamp":"2025-03-19T12:00:03Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_result","tool_use_id":"t2","is_error":true}]}}
{"timestamp":"2025-03-19T12:00:04Z","type":"user","message":{"role":"user","content":"fix it"}}
{"timestamp":"2025-03-19T12:00:05Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t3","name":"Write","input":{"file_path":"/tmp/c.go"}}]}}
{"timestamp":"2025-03-19T12:00:06Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_result","tool_use_id":"t3","is_error":false}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, err := Parse(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Only Write (after user message) should remain; Read and Edit (before) should be cleared
	if len(data.Tools) != 1 {
		t.Fatalf("expected 1 tool after reset, got %d: %+v", len(data.Tools), data.Tools)
	}
	if data.Tools[0].Name != "Write" {
		t.Fatalf("expected Write, got %s", data.Tools[0].Name)
	}
	// SessionToolNames should contain all 3 tools (not affected by reset)
	if len(data.SessionToolNames) != 3 {
		t.Fatalf("expected 3 session tool names, got %d: %v", len(data.SessionToolNames), data.SessionToolNames)
	}
}

func TestUserMessageKeepsRunningTools(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"sleep 60"}}]}}
{"timestamp":"2025-03-19T12:00:01Z","type":"user","message":{"role":"user","content":"status?"}}
{"timestamp":"2025-03-19T12:00:02Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t2","name":"Read","input":{"file_path":"/tmp/a.go"}}]}}
{"timestamp":"2025-03-19T12:00:03Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_result","tool_use_id":"t2","is_error":false}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, err := Parse(path, 0)
	if err != nil {
		t.Fatal(err)
	}
	// Bash (running, no result) + Read (after user msg) = 2 tools
	if len(data.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d: %+v", len(data.Tools), data.Tools)
	}
	if data.Tools[0].Name != "Bash" || data.Tools[0].Status != "running" {
		t.Fatalf("expected running Bash, got %+v", data.Tools[0])
	}
	if data.Tools[1].Name != "Read" || data.Tools[1].Status != "completed" {
		t.Fatalf("expected completed Read, got %+v", data.Tools[1])
	}
}

func TestToolErrorStatus(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"exit 1"}}]}}
{"timestamp":"2025-03-19T12:00:01Z","type":"assistant","message":{"role":"assistant","content":[{"type":"tool_result","tool_use_id":"t1","is_error":true}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, _ := Parse(path, 0)
	if len(data.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(data.Tools))
	}
	if data.Tools[0].Status != "error" {
		t.Fatalf("expected error status, got %s", data.Tools[0].Status)
	}
}

func TestSessionToolNamesSortedByRecency(t *testing.T) {
	// Read at :00, Edit at :01, Write at :02 -> sorted: Write, Edit, Read
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/tmp/a.go"}}]}}
{"timestamp":"2025-03-19T12:00:00Z","type":"assistant","message":{"content":[{"type":"tool_result","tool_use_id":"t1"}]}}
{"timestamp":"2025-03-19T12:00:01Z","type":"assistant","message":{"content":[{"type":"tool_use","id":"t2","name":"Edit","input":{"file_path":"/tmp/b.go"}}]}}
{"timestamp":"2025-03-19T12:00:01Z","type":"assistant","message":{"content":[{"type":"tool_result","tool_use_id":"t2"}]}}
{"timestamp":"2025-03-19T12:00:02Z","type":"assistant","message":{"content":[{"type":"tool_use","id":"t3","name":"Write","input":{"file_path":"/tmp/c.go"}}]}}
{"timestamp":"2025-03-19T12:00:02Z","type":"assistant","message":{"content":[{"type":"tool_result","tool_use_id":"t3"}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, _ := Parse(path, 0)
	if len(data.SessionToolNames) != 3 {
		t.Fatalf("expected 3 session tools, got %d", len(data.SessionToolNames))
	}
	// Most recent first
	if data.SessionToolNames[0] != "Write" || data.SessionToolNames[1] != "Edit" || data.SessionToolNames[2] != "Read" {
		t.Fatalf("expected [Write Edit Read], got %v", data.SessionToolNames)
	}
}

func TestSessionToolNamesLimitedTo6(t *testing.T) {
	// Create 8 different tools, expect only most recent 6
	var lines string
	tools := []string{"A", "B", "C", "D", "E", "F", "G", "H"}
	for i, name := range tools {
		ts := fmt.Sprintf("2025-03-19T12:00:%02dZ", i)
		lines += fmt.Sprintf(`{"timestamp":"%s","type":"assistant","message":{"content":[{"type":"tool_use","id":"t%d","name":"%s","input":{}}]}}`, ts, i, name) + "\n"
		lines += fmt.Sprintf(`{"timestamp":"%s","type":"assistant","message":{"content":[{"type":"tool_result","tool_use_id":"t%d"}]}}`, ts, i) + "\n"
	}
	path := writeTempJSONL(t, lines)
	data, _ := Parse(path, 0)
	if len(data.SessionToolNames) != 6 {
		t.Fatalf("expected 6 session tools, got %d: %v", len(data.SessionToolNames), data.SessionToolNames)
	}
	// Most recent 6: H, G, F, E, D, C (A and B dropped)
	if data.SessionToolNames[0] != "H" || data.SessionToolNames[5] != "C" {
		t.Fatalf("expected [H G F E D C], got %v", data.SessionToolNames)
	}
}

func TestSessionToolNamesExcludesManagementTools(t *testing.T) {
	jsonl := `{"timestamp":"2025-03-19T12:00:00Z","type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Read","input":{"file_path":"/tmp/a.go"}}]}}
{"timestamp":"2025-03-19T12:00:00Z","type":"assistant","message":{"content":[{"type":"tool_result","tool_use_id":"t1"}]}}
{"timestamp":"2025-03-19T12:00:01Z","type":"assistant","message":{"content":[{"type":"tool_use","id":"t2","name":"Task","input":{"subagent_type":"explore","description":"Search code"}}]}}
{"timestamp":"2025-03-19T12:00:02Z","type":"assistant","message":{"content":[{"type":"tool_use","id":"a1","name":"Agent","input":{"subagent_type":"Explore","description":"Search code"}}]}}
`
	path := writeTempJSONL(t, jsonl)
	data, _ := Parse(path, 0)
	if len(data.SessionToolNames) != 1 {
		t.Fatalf("expected 1 session tool (only Read), got %d: %v", len(data.SessionToolNames), data.SessionToolNames)
	}
	if data.SessionToolNames[0] != "Read" {
		t.Fatalf("expected Read, got %s", data.SessionToolNames[0])
	}
}

func TestTailScanSkipsOldEntries(t *testing.T) {
	// Build a file where the first agent is in the "old" part and the second
	// agent is in the "tail" part. With a small maxTailBytes only the second
	// agent should be found.
	old := `{"timestamp":"2025-01-01T00:00:00Z","type":"assistant","message":{"content":[{"type":"tool_use","id":"a1","name":"Agent","input":{"description":"Old agent"}}]}}` + "\n"
	recent := `{"timestamp":"2025-01-01T01:00:00Z","type":"assistant","message":{"content":[{"type":"tool_use","id":"a2","name":"Agent","input":{"description":"New agent"}}]}}` + "\n"

	// Pad old section so it's clearly larger than our tail window
	padding := ""
	for len(padding) < 2048 {
		padding += `{"timestamp":"2025-01-01T00:30:00Z","type":"progress"}` + "\n"
	}

	content := old + padding + recent
	path := writeTempJSONL(t, content)

	// Full scan: both agents found
	full, _ := Parse(path, 0)
	if len(full.Agents) != 2 {
		t.Fatalf("full scan: expected 2 agents, got %d", len(full.Agents))
	}

	// Tail scan with small window: only recent agent
	tail, _ := Parse(path, 512)
	if len(tail.Agents) != 1 {
		t.Fatalf("tail scan: expected 1 agent, got %d", len(tail.Agents))
	}
	if tail.Agents[0].Description != "New agent" {
		t.Fatalf("tail scan: expected 'New agent', got %q", tail.Agents[0].Description)
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
