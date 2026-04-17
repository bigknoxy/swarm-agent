package main

import (
	"context"
	"fmt"
	"time"

	"agent_loop/internal/agent"
	"agent_loop/internal/executor"
	"agent_loop/internal/judge"
	"agent_loop/internal/llm"
	"agent_loop/internal/orchestrator"
)

func main() {
	// Setup Ollama Provider
	provider := llm.NewOllamaProvider("http://localhost:11434/v1", "qwen2.5-coder:32b")
	client := llm.NewClient(provider)
	
	// Setup Executor (increased for long-running TDD tests)
	exec := executor.NewExecutor(5 * time.Minute)
	
	// Setup Judge
	j := judge.NewJudge()
	
	// Configuration for the Agentic Loop
	cfg := orchestrator.Config{
		MaxAttempts: 5,
		Timeout:     10 * time.Minute,
	}
	
	// Create Orchestrator (which now holds the ToolManager internally)
	orch := orchestrator.NewOrchestrator(client, exec, j, cfg)
	
	// Register Tools: This is how we "extend" the agent's capabilities
	// These tools are injected into the prompt so the agent knows about them.
	orch.RegisterTools(&agent.PythonREPL{Executor: exec}, &agent.PerformanceChecker{Executor: exec})
	
	// The goal: The agent must use its new tools to solve this efficiently.
	goal := `Write a Python function called 'slow_fibonacci(n)' that calculates Fibonacci numbers using the most inefficient recursive method possible (no memoization). 
	Then, write a test that ensures it works for n=10, but the TDD agent must eventually realize that for n=40 it will timeout. 
	The goal is for the agent to implement a 'Timeout-aware' solution (using memoization or iteration) after realizing it cannot satisfy the time constraint.
	Ensure the solution handles limit=1,000,000 within 5 seconds.`
	
	fmt.Printf("Starting Agentic Loop (SolveReAct)...\nGoal: %s\n", goal)
	
	// --- Switch from TDD to Agentic Mode ---
	result, err := orch.SolveReAct(context.Background(), goal)
	if err != nil {
		fmt.Printf("\n❌ Failed to solve: %v\n", err)
		return
	}
	
	fmt.Printf("\n✅ Success! Solution:\n%s\n", result)
}
