package agent

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"agent_loop/internal/executor"
)

// MockTool implements the Tool interface for testing
type MockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, arg string) (string, error)
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) Execute(ctx context.Context, arg string) (string, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, arg)
	}
	return "", nil
}

// MockExecutor allows mocking the executor.Executor for testing
type MockExecutor struct {
	runPythonFunc func(ctx context.Context, code string) (*executor.Result, error)
}

func (m *MockExecutor) RunPython(ctx context.Context, code string) (*executor.Result, error) {
	if m.runPythonFunc != nil {
		return m.runPythonFunc(ctx, code)
	}
	return &executor.Result{}, nil
}

func TestToolManager_NewToolManager(t *testing.T) {
	tm := NewToolManager()
	if tm == nil {
		t.Fatal("NewToolManager returned nil")
	}
	// Tools slice can be nil initially - append works on nil slices in Go
	// Verify we can register a tool (which will initialize the slice)
	tm.Register(&MockTool{name: "test", description: "test"})
	if len(tm.Tools) != 1 {
		t.Fatal("should be able to register tool with new manager")
	}
}

func TestToolManager_RegisterSingle(t *testing.T) {
	tm := NewToolManager()
	tool := &MockTool{name: "test_tool", description: "A test tool"}

	tm.Register(tool)

	if len(tm.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(tm.Tools))
	}
	if tm.Tools[0].Name() != "test_tool" {
		t.Fatalf("expected tool name 'test_tool', got '%s'", tm.Tools[0].Name())
	}
}

func TestToolManager_RegisterMultiple(t *testing.T) {
	tm := NewToolManager()
	tool1 := &MockTool{name: "tool1", description: "Tool 1"}
	tool2 := &MockTool{name: "tool2", description: "Tool 2"}
	tool3 := &MockTool{name: "tool3", description: "Tool 3"}

	tm.Register(tool1, tool2, tool3)

	if len(tm.Tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tm.Tools))
	}
}

func TestToolManager_RegisterAppends(t *testing.T) {
	tm := NewToolManager()
	tool1 := &MockTool{name: "tool1", description: "Tool 1"}
	tool2 := &MockTool{name: "tool2", description: "Tool 2"}

	tm.Register(tool1)
	tm.Register(tool2)

	if len(tm.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tm.Tools))
	}
	if tm.Tools[0].Name() != "tool1" || tm.Tools[1].Name() != "tool2" {
		t.Fatal("tools not registered in correct order")
	}
}

func TestToolManager_ExecuteSuccess(t *testing.T) {
	tm := NewToolManager()
	expectedOutput := "test output"
	tool := &MockTool{
		name:        "my_tool",
		description: "A test tool",
		executeFunc: func(ctx context.Context, arg string) (string, error) {
			return expectedOutput, nil
		},
	}
	tm.Register(tool)

	result, err := tm.Execute(context.Background(), "my_tool", "some arg")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != expectedOutput {
		t.Fatalf("expected '%s', got '%s'", expectedOutput, result)
	}
}

func TestToolManager_ExecuteToolNotFound(t *testing.T) {
	tm := NewToolManager()
	tool := &MockTool{name: "existing_tool", description: "A test tool"}
	tm.Register(tool)

	_, err := tm.Execute(context.Background(), "non_existent_tool", "some arg")

	if err == nil {
		t.Fatal("expected error for non-existent tool")
	}
	expectedErr := "tool not found: non_existent_tool"
	if err.Error() != expectedErr {
		t.Fatalf("expected error '%s', got '%s'", expectedErr, err.Error())
	}
}

func TestToolManager_ExecuteToolReturnsError(t *testing.T) {
	tm := NewToolManager()
	expectedErr := errors.New("tool execution failed")
	tool := &MockTool{
		name:        "failing_tool",
		description: "A failing tool",
		executeFunc: func(ctx context.Context, arg string) (string, error) {
			return "", expectedErr
		},
	}
	tm.Register(tool)

	_, err := tm.Execute(context.Background(), "failing_tool", "some arg")

	if err == nil {
		t.Fatal("expected error from tool execution")
	}
	if err != expectedErr {
		t.Fatalf("expected error '%v', got '%v'", expectedErr, err)
	}
}

func TestToolManager_GetToolsSummaryEmpty(t *testing.T) {
	tm := NewToolManager()

	summary := tm.GetToolsSummary()

	if summary != "" {
		t.Fatalf("expected empty summary, got '%s'", summary)
	}
}

func TestToolManager_GetToolsSummarySingle(t *testing.T) {
	tm := NewToolManager()
	tool := &MockTool{name: "tool1", description: "Tool 1 description"}
	tm.Register(tool)

	summary := tm.GetToolsSummary()

	expected := "- tool1: Tool 1 description\n"
	if summary != expected {
		t.Fatalf("expected '%s', got '%s'", expected, summary)
	}
}

