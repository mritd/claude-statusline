package transcript

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/mritd/claude-statusline/internal/debug"
)

const maxSessionTools = 6

// Status is the lifecycle state of a tool or agent call.
type Status string

const (
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusError     Status = "error"
)

type Data struct {
	Tools             []ToolEntry
	Agents            []AgentEntry
	SessionToolNames  []string       // all unique tool names across entire session (never reset)
	SessionToolCounts map[string]int // cumulative tool call counts across session (never reset)
}

type ToolEntry struct {
	ID, Name, Target   string
	Status             Status
	StartTime, EndTime time.Time
}

type AgentEntry struct {
	ID, Type, Model, Description string
	Status                       Status
	StartTime, EndTime           time.Time
}

func Parse(path string, maxTailBytes int64) (*Data, error) {
	data := &Data{}
	if path == "" {
		return data, nil
	}

	f, err := os.Open(path)
	if err != nil {
		debug.Log("transcript", "open failed: %v", err)
		return data, nil
	}
	defer func() { _ = f.Close() }()

	// Tail scan: skip to the last maxTailBytes of the file
	var seeked bool
	if maxTailBytes > 0 {
		if info, err := f.Stat(); err == nil && info.Size() > maxTailBytes {
			debug.Log("transcript", "tail scan: file %d bytes, seeking to last %d", info.Size(), maxTailBytes)
			_, _ = f.Seek(-maxTailBytes, io.SeekEnd)
			seeked = true
		}
	}

	toolMap := make(map[string]int)
	sessionToolLast := make(map[string]time.Time) // last usage time per tool name
	sessionToolCounts := make(map[string]int)     // cumulative call counts per tool name
	var lastPromptTS time.Time                    // timestamp of the last user prompt

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	// After seeking into the middle of the file, the first line is likely
	// partial (we landed mid-line). Discard it.
	if seeked {
		scanner.Scan()
	}

	for scanner.Scan() {
		var entry jsonEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		if entry.Message == nil {
			continue
		}
		// Typed prompts carry string content; everything else (assistant
		// turns, tool_result carriers) is a list of content blocks.
		var blocks []contentBlock
		if content := entry.Message.Content; len(content) > 0 && content[0] == '[' {
			if err := json.Unmarshal(content, &blocks); err != nil {
				continue
			}
		}

		// Reset completed/error tools on each user prompt so stats reflect
		// only the current turn.
		if isUserPrompt(&entry, blocks) {
			lastPromptTS = entry.Timestamp
			var kept []ToolEntry
			newMap := make(map[string]int)
			for _, t := range data.Tools {
				if t.Status == StatusRunning {
					newMap[t.ID] = len(kept)
					kept = append(kept, t)
				}
			}
			data.Tools = kept
			toolMap = newMap
		}

		for i := range blocks {
			block := &blocks[i]
			switch block.Type {
			case "tool_use":
				handleToolUse(data, block, entry.Timestamp, toolMap)
				if !isAgentTool(block.Name) {
					sessionToolLast[block.Name] = entry.Timestamp
					sessionToolCounts[block.Name]++
				}
			case "tool_result":
				handleToolResult(data, block, entry.Timestamp, toolMap)
			}
		}
	}

	// After tail scan, a tool_use near the scan boundary may lack its
	// tool_result (truncated away). These orphan entries stay "running"
	// forever. Discard running tools that started before the last user
	// prompt — real running tools belong to the current (latest) turn.
	// Only apply when tail scan was used; full scans preserve legitimate
	// cross-turn running tools.
	if seeked && !lastPromptTS.IsZero() {
		var cleaned []ToolEntry
		for _, t := range data.Tools {
			if t.Status == StatusRunning && t.StartTime.Before(lastPromptTS) {
				debug.Log("transcript", "discarding orphan running tool %s (%s)", t.Name, t.ID)
				continue
			}
			cleaned = append(cleaned, t)
		}
		data.Tools = cleaned
	}

	// Build session tool names sorted by most recent usage, limited to 6.
	for name := range sessionToolLast {
		data.SessionToolNames = append(data.SessionToolNames, name)
	}
	slices.SortFunc(data.SessionToolNames, func(a, b string) int {
		ta, tb := sessionToolLast[a], sessionToolLast[b]
		if tb.Before(ta) {
			return -1
		}
		if ta.Before(tb) {
			return 1
		}
		return 0
	})
	if len(data.SessionToolNames) > maxSessionTools {
		data.SessionToolNames = data.SessionToolNames[:maxSessionTools]
	}
	data.SessionToolCounts = sessionToolCounts

	return data, nil
}

