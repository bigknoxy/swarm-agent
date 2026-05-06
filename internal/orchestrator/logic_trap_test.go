package orchestrator

import (
	"context"
	"testing"
	"time"

	"agent_loop/internal/executor"
	"agent_loop/internal/judge"
)

// TestLogicTrap_PerformanceFailure tests when judge says code is too slow
func TestLogicTrap_PerformanceFailure(t *testing.T) {
	// Mock LLM returns perfect-looking code
	mockLLM := &MockLLM2{
		Responses: []string{
			"SOLUTION:\ndef slow_fibonacci(n):\n    if n <= 1: return n\n    return slow_fibonacci(n-1) + slow_fibonacci(n-2)",
		},
	}

	// MockJudge (from orchestrator_react_test.go) returns PERFORMANCE fault
	mockJudge := &MockJudge{}

	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 3, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), "", cfg, &MockCLI{})

	ctx := context.Background()
	_, err := orch.SolveReAct(ctx, "fibonacci")

	// Should fail after max attempts because judge always says it's too slow
	if err == nil {
		t.Fatal("Expected error due to performance failure")
	}
}

// TestLogicTrap_JudgeRejection tests when judge rejects the solution
func TestLogicTrap_JudgeRejection(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{
			"SOLUTION:\ndef add(a, b):\n    return a - b  # Wrong logic",
		},
	}

	// MockJudge2 with LOGIC fault
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{
			IsCorrect: false,
			Feedback:  "Logic error: subtraction instead of addition",
			Fault:     "LOGIC",
		},
	}

	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 2, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), "", cfg, &MockCLI{})

	ctx := context.Background()
	_, err := orch.SolveReAct(ctx, "add function")

	if err == nil {
		t.Fatal("Expected error due to judge rejection")
	}
}

// TestTimeout_Handling tests that timeout is handled gracefully
func TestTimeout_Handling(t *testing.T) {
	// Mock LLM returns code that sleeps
	mockLLM := &MockLLM2{
		Responses: []string{
			"SOLUTION:\nimport time\nprint('starting')\ntime.sleep(100)\nprint('done')",
		},
	}

	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: true},
	}

	// Very short timeout
	exec := executor.NewExecutor(100 * time.Millisecond)
	cfg := Config{MaxAttempts: 1, Timeout: 100 * time.Millisecond}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), "", cfg, &MockCLI{})

	ctx := context.Background()
	result, err := orch.SolveReAct(ctx, "goal")

	// Should return the solution even if verification times out
	// Just verify it doesn't panic
	_ = result
	_ = err
}

// TestTimeout_MaxAttemptsReached tests behavior when max attempts is reached
func TestTimeout_MaxAttemptsReached(t *testing.T) {
	// LLM never returns SOLUTION
	mockLLM := &MockLLM2{
		Responses: []string{"Thought: still thinking..."},
	}

	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: false},
	}

	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 2, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), "", cfg, &MockCLI{})

	ctx := context.Background()
	start := time.Now()
	_, err := orch.SolveReAct(ctx, "goal")
	elapsed := time.Since(start)

	if err == nil {
		t.Fatal("Expected error after max attempts")
	}

	// Should not take too long (each attempt is quick since no real work is done)
	if elapsed > 5*time.Second {
		t.Errorf("Test took too long: %v", elapsed)
	}
}
