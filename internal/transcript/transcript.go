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

type Data struct {
	Tools             []ToolEntry
	Agents            []AgentEntry
	SessionName       string
	SessionToolNames  []string       // all unique tool names across entire session (never reset)
	SessionToolCounts map[string]int // cumulative tool call counts across session (never reset)
}

type ToolEntry struct {
	ID, Name, Target, Status string
	StartTime, EndTime       time.Time
}

type AgentEntry struct {
	ID, Type, Model, Description, Status string
	StartTime, EndTime                   time.Time
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

		if entry.Type == "custom-title" && entry.CustomTitle != "" {
			data.SessionName = entry.CustomTitle
		} else if entry.Slug != "" {
			data.SessionName = entry.Slug
		}

		// Reset completed/error tools on each user message so stats
		// reflect only the current turn.
		if entry.Type == "user" {
			var kept []ToolEntry
			newMap := make(map[string]int)
			for _, t := range data.Tools {
				if t.Status == "running" {
					newMap[t.ID] = len(kept)
					kept = append(kept, t)
				}
			}
			data.Tools = kept
			toolMap = newMap
		}

		// Decode message content only when present (user messages have
		// string content that would fail decoding to []contentBlock).
		if entry.RawMessage == nil {
			continue
		}
		var msg jsonMessage
		if err := json.Unmarshal(*entry.RawMessage, &msg); err != nil {
			continue
		}

		for i := range msg.Content {
			block := &msg.Content[i]
			switch block.Type {
			case "tool_use":
				handleToolUse(data, block, entry.Timestamp, toolMap)
				if !managementTools[block.Name] {
					sessionToolLast[block.Name] = entry.Timestamp
					sessionToolCounts[block.Name]++
				}
			case "tool_result":
				handleToolResult(data, block, entry.Timestamp, toolMap)
			}
		}
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
	switch block.Name {
	case "Task", "Agent":
		data.Agents = append(data.Agents, AgentEntry{
			ID:          block.ID,
			Type:        block.Input.SubagentType,
			Model:       block.Input.Model,
			Description: block.Input.Description,
			Status:      "running",
			StartTime:   ts,
		})
	default:
		data.Tools = append(data.Tools, ToolEntry{
			ID:        block.ID,
			Name:      block.Name,
			Target:    extractTarget(block),
			Status:    "running",
			StartTime: ts,
		})
		toolMap[block.ID] = len(data.Tools) - 1
	}
}

func handleToolResult(data *Data, block *contentBlock, ts time.Time, toolMap map[string]int) {
	idx, ok := toolMap[block.ToolUseID]
	if !ok || idx >= len(data.Tools) {
		// Check if it's an agent result
		for i := range data.Agents {
			if data.Agents[i].ID == block.ToolUseID {
				data.Agents[i].Status = "completed"
				data.Agents[i].EndTime = ts
				break
			}
		}
		return
	}
	if block.IsError {
		data.Tools[idx].Status = "error"
	} else {
		data.Tools[idx].Status = "completed"
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
	Timestamp   time.Time        `json:"timestamp"`
	Type        string           `json:"type"`
	CustomTitle string           `json:"customTitle"`
	Slug        string           `json:"slug"`
	RawMessage  *json.RawMessage `json:"message"`
}

// managementTools are tool names handled by dedicated branches in handleToolUse
// (not tracked as regular tools or in sessionToolLast).
var managementTools = map[string]bool{
	"Task":  true,
	"Agent": true,
}

type jsonMessage struct {
	Content []contentBlock `json:"content"`
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
