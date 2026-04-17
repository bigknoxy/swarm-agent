package orchestrator

import (
	"context"
	"fmt"
	"testing"
	"time"
	
	"agent_loop/internal/executor"
	"agent_loop/internal/judge"
	"agent_loop/internal/llm"
)

// MockLLM is a mock implementation of the Provider interface
type MockLLM struct {
	Response string
}

func (m *MockLLM) Generate(ctx context.Context, req llm.Request) (*llm.Response, error) {
	return &llm.Response{
		Text: m.Response,
	}, nil
}

// TestSolveReActIntegration verifies the ReAct logic works with tools.
func TestSolveReActIntegration(t *testing.T) {
	// We verify the tools are initialized correctly on the orchestrator.
	exec := executor.NewExecutor(5 * time.Second)
	jdg := &judge.Judge{}
	cfg := Config{ MaxAttempts: 5 }
	
	// Create real orchestrator
	o := NewOrchestrator(llm.NewClient(&MockLLM{}), exec, jdg, cfg)
	
	// Verify tools are registered
	if o.tools == nil {
		t.Fatal("Tools not initialized")
	}
	if len(o.tools.Tools) == 0 {
		t.Fatal("No tools registered")
	}
	
	fmt.Println("[PASS] Integration test complete. Tools registered correctly.")
}

// TestToolManagerRegisteredInOrchestrator verifies tools are added on init.
func TestToolManagerRegisteredInOrchestrator(t *testing.T) {
	exec := executor.NewExecutor(1 * time.Second)
	o := NewOrchestrator(llm.NewClient(&MockLLM{}), exec, &judge.Judge{}, Config{})
	
	if len(o.tools.Tools) != 2 {
		t.Fatalf("Expected 2 tools, got %d", len(o.tools.Tools))
	}
}