func handleToolUse(data *Data, block *contentBlock, ts time.Time, toolMap map[string]int) {
	if isAgentTool(block.Name) {
		data.Agents = append(data.Agents, AgentEntry{
			ID:          block.ID,
			Type:        block.Input.SubagentType,
			Model:       block.Input.Model,
			Description: block.Input.Description,
			Status:      StatusRunning,
			StartTime:   ts,
		})
		return
	}
	data.Tools = append(data.Tools, ToolEntry{
		ID:        block.ID,
		Name:      block.Name,
		Target:    extractTarget(block),
		Status:    StatusRunning,
		StartTime: ts,
	})
	toolMap[block.ID] = len(data.Tools) - 1
}

// isUserPrompt reports whether the entry is a prompt the user typed, which
// starts a new turn. Tool results are also recorded as "user" entries, and
// meta entries (skill bodies, command caveats) are injected mid-turn; neither
// starts a turn.
func isUserPrompt(entry *jsonEntry, blocks []contentBlock) bool {
	if entry.Type != "user" || entry.IsMeta {
		return false
	}
	for i := range blocks {
		if blocks[i].Type == "tool_result" {
			return false
		}
	}
	return true
}

// isAgentTool reports whether the tool spawns a subagent. Agent tools are
// tracked by the agents module, not counted as regular tools.
func isAgentTool(name string) bool {
	return name == "Task" || name == "Agent"
}

func handleToolResult(data *Data, block *contentBlock, ts time.Time, toolMap map[string]int) {
	idx, ok := toolMap[block.ToolUseID]
	if !ok || idx >= len(data.Tools) {
		// Check if it's an agent result
		for i := range data.Agents {
			if data.Agents[i].ID == block.ToolUseID {
				data.Agents[i].Status = StatusCompleted
				data.Agents[i].EndTime = ts
				break
			}
		}
		return
	}
	if block.IsError {
		data.Tools[idx].Status = StatusError
	} else {
		data.Tools[idx].Status = StatusCompleted
	}
	data.Tools[idx].EndTime = ts
}

func extractTarget(block *contentBlock) string {
	if block.Input.FilePath != "" {
		return truncatePath(block.Input.FilePath, 20)
	}
	if block.Input.Path != "" {
		return truncatePath(block.Input.Path, 20)
	}
	if block.Input.Pattern != "" {
		return truncatePath(block.Input.Pattern, 20)
	}
	if block.Input.Command != "" {
		cmd := block.Input.Command
		if i := strings.IndexAny(cmd, "\n\r"); i >= 0 {
			cmd = cmd[:i]
		}
		runes := []rune(cmd)
		if len(runes) > 30 {
			return string(runes[:30]) + "..."
		}
		return cmd
	}
	return ""
}

func truncatePath(p string, maxLen int) string {
	runes := []rune(p)
	if len(runes) <= maxLen {
		return p
	}
	return "..." + string(runes[len(runes)-maxLen+3:])
}

type jsonEntry struct {
	Timestamp time.Time    `json:"timestamp"`
	Type      string       `json:"type"`
	IsMeta    bool         `json:"isMeta"`
	Message   *jsonMessage `json:"message"`
}

type jsonMessage struct {
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type      string     `json:"type"`
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Input     blockInput `json:"input"`
	ToolUseID string     `json:"tool_use_id"`
	IsError   bool       `json:"is_error"`
}

type blockInput struct {
	FilePath     string `json:"file_path"`
	Path         string `json:"path"`
	Pattern      string `json:"pattern"`
	Command      string `json:"command"`
	SubagentType string `json:"subagent_type"`
	Model        string `json:"model"`
	Description  string `json:"description"`
}