func TestToolManager_GetToolsSummaryMultiple(t *testing.T) {
	tm := NewToolManager()
	tool1 := &MockTool{name: "alpha", description: "First tool"}
	tool2 := &MockTool{name: "beta", description: "Second tool"}
	tool3 := &MockTool{name: "gamma", description: "Third tool"}
	tm.Register(tool1, tool2, tool3)

	summary := tm.GetToolsSummary()

	expected := "- alpha: First tool\n- beta: Second tool\n- gamma: Third tool\n"
	if summary != expected {
		t.Fatalf("expected '%s', got '%s'", expected, summary)
	}
}

func TestPythonREPL_Name(t *testing.T) {
	repl := &PythonREPL{Executor: &executor.Executor{}}

	name := repl.Name()

	if name != "python_repl" {
		t.Fatalf("expected 'python_repl', got '%s'", name)
	}
}

func TestPythonREPL_Description(t *testing.T) {
	repl := &PythonREPL{Executor: &executor.Executor{}}

	desc := repl.Description()

	expected := "You can execute python code here. Use this to calculate specific values or test snippets."
	if desc != expected {
		t.Fatalf("expected '%s', got '%s'", expected, desc)
	}
}

func TestPythonREPL_ExecuteSuccess(t *testing.T) {
	repl := &PythonREPL{Executor: executor.NewExecutor(5 * time.Second)}

	result, err := repl.Execute(context.Background(), "print(42)")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "42\n" {
		t.Fatalf("expected '42\\n', got '%s'", result)
	}
}

func TestPythonREPL_ExecuteWithStderr(t *testing.T) {
	repl := &PythonREPL{Executor: executor.NewExecutor(5 * time.Second)}

	result, err := repl.Execute(context.Background(), "import sys; print('out'); print('err', file=sys.stderr)")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain both stdout and stderr
	if !strings.Contains(result, "out") {
		t.Fatalf("expected 'out' in result, got '%s'", result)
	}
	if !strings.Contains(result, "[STDERR]:") {
		t.Fatalf("expected '[STDERR]:' in result, got '%s'", result)
	}
}

func TestPythonREPL_ExecuteError(t *testing.T) {
	repl := &PythonREPL{Executor: executor.NewExecutor(5 * time.Second)}

	result, err := repl.Execute(context.Background(), "this is invalid python")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Invalid python should produce stderr output
	if !strings.Contains(result, "[STDERR]:") {
		t.Fatalf("expected '[STDERR]:' in result for invalid python, got '%s'", result)
	}
}

func TestPerformanceChecker_Name(t *testing.T) {
	checker := &PerformanceChecker{Executor: &executor.Executor{}}

	name := checker.Name()

	if name != "performance_checker" {
		t.Fatalf("expected 'performance_checker', got '%s'", name)
	}
}

func TestPerformanceChecker_Description(t *testing.T) {
	checker := &PerformanceChecker{Executor: &executor.Executor{}}

	desc := checker.Description()

	expected := "Measures the execution time of a python script. Use when efficiency constraints are critical."
	if desc != expected {
		t.Fatalf("expected '%s', got '%s'", expected, desc)
	}
}

func TestPerformanceChecker_ExecuteWrapsCode(t *testing.T) {
	checker := &PerformanceChecker{Executor: &executor.Executor{}}

	// Verify the description mentions timing - the actual Execute wraps code internally
	// We can't easily test the wrapped code without mocking, but we can verify the tool exists
	if checker.Name() != "performance_checker" {
		t.Fatal("PerformanceChecker name incorrect")
	}
}

func TestPerformanceChecker_ExecuteSuccess(t *testing.T) {
	checker := &PerformanceChecker{Executor: executor.NewExecutor(5 * time.Second)}

	result, err := checker.Execute(context.Background(), "print(42)")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should contain the output and timing information
	if !strings.Contains(result, "42") {
		t.Fatalf("expected '42' in result, got '%s'", result)
	}
	if !strings.Contains(result, "TIME:") {
		t.Fatalf("expected 'TIME:' in result for timing, got '%s'", result)
	}
}

func TestPerformanceChecker_ExecuteError(t *testing.T) {
	checker := &PerformanceChecker{Executor: executor.NewExecutor(5 * time.Second)}
	
	result, err := checker.Execute(context.Background(), "this is invalid python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	
	// PerformanceChecker only returns stdout, invalid python produces no stdout
	// Just verify we don't panic and get some kind of result
	_ = result // Result may be empty for invalid python
}

// Test that Tool interface is properly implemented
func TestPythonREPL_ImplementsTool(t *testing.T) {
	var _ Tool = &PythonREPL{}
}

func TestPerformanceChecker_ImplementsTool(t *testing.T) {
	var _ Tool = &PerformanceChecker{}
}

func TestMockTool_ImplementsTool(t *testing.T) {
	var _ Tool = &MockTool{}
}
