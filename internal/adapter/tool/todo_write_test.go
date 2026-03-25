package tool

import (
	"context"
	"strings"
	"testing"
)

func TestTodoWriteToolExecuteRendersTodos(t *testing.T) {
	tool := NewTodoWriteTool()

	output, err := tool.Execute(context.Background(), `{"todos":[{"id":"1","content":"调查项目结构","status":"completed"},{"id":"2","content":"实现 todo_write","status":"in_progress"},{"id":"3","content":"运行测试","status":"pending"}]}`)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	expected := []string{
		"[x] #1: 调查项目结构",
		"[>] #2: 实现 todo_write",
		"[ ] #3: 运行测试",
		"(1/3 completed)",
	}

	for _, part := range expected {
		if !strings.Contains(output, part) {
			t.Fatalf("expected output to contain %q, got %q", part, output)
		}
	}
}

func TestTodoWriteToolRejectsMultipleInProgress(t *testing.T) {
	tool := NewTodoWriteTool()

	_, err := tool.Execute(context.Background(), `{"todos":[{"id":"1","content":"第一步","status":"in_progress"},{"id":"2","content":"第二步","status":"in_progress"}]}`)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestTodoWriteToolSupportsItemsAlias(t *testing.T) {
	tool := NewTodoWriteTool()

	output, err := tool.Execute(context.Background(), `{"items":[{"id":"1","text":"读取参考实现","status":"completed"}]}`)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !strings.Contains(output, "[x] #1: 读取参考实现") {
		t.Fatalf("unexpected output: %q", output)
	}
}
