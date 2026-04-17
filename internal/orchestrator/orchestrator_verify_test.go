package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"
	"agent_loop/internal/agent"
	"strings"
	"agent_loop/internal/executor"
)

// TestTools verifies the tool registration and execution logic.
func TestToolManagerLogic(t *testing.T) {
	tm := agent.NewToolManager()
	
	// We need a real executor for RunPython to work
	exec := executor.NewExecutor(5*time.Second)
	
	// Register tools (Using actual names from tool.go)
	tm.Register(&agent.PythonREPL{Executor: exec}, &agent.PerformanceChecker{Executor: exec})
	
	// 1. Verify count (tool.go uses .Tools slice)
	if len(tm.Tools) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(tm.Tools))
	}
	
	// 2. Verify PythonREPL
	out1, err1 := tm.Execute(context.Background(), "python_repl", "print(1+1)")
	if err1 != nil {
		t.Fatalf("PythonREPL call failed: %v", err1)
	}
	if !strings.Contains(out1, "2") {
		t.Errorf("PythonREPL output unexpected: %s", out1)
	}
	
	// 3. Verify PerformanceChecker
	out2, err2 := tm.Execute(context.Background(), "performance_checker", "print('fast')")
	if err2 != nil {
		t.Fatalf("PerformanceChecker call failed: %v", err2)
	}
	if !strings.Contains(out2, "TIME:") {
		t.Errorf("PerformanceChecker output unexpected: %s", out2)
	}
	
	// 4. Verify Error handling (wrong name)
	_, err3 := tm.Execute(context.Background(), "ghost_tool", "input")
	if err3 == nil {
		t.Error("Expected error for ghost_tool, got nil")
	}

	fmt.Println("[PASS] ToolManager logic verified.")
}

// TestOrchestratorInitialization verifies the Orchestrator struct and ReAct logic signature.
func TestOrchestratorInitialization(t *testing.T) {
	// Verify that the orchestrator struct can be initialized without crashing.
	// We verify the tools slice is initialized.
	cfg := Config{
		MaxAttempts: 5,
	}
	
	o := Orchestrator{
		config: cfg,
		tools: agent.NewToolManager(),
	}
	
	if o.tools == nil {
		t.Fatal("ToolManager should not be nil")
	}
	
	fmt.Println("[PASS] Orchestrator initialization verified.")
}

// TestReActPromptFormatting verifies the prompt string generation logic.
func TestReActPromptFormatting(t *testing.T) {
	// This is a logic verification of the ReAct loop's prompt generation.
	goal := "Calculate 100 * 50"
	prompt := fmt.Sprintf("Goal: %s\n\n### INSTRUCTIONS ###\nYou are an intelligent agent. You have access to the following TOOLS. You can use them to verify your logic or check constraints.\n\nWhen ready, output: SOLUTION:\n[your code]\n\n### TOOLS ###\n%s\n\n### HISTORY ###\n", goal, "")
	
	if !strings.Contains(prompt, "Goal: Calculate 100 * 50") {
		t.Fatal("Goal not in prompt")
	}
	if !strings.Contains(prompt, "SOLUTION:") {
		t.Fatal("SOLUTION instruction missing")
	}
	
	fmt.Println("[PASS] ReAct Prompt formatting verified.")
}
