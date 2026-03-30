package tui

import (
	"fmt"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"ai_code/internal/usecase"
)

func TestKeyDownScrollDoesNotJumpToBottom(t *testing.T) {
	m := newScrollTestModel()
	m.scrollOffset = 2
	m.followBottom = false

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyDown})
	model := updated.(*Model)

	if model.scrollOffset != 3 {
		t.Fatalf("scrollOffset = %d, want 3", model.scrollOffset)
	}
	if model.followBottom {
		t.Fatal("followBottom = true, want false")
	}

	view := model.renderMessagesAreaWithBottom(4)
	if strings.Contains(view, "line 19") {
		t.Fatalf("view unexpectedly jumped to bottom: %q", view)
	}
}

func TestMouseWheelDownDoesNotJumpToBottom(t *testing.T) {
	m := newScrollTestModel()
	m.scrollOffset = 2
	m.followBottom = false

	updated, _ := m.Update(tea.MouseMsg{Type: tea.MouseWheelDown})
	model := updated.(*Model)

	if model.scrollOffset != 5 {
		t.Fatalf("scrollOffset = %d, want 5", model.scrollOffset)
	}
	if model.followBottom {
		t.Fatal("followBottom = true, want false")
	}

	view := model.renderMessagesAreaWithBottom(4)
	if strings.Contains(view, "line 19") {
		t.Fatalf("view unexpectedly jumped to bottom: %q", view)
	}
}

func newScrollTestModel() *Model {
	messages := make([]UIMessage, 0, 20)
	for i := range 20 {
		messages = append(messages, UIMessage{
			Type:    usecase.OutputResult,
			Content: fmt.Sprintf("line %02d", i),
		})
	}

	return &Model{
		state:    StateInput,
		height:   12,
		messages: messages,
		styles:   Styles{},
	}
}
