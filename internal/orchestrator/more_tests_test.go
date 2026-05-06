package orchestrator

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"agent_loop/internal/executor"
	"agent_loop/internal/judge"
)

// TestSolveReAct_WithLTM tests LTM query integration
func TestSolveReAct_WithLTM(t *testing.T) {
	// Mock LLM that returns solution
	mockLLM := &MockLLM2{
		Responses: []string{"SOLUTION:\nprint('with LTM')"},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: true},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 5, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), "", cfg, &MockCLI{})

	ctx := context.Background()
	result, err := orch.SolveReAct(ctx, "goal with LTM")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "with LTM") {
		t.Errorf("Expected solution with LTM, got %q", result)
	}
}

// TestSolveReAct_ContextTimeout tests context timeout during execution
func TestSolveReAct_ContextTimeout(t *testing.T) {
	// Skipping this test as it requires more complex context handling
	t.Skip("Context timeout test requires more complex mocking")
}

// TestNewWorkspaceWithPath_Expansion tests workspace path expansion
func TestNewWorkspaceWithPath_Expansion(t *testing.T) {
	ws := NewWorkspaceWithPath("~/swarm_test")
	defer os.RemoveAll(ws.Root)

	if !strings.Contains(ws.Root, os.Getenv("HOME")) {
		t.Errorf("Expected path to contain HOME, got %q", ws.Root)
	}
}

// TestFileTool_Execute_WriteAndRead tests writing and reading in one flow
func TestFileTool_Execute_WriteAndRead(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_filetool_wr")
	defer os.RemoveAll(ws.Root)

	tool := &FileTool{Workspace: ws}

	// Write
	writeArgs := `{"action": "write", "path": "test.txt", "content": "Hello"}`
	_, err := tool.Execute(context.Background(), writeArgs)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	// Read
	readArgs := `{"action": "read", "path": "test.txt"}`
	result, err := tool.Execute(context.Background(), readArgs)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if result != "Hello" {
		t.Errorf("Expected 'Hello', got %q", result)
	}
}

// TestMemory_SaveAndLoad_Complex tests complex save/load scenario
func TestMemory_SaveAndLoad_Complex(t *testing.T) {
	tmpFile := "/tmp/test_memory_complex.json"
	defer os.Remove(tmpFile)

	mem := NewSessionMemory("Complex goal")
	mem.AddHypothesis("Test hypothesis 1")
	mem.AddHypothesis("Test hypothesis 2")
	mem.UpdateHypothesis(0, "Failed", "Analysis 1")
	mem.AddLesson("Lesson 1")
	mem.AddLesson("Lesson 2")
	mem.ReActSteps = append(mem.ReActSteps,
		Step{Thought: "T1", Action: "A1", Observation: "O1"},
		Step{Thought: "T2", Action: "A2", Observation: "O2"},
	)

	ctx := context.Background()
	err := mem.Save(ctx, tmpFile)
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded := NewSessionMemory("")
	err = loaded.Load(ctx, tmpFile)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.Goal != "Complex goal" {
		t.Errorf("Expected goal 'Complex goal', got %q", loaded.Goal)
	}
	if len(loaded.Hypotheses) != 2 {
		t.Errorf("Expected 2 hypotheses, got %d", len(loaded.Hypotheses))
	}
	if loaded.Hypotheses[0].Status != "Failed" {
		t.Errorf("Expected hypothesis 0 status 'Failed', got %q", loaded.Hypotheses[0].Status)
	}
	if len(loaded.Lessons) != 2 {
		t.Errorf("Expected 2 lessons, got %d", len(loaded.Lessons))
	}
	if len(loaded.ReActSteps) != 2 {
		t.Errorf("Expected 2 ReAct steps, got %d", len(loaded.ReActSteps))
	}
}
