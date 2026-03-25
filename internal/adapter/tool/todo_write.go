package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"ai_code/internal/domain/errors"
	"ai_code/pkg/logger"
)

type TodoWriteTool struct {
	mu     sync.RWMutex
	items  []todoItem
	logger logger.Logger
}

type todoItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Status  string `json:"status"`
}

type todoWriteInput struct {
	Todos []todoWriteItem `json:"todos"`
	Items []todoWriteItem `json:"items"`
}

type todoWriteItem struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	Text    string `json:"text"`
	Status  string `json:"status"`
}

func NewTodoWriteTool() *TodoWriteTool {
	return &TodoWriteTool{
		logger: logger.Default().WithPrefix("todo_write"),
	}
}

func (t *TodoWriteTool) Name() string {
	return "todo_write"
}

func (t *TodoWriteTool) Description() string {
	return "Update task list. Track progress on multi-step tasks."
}

func (t *TodoWriteTool) Parameters() map[string]interface{} {
	itemSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"id": map[string]interface{}{
				"type":        "string",
				"description": "The unique identifier of the todo item",
			},
			"content": map[string]interface{}{
				"type":        "string",
				"description": "The task content",
			},
			"text": map[string]interface{}{
				"type":        "string",
				"description": "Alias of content for compatibility",
			},
			"status": map[string]interface{}{
				"type":        "string",
				"description": "The task status",
				"enum":        []string{"pending", "in_progress", "completed"},
			},
		},
		"required": []string{"status"},
	}

	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"todos": map[string]interface{}{
				"type":        "array",
				"description": "The full todo list to persist",
				"items":       itemSchema,
			},
			"items": map[string]interface{}{
				"type":        "array",
				"description": "Alias of todos for compatibility",
				"items":       itemSchema,
			},
		},
		"oneOf": []map[string]interface{}{
			{"required": []string{"todos"}},
			{"required": []string{"items"}},
		},
		"additionalProperties": false,
	}
}

func (t *TodoWriteTool) Execute(ctx context.Context, args string) (string, error) {
	var input todoWriteInput
	if err := json.Unmarshal([]byte(args), &input); err != nil {
		return "", errors.Wrap(errors.CodeInvalidInput, "failed to parse arguments", err)
	}

	items := input.Todos
	if items == nil {
		items = input.Items
	}

	rendered, err := t.replace(items)
	if err != nil {
		return "", err
	}

	t.logger.Debug("todo list updated", logger.F("count", len(items)))
	return rendered, nil
}

func (t *TodoWriteTool) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.items = nil
}

func (t *TodoWriteTool) replace(items []todoWriteItem) (string, error) {
	if len(items) > 20 {
		return "", errors.New(errors.CodeInvalidInput, "max 20 todos allowed")
	}

	validated := make([]todoItem, 0, len(items))
	inProgressCount := 0
	seen := make(map[string]struct{}, len(items))

	for i, item := range items {
		content := item.Content
		if content == "" {
			content = item.Text
		}
		status := item.Status
		if status == "" {
			status = "pending"
		}
		id := item.ID
		if id == "" {
			id = fmt.Sprintf("%d", i+1)
		}

		if content == "" {
			return "", errors.New(errors.CodeInvalidInput, "todo content required")
		}
		if status != "pending" && status != "in_progress" && status != "completed" {
			return "", errors.New(errors.CodeInvalidInput, "invalid todo status: "+status)
		}
		if _, exists := seen[id]; exists {
			return "", errors.New(errors.CodeInvalidInput, "duplicate todo id: "+id)
		}
		if status == "in_progress" {
			inProgressCount++
		}

		seen[id] = struct{}{}
		validated = append(validated, todoItem{
			ID:      id,
			Content: content,
			Status:  status,
		})
	}

	if inProgressCount > 1 {
		return "", errors.New(errors.CodeInvalidInput, "only one todo can be in_progress at a time")
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	t.items = validated

	return t.renderLocked(), nil
}

func (t *TodoWriteTool) renderLocked() string {
	if len(t.items) == 0 {
		return "No todos."
	}

	done := 0
	lines := make([]string, 0, len(t.items)+1)
	for _, item := range t.items {
		marker := "[ ]"
		if item.Status == "in_progress" {
			marker = "[>]"
		}
		if item.Status == "completed" {
			marker = "[x]"
			done++
		}
		lines = append(lines, fmt.Sprintf("%s #%s: %s", marker, item.ID, item.Content))
	}

	lines = append(lines, fmt.Sprintf("\n(%d/%d completed)", done, len(t.items)))
	return joinLines(lines)
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}

	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n" + lines[i]
	}
	return result
}
