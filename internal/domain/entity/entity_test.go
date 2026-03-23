package entity_test

import (
	"testing"
	"time"

	"ai_code/internal/domain/entity"
)

func TestNewMessage(t *testing.T) {
	msg := entity.NewMessage(entity.RoleUser, "Hello")

	if msg.Role != entity.RoleUser {
		t.Errorf("expected role %s, got %s", entity.RoleUser, msg.Role)
	}
	if msg.Content != "Hello" {
		t.Errorf("expected content 'Hello', got '%s'", msg.Content)
	}
	if msg.ID == "" {
		t.Error("expected ID to be generated")
	}
	if msg.Timestamp.IsZero() {
		t.Error("expected timestamp to be set")
	}
}

func TestMessageWithToolCalls(t *testing.T) {
	msg := entity.NewMessage(entity.RoleAssistant, "Response")
	toolCalls := []entity.ToolCall{
		entity.NewToolCall("bash", "ls -la"),
	}

	msg = msg.WithToolCalls(toolCalls)

	if len(msg.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(msg.ToolCalls))
	}
	if msg.ToolCalls[0].GetName() != "bash" {
		t.Errorf("expected tool name 'bash', got '%s'", msg.ToolCalls[0].GetName())
	}
}

func TestNewSession(t *testing.T) {
	session := entity.NewSession("test-model", "test-provider")

	if session.Model != "test-model" {
		t.Errorf("expected model 'test-model', got '%s'", session.Model)
	}
	if session.Provider != "test-provider" {
		t.Errorf("expected provider 'test-provider', got '%s'", session.Provider)
	}
	if len(session.Messages) != 0 {
		t.Errorf("expected 0 messages, got %d", len(session.Messages))
	}
}

func TestSessionAddMessage(t *testing.T) {
	session := entity.NewSession("test-model", "test-provider")
	msg := entity.NewMessage(entity.RoleUser, "Hello")

	session.AddMessage(msg)

	if len(session.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(session.Messages))
	}

	// Check UpdatedAt changed
	oldUpdatedAt := session.UpdatedAt
	time.Sleep(time.Millisecond)
	session.AddMessage(entity.NewMessage(entity.RoleUser, "World"))
	if !session.UpdatedAt.After(oldUpdatedAt) {
		t.Error("expected UpdatedAt to be updated")
	}
}

func TestSessionClear(t *testing.T) {
	session := entity.NewSession("test-model", "test-provider")
	session.AddMessage(entity.NewMessage(entity.RoleUser, "Hello"))

	session.Clear()

	if len(session.Messages) != 0 {
		t.Errorf("expected 0 messages after clear, got %d", len(session.Messages))
	}
}

func TestSessionLastMessage(t *testing.T) {
	session := entity.NewSession("test-model", "test-provider")

	// Empty session
	if session.LastMessage() != nil {
		t.Error("expected nil for empty session")
	}

	// With messages
	msg1 := entity.NewMessage(entity.RoleUser, "Hello")
	msg2 := entity.NewMessage(entity.RoleAssistant, "Hi there")
	session.AddMessage(msg1)
	session.AddMessage(msg2)

	lastMsg := session.LastMessage()
	if lastMsg.Content != "Hi there" {
		t.Errorf("expected last message content 'Hi there', got '%s'", lastMsg.Content)
	}
}

func TestNewToolCall(t *testing.T) {
	call := entity.NewToolCall("bash", "ls -la")

	if call.GetName() != "bash" {
		t.Errorf("expected name 'bash', got '%s'", call.GetName())
	}
	if call.GetArguments() != "ls -la" {
		t.Errorf("expected arguments 'ls -la', got '%s'", call.GetArguments())
	}
	if call.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", call.Status)
	}
}

func TestToolCallWithResult(t *testing.T) {
	call := entity.NewToolCall("bash", "ls -la")
	call = call.WithResult("file1.txt\nfile2.txt", "success")

	if call.Result != "file1.txt\nfile2.txt" {
		t.Errorf("expected result, got '%s'", call.Result)
	}
	if call.Status != "success" {
		t.Errorf("expected status 'success', got '%s'", call.Status)
	}
}