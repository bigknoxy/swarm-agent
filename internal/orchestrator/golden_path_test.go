package orchestrator

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent_loop/internal/executor"
	"agent_loop/internal/judge"
)

// TestGoldenPath_SolutionOnFirstTry tests the happy path where LLM returns a solution immediately
func TestGoldenPath_SolutionOnFirstTry(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{"SOLUTION:\nprint('hello world')"},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: true, Feedback: "Verified!"},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 5, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), "", cfg, &MockCLI{})

	ctx := context.Background()
	result, err := orch.SolveReAct(ctx, "print hello")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "print('hello world')") {
		t.Errorf("Expected solution in result, got %q", result)
	}
}

// TestGoldenPath_MaxAttempts tests when LLM never returns valid solution
func TestGoldenPath_MaxAttempts(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{"Thought: thinking..."},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: false},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 3, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), "", cfg, &MockCLI{})

	ctx := context.Background()
	_, err := orch.SolveReAct(ctx, "goal")

	if err == nil {
		t.Fatal("Expected error after max attempts")
	}
	if !strings.Contains(err.Error(), "failed to converge") {
		t.Errorf("Expected 'failed to converge' error, got %v", err)
	}
}

// TestGoldenPath_ContextCancelled tests handling of context cancellation
func TestGoldenPath_ContextCancelled(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{"Thought: thinking..."},
	}
	mockJudge := &MockJudge2{}
	exec := executor.NewExecutor(5 * time.Second)
	cfg := Config{MaxAttempts: 10, Timeout: 5 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), "", cfg, &MockCLI{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := orch.SolveReAct(ctx, "goal")

	if err == nil {
		t.Fatal("Expected error due to cancelled context")
	}
}

// TestGoldenPath_WithToolCall tests ReAct loop with tool usage
func TestGoldenPath_WithToolCall(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{
			"Thought: I need to check something\nAction: python_repl\nAction Input: print(1+1)",
			"SOLUTION:\nprint('done')",
		},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: true},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 5, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), "", cfg, &MockCLI{})

	ctx := context.Background()
	result, err := orch.SolveReAct(ctx, "goal")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "print('done')") {
		t.Errorf("Expected solution in result, got %q", result)
	}
	// Verify LLM was called at least twice (once for tool, once for solution)
	if mockLLM.CallCount < 2 {
		t.Errorf("Expected at least 2 LLM calls, got %d", mockLLM.CallCount)
	}
}
