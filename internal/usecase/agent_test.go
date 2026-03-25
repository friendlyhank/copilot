package usecase

import (
	"testing"

	"ai_code/internal/domain/entity"
)

func TestAgentInjectTodoReminder(t *testing.T) {
	agent := &Agent{
		todoNagAfter: 3,
	}
	results := []entity.ToolResult{
		{ToolCallID: "tool-1", Content: "result body"},
	}

	results = agent.injectTodoReminder(results, false)
	results = agent.injectTodoReminder(results, false)

	if results[0].Content != "result body" {
		t.Fatalf("expected no reminder before threshold, got %q", results[0].Content)
	}

	results = agent.injectTodoReminder(results, false)

	expected := "<reminder>Update your todos.</reminder>\nresult body"
	if results[0].Content != expected {
		t.Fatalf("unexpected reminder content: %q", results[0].Content)
	}

	results = agent.injectTodoReminder(results, true)

	if agent.todoRounds != 0 {
		t.Fatalf("expected todoRounds to reset, got %d", agent.todoRounds)
	}
}
