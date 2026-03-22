package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/mritd/claude-statusline/internal/debug"
)

const maxSessionTools = 6

type Data struct {
	Tools            []ToolEntry
	Agents           []AgentEntry
	Todos            []TodoItem
	SessionStart     time.Time
	SessionName      string
	SessionToolNames []string // all unique tool names across entire session (never reset)
}

type ToolEntry struct {
	ID, Name, Target, Status string
	StartTime, EndTime       time.Time
}

type AgentEntry struct {
	ID, Type, Model, Description, Status string
	StartTime, EndTime                   time.Time
}

type TodoItem struct {
	Content, Status string
}

func Parse(path string) (*Data, error) {
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

	toolMap := make(map[string]int)
	sessionToolLast := make(map[string]time.Time) // last usage time per tool name
	taskIDMap := make(map[string]int)
	nextTaskID := 1

	// Batch detection: 2+ consecutive TaskCreate calls indicate a new plan.
	// Record the confirmed batch start so old todos can be trimmed after parsing.
	var consecutiveCreates int
	var pendingBatchStart int
	confirmedBatchStart := -1

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		var entry jsonEntry
		if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
			continue
		}

		if !entry.Timestamp.IsZero() && data.SessionStart.IsZero() {
			data.SessionStart = entry.Timestamp
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
				switch block.Name {
				case "TaskCreate":
					if consecutiveCreates == 0 {
						pendingBatchStart = len(data.Todos)
					}
					consecutiveCreates++
					if consecutiveCreates == 2 {
						confirmedBatchStart = pendingBatchStart
					}
				case "TaskUpdate":
					// TaskUpdate means create phase is over; reset so
					// the next batch of TaskCreate is detected properly
					consecutiveCreates = 0
				case "TodoWrite":
					// Don't reset - TodoWrite replaces all todos inline
				default:
					consecutiveCreates = 0
				}
				handleToolUse(data, block, entry.Timestamp, toolMap, taskIDMap, &nextTaskID)
				if !managementTools[block.Name] {
					sessionToolLast[block.Name] = entry.Timestamp
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

	// If a confirmed batch (2+ consecutive creates) was found,
	// discard all older todos that preceded it
	if confirmedBatchStart > 0 {
		data.Todos = data.Todos[confirmedBatchStart:]
	}

	// Filter out deleted todos
	filtered := data.Todos[:0]
	for _, t := range data.Todos {
		if t.Status != "deleted" {
			filtered = append(filtered, t)
		}
	}
	data.Todos = filtered

	return data, nil
}

func handleToolUse(data *Data, block *contentBlock, ts time.Time, toolMap map[string]int, taskIDMap map[string]int, nextTaskID *int) {
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
	case "TodoWrite":
		data.Todos = data.Todos[:0]
		for _, t := range block.Input.Todos {
			data.Todos = append(data.Todos, TodoItem{Content: t.Content, Status: normalizeStatus(t.Status)})
		}
	case "TaskCreate":
		content := block.Input.Subject
		if content == "" {
			content = block.Input.Description
		}
		data.Todos = append(data.Todos, TodoItem{Content: content, Status: "pending"})
		id := block.Input.TaskID
		if id == "" {
			id = fmt.Sprintf("%d", *nextTaskID)
		}
		taskIDMap[id] = len(data.Todos) - 1
		*nextTaskID++
	case "TaskUpdate":
		if idx, ok := taskIDMap[block.Input.TaskID]; ok && idx < len(data.Todos) {
			st := normalizeStatus(block.Input.Status)
			// When a new task goes in_progress, auto-complete any
			// previously in_progress tasks (Claude often skips the
			// explicit completed event)
			if st == "in_progress" {
				for i := range data.Todos {
					if data.Todos[i].Status == "in_progress" {
						data.Todos[i].Status = "completed"
					}
				}
			}
			if block.Input.Status != "" {
				data.Todos[idx].Status = st
			}
			if block.Input.Subject != "" {
				data.Todos[idx].Content = block.Input.Subject
			}
		}
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

	for i := range data.Agents {
		if data.Agents[i].ID == block.ToolUseID {
			data.Agents[i].Status = "completed"
			data.Agents[i].EndTime = ts
			break
		}
	}
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

func normalizeStatus(s string) string {
	switch strings.ToLower(s) {
	case "pending", "not_started":
		return "pending"
	case "in_progress", "running":
		return "in_progress"
	case "completed", "complete", "done":
		return "completed"
	default:
		return s
	}
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
	"TaskCreate": true,
	"TaskUpdate": true,
	"TodoWrite":  true,
	"Task":       true,
	"Agent":      true,
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
	FilePath     string      `json:"file_path"`
	Path         string      `json:"path"`
	Pattern      string      `json:"pattern"`
	Command      string      `json:"command"`
	SubagentType string      `json:"subagent_type"`
	Model        string      `json:"model"`
	Description  string      `json:"description"`
	Subject      string      `json:"subject"`
	Status       string      `json:"status"`
	TaskID       string      `json:"taskId"`
	Todos        []todoInput `json:"todos"`
}

type todoInput struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}
