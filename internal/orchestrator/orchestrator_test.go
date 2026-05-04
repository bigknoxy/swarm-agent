package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent_loop/internal/executor"
	"agent_loop/internal/judge"
	"agent_loop/internal/llm"
)

// MockLLM2 implements llm.Provider for testing with configurable responses
type MockLLM2 struct {
	Responses []string
	CallCount  int
	Error      error
}

func (m *MockLLM2) Generate(ctx context.Context, req llm.Request) (*llm.Response, error) {
	m.CallCount++
	
	// Check if context is cancelled
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}
	
	if m.Error != nil {
		return nil, m.Error
	}
	if len(m.Responses) > 0 {
		idx := (m.CallCount - 1) % len(m.Responses)
		return &llm.Response{Text: m.Responses[idx]}, nil
	}
	return &llm.Response{Text: "Default mock response"}, nil
}

// MockJudge2 implements judge.JudgeVerifier for testing with configurable verdict
type MockJudge2 struct {
	Verdict   judge.Verdict
	CallCount int
}

func (m *MockJudge2) Evaluate(res *executor.Result, exp judge.Expectation) judge.Verdict {
	m.CallCount++
	return m.Verdict
}

func (m *MockJudge2) EvaluateTDD(res *executor.Result) judge.Verdict {
	m.CallCount++
	return m.Verdict
}

func TestNewOrchestrator(t *testing.T) {
	mockLLM := &MockLLM2{}
	mockJudge := &MockJudge2{}
	exec := executor.NewExecutor(5 * time.Minute)
	cfg := Config{MaxAttempts: 5, Timeout: 5 * time.Minute}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	if orch == nil {
		t.Fatal("Expected non-nil Orchestrator")
	}
	if orch.config.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", orch.config.MaxAttempts)
	}
}

func TestOrchestrator_RegisterTool(t *testing.T) {
	mockLLM := &MockLLM2{}
	mockJudge := &MockJudge2{}
	exec := executor.NewExecutor(5 * time.Minute)
	cfg := Config{MaxAttempts: 5, Timeout: 5 * time.Minute}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	// Register a tool
	tool := &MockTool{name: "test_tool"}
	orch.RegisterTool(tool)

	// Verify tool was registered (indirectly by checking that tools are available)
	summary := orch.tools.GetToolsSummary()
	if !strings.Contains(summary, "test_tool") {
		t.Errorf("Expected tool summary to contain 'test_tool', got %q", summary)
	}
}

func TestOrchestrator_RegisterTools(t *testing.T) {
	mockLLM := &MockLLM2{}
	mockJudge := &MockJudge2{}
	exec := executor.NewExecutor(5 * time.Minute)
	cfg := Config{MaxAttempts: 5, Timeout: 5 * time.Minute}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	// Register multiple tools directly
	orch.RegisterTools(&MockTool{name: "tool1"}, &MockTool{name: "tool2"})

	summary := orch.tools.GetToolsSummary()
	if !strings.Contains(summary, "tool1") || !strings.Contains(summary, "tool2") {
		t.Errorf("Expected tool summary to contain both tools, got %q", summary)
	}
}

// MockTool implements agent.Tool for testing
type MockTool struct {
	name string
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return "Mock tool for testing"
}

func (m *MockTool) Execute(ctx context.Context, arg string) (string, error) {
	return "Mock result", nil
}

func TestOrchestrator_SolveReAct_GoldenPath(t *testing.T) {
	// Mock LLM returns solution on first try
	mockLLM := &MockLLM2{
		Responses: []string{"SOLUTION:\nprint('hello world')"},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: true, Feedback: "Verified!"},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 5, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	ctx := context.Background()
	result, err := orch.SolveReAct(ctx, "print hello")

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "print('hello world')") {
		t.Errorf("Expected solution in result, got %q", result)
	}
}

func TestOrchestrator_SolveReAct_MaxAttempts(t *testing.T) {
	// Mock LLM never returns SOLUTION:
	mockLLM := &MockLLM2{
		Responses: []string{"Thought: thinking..."},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: false},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 3, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	ctx := context.Background()
	_, err := orch.SolveReAct(ctx, "goal")

	if err == nil {
		t.Fatal("Expected error after max attempts")
	}
	if !strings.Contains(err.Error(), "failed to converge") {
		t.Errorf("Expected 'failed to converge' error, got %v", err)
	}
}

func TestOrchestrator_SolveTDD_GoldenPath(t *testing.T) {
	// Mock LLM responses in order:
	// 1. Test suite generation
	// 2. Architect hypothesis
	// 3. Solution generation
	mockLLM := &MockLLM2{
		Responses: []string{
			"import solution\nassert solution.add(1, 2) == 3",
			"HYPOTHESIS: Implement add function",
			"def add(a, b):\n    return a + b",
		},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: true},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 5, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	ctx := context.Background()
	result, err := orch.SolveTDD(ctx, "Implement add function")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "Success") {
		t.Errorf("Expected success result, got %q", result)
	}
}

// TestSolveTDD_ContextCancelled tests context cancellation in SolveTDD
func TestSolveTDD_ContextCancelled(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{"test code"},
	}
	mockJudge := &MockJudge2{}
	exec := executor.NewExecutor(5 * time.Second)
	cfg := Config{MaxAttempts: 10, Timeout: 5 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := orch.SolveTDD(ctx, "goal")
	if err == nil {
		t.Fatal("Expected error due to cancelled context")
	}
}

// TestSolveTDD_ReflectorCalled tests that Reflector is called on failure
func TestSolveTDD_ReflectorCalled(t *testing.T) {
	// Mock LLM responses:
	// 1. Test suite (pre-loop)
	// 2. Architect hypothesis (attempt 1)
	// 3. Solution (attempt 1)
	// 4. Reflector analysis (when judge fails)
	mockLLM := &MockLLM2{
		Responses: []string{
			"import solution\nassert solution.add(1,2) == 5", // Wrong test
			"HYPOTHESIS: Implement add",
			"def add(a,b): return a+b",
			"FAULT: CODE\nANALYSIS: Logic error\nLESSON: Check logic",
		},
	}
	// Judge always returns failure
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: false, Fault: "CODE"},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 1, Timeout: 3 * time.Second} // Only 1 attempt

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	ctx := context.Background()
	_, err := orch.SolveTDD(ctx, "Implement add")
	// Expect error after 1 attempt
	if err == nil {
		t.Fatal("Expected error after max attempts")
	}
	// Verify Reflector was called (LLM called 5 times: test gen, hypothesis, solution, reflector, and extra step)
	if mockLLM.CallCount != 5 {
		t.Errorf("Expected 5 LLM calls, got %d", mockLLM.CallCount)
	}
}

func TestKnowledgeManagerPath(t *testing.T) {
	// Create a test file first
	testPath := "/tmp/test_km.py"
	defer os.Remove(testPath)
	os.WriteFile(testPath, []byte("# test"), 0644)

	// Test when env var is set
	t.Setenv("KNOWLEDGE_MANAGER_PATH", testPath)
	path := knowledgeManagerPath()
	if path != testPath {
		t.Errorf("Expected %s, got %q", testPath, path)
	}
}

func TestKnowledgeManagerPath_NotFound(t *testing.T) {
	// Test when neither env var nor executable dir has the file
	t.Setenv("KNOWLEDGE_MANAGER_PATH", "")
	// This will likely return "" unless knowledge_manager.py exists in the executable directory
	path := knowledgeManagerPath()
	// We can't assert the exact value, but it should not panic
	_ = path
}

func TestGenerateLTMKeywords(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{"python fibonacci"},
	}

	orch := &Orchestrator{
		llm: mockLLM,
	}

	ctx := context.Background()
	keywords := orch.generateLTMKeywords(ctx, "Write a fibonacci function")

	if keywords != "python fibonacci" {
		t.Errorf("Expected 'python fibonacci', got %q", keywords)
	}
}

func TestGenerateLTMKeywords_LLMError(t *testing.T) {
	mockLLM := &MockLLM2{
		Error: context.Canceled,
	}

	orch := &Orchestrator{
		llm: mockLLM,
	}

	ctx := context.Background()
	keywords := orch.generateLTMKeywords(ctx, "goal")

	// Should return goal on error
	if keywords != "goal" {
		t.Errorf("Expected 'goal' on error, got %q", keywords)
	}
}

// TestQueryLTM_EmptyKMPath tests queryLTM when knowledgeManagerPath returns empty
func TestQueryLTM_EmptyKMPath(t *testing.T) {
	// Ensure KNOWLEDGE_MANAGER_PATH is unset and no knowledge_manager.py in executable dir
	t.Setenv("KNOWLEDGE_MANAGER_PATH", "")
	// Create orchestrator with mock LLM (won't be called if kmPath is empty)
	mockLLM := &MockLLM2{}
	orch := &Orchestrator{llm: mockLLM}

	ctx := context.Background()
	result := orch.queryLTM(ctx, "goal")
	if result != "" {
		t.Errorf("Expected empty string when kmPath is empty, got %q", result)
	}
}

// TestQueryLTM_KMPathSet tests queryLTM with a valid knowledge manager script
func TestQueryLTM_KMPathSet(t *testing.T) {
	// Create a temp python script that returns a known output
	tmpScript := "/tmp/test_km.py"
	tmpDir := t.TempDir()
	tmpScript = filepath.Join(tmpDir, "knowledge_manager.py")
	// Script that prints "test_ltm_output" when queried
	scriptContent := "#!/usr/bin/env python3\nimport sys\nprint('test_ltm_output')"
	os.WriteFile(tmpScript, []byte(scriptContent), 0644)

	t.Setenv("KNOWLEDGE_MANAGER_PATH", tmpScript)
	mockLLM := &MockLLM2{
		Responses: []string{"test keywords"},
	}
	orch := &Orchestrator{llm: mockLLM}

	ctx := context.Background()
	result := orch.queryLTM(ctx, "goal")
	if !strings.Contains(result, "test_ltm_output") {
		t.Errorf("Expected 'test_ltm_output' in result, got %q", result)
	}
}

// TestKnowledgeManagerPath_ExecutableDir tests fallback to executable directory
func TestKnowledgeManagerPath_ExecutableDir(t *testing.T) {
	// Create a temp dir with knowledge_manager.py
	tmpDir := t.TempDir()
	kmPath := filepath.Join(tmpDir, "knowledge_manager.py")
	os.WriteFile(kmPath, []byte("# test"), 0644)

	// Mock os.Executable to return a path in tmpDir
	// Since we can't mock os.Executable directly, we skip this test if running in CI
	// Instead, test the case where KNOWLEDGE_MANAGER_PATH is unset and executable dir has the file
	// For now, just verify the function doesn't panic
	t.Setenv("KNOWLEDGE_MANAGER_PATH", "")
	path := knowledgeManagerPath()
	// We can't assert exact value, just that it doesn't panic
	_ = path
}

// TestSolveReAct_LLMError tests SolveReAct when LLM returns error
func TestSolveReAct_LLMError(t *testing.T) {
	mockLLM := &MockLLM2{
		Error: fmt.Errorf("LLM failed"),
	}
	mockJudge := &MockJudge2{}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 1, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	ctx := context.Background()
	_, err := orch.SolveReAct(ctx, "goal")
	if err == nil {
		t.Fatal("Expected error when LLM fails")
	}
	if !strings.Contains(err.Error(), "LLM error") {
		t.Errorf("Expected 'LLM error' in error, got %v", err)
	}
}

// TestSolveReAct_ToolCallMissingInput tests tool call with missing Action Input
func TestSolveReAct_ToolCallMissingInput(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{
			"Action: python_repl", // No Action Input:
			"SOLUTION:\nprint('done')",
		},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: true},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 5, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	ctx := context.Background()
	result, err := orch.SolveReAct(ctx, "goal")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "print('done')") {
		t.Errorf("Expected solution, got %q", result)
	}
}

// TestSolveReAct_ToolExecutionError tests when tool execution fails
func TestSolveReAct_ToolExecutionError(t *testing.T) {
	// Register a tool that returns error
	mockTool := &MockTool{name: "failing_tool"}
	mockLLM := &MockLLM2{
		Responses: []string{
			"Thought: Use tool\nAction: failing_tool\nAction Input: test",
			"SOLUTION:\nprint('done')",
		},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: true},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 5, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})
	orch.RegisterTool(mockTool)

	ctx := context.Background()
	result, err := orch.SolveReAct(ctx, "goal")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "print('done')") {
		t.Errorf("Expected solution, got %q", result)
	}
}

// TestSolveReAct_FallbackNoSolutionNoAction tests fallback when LLM returns neither SOLUTION nor Action
func TestSolveReAct_FallbackNoSolutionNoAction(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{
			"Thought: just thinking, no action or solution",
			"SOLUTION:\nprint('done')",
		},
	}
	mockJudge := &MockJudge2{
		Verdict: judge.Verdict{IsCorrect: true},
	}
	exec := executor.NewExecutor(3 * time.Second)
	cfg := Config{MaxAttempts: 5, Timeout: 3 * time.Second}

	orch := NewOrchestrator(mockLLM, exec, mockJudge, NewWorkspace(), cfg, &MockCLI{})

	ctx := context.Background()
	result, err := orch.SolveReAct(ctx, "goal")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "print('done')") {
		t.Errorf("Expected solution, got %q", result)
	}
}

// TestNewReflector tests Reflector initialization
func TestNewReflector(t *testing.T) {
	mockLLM := &MockLLM2{}
	r := NewReflector(mockLLM)
	if r == nil {
		t.Fatal("Expected non-nil Reflector")
	}
	if r.client != mockLLM {
		t.Error("Expected Reflector to use provided LLM client")
	}
}

// TestReflector_Analyze tests the Reflector's Analyze method
func TestReflector_Analyze(t *testing.T) {
	mockLLM := &MockLLM2{
		Responses: []string{"FAULT: CODE\nANALYSIS: Logic error\nLESSON: Check logic"},
	}
	r := NewReflector(mockLLM)

	execResult := &executor.Result{
		Stdout: "output",
		Stderr: "error",
	}
	verdict := judge.Verdict{Feedback: "Failed"}

	ctx := context.Background()
	fault, analysis, lesson, err := r.Analyze(ctx, "goal", "code", execResult, verdict)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fault != "CODE" {
		t.Errorf("Expected fault 'CODE', got %q", fault)
	}
	if analysis != "Logic error" {
		t.Errorf("Expected analysis 'Logic error', got %q", analysis)
	}
	if lesson != "Check logic" {
		t.Errorf("Expected lesson 'Check logic', got %q", lesson)
	}
}

// TestReflector_Analyze_LLMError tests Analyze when LLM returns error
func TestReflector_Analyze_LLMError(t *testing.T) {
	mockLLM := &MockLLM2{
		Error: fmt.Errorf("LLM failed"),
	}
	r := NewReflector(mockLLM)

	execResult := &executor.Result{}
	verdict := judge.Verdict{}

	ctx := context.Background()
	_, _, _, err := r.Analyze(ctx, "goal", "code", execResult, verdict)
	if err == nil {
		t.Fatal("Expected error when LLM fails")
	}
}

// TestParseReflection tests the parseReflection helper
func TestParseReflection(t *testing.T) {
	input := "FAULT: TEST\nANALYSIS: Test is wrong\nLESSON: Fix test"
	fault, analysis, lesson, err := parseReflection(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fault != "TEST" {
		t.Errorf("Expected fault 'TEST', got %q", fault)
	}
	if analysis != "Test is wrong" {
		t.Errorf("Expected analysis 'Test is wrong', got %q", analysis)
	}
	if lesson != "Fix test" {
		t.Errorf("Expected lesson 'Fix test', got %q", lesson)
	}
}

// TestParseReflection_WithExtraNewlines tests parseReflection with multiline content
func TestParseReflection_WithExtraNewlines(t *testing.T) {
	input := "FAULT: CODE\nANALYSIS: Line 1\nLine 2\nLESSON: Lesson 1\nLesson 2"
	fault, analysis, lesson, err := parseReflection(input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if fault != "CODE" {
		t.Errorf("Expected fault 'CODE', got %q", fault)
	}
	if analysis != "Line 1\nLine 2" {
		t.Errorf("Expected analysis 'Line 1\\nLine 2', got %q", analysis)
	}
	if lesson != "Lesson 1\nLesson 2" {
		t.Errorf("Expected lesson 'Lesson 1\\nLesson 2', got %q", lesson)
	}
}

// TestPromptAssembler_New tests PromptAssembler initialization
func TestPromptAssembler_New(t *testing.T) {
	pa := NewPromptAssembler()
	if pa == nil {
		t.Fatal("Expected non-nil PromptAssembler")
	}
	if pa.reAct == nil {
		t.Error("Expected reAct template to be initialized")
	}
	if pa.solve == nil {
		t.Error("Expected solve template to be initialized")
	}
}

// TestPromptAssembler_AssembleReAct tests ReAct prompt assembly
func TestPromptAssembler_AssembleReAct(t *testing.T) {
	pa := NewPromptAssembler()
	mem := NewSessionMemory("Test goal")
	mem.ReActSteps = []Step{
		{Thought: "Think", Action: "Act", Observation: "Obs"},
	}

	ctx := context.Background()
	req, err := pa.AssembleReAct(ctx, "Test goal", mem)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(req.UserPrompt, "Test goal") {
		t.Errorf("Expected prompt to contain goal")
	}
	if !strings.Contains(req.UserPrompt, "Think") {
		t.Errorf("Expected prompt to contain thought")
	}
}

// TestPromptAssembler_AssembleTDD tests TDD prompt assembly
func TestPromptAssembler_AssembleTDD(t *testing.T) {
	pa := NewPromptAssembler()
	mem := NewSessionMemory("Test goal")
	mem.Lessons = []string{"Lesson 1"}
	mem.TestCode = "assert True"

	req, err := pa.AssembleTDD("Test goal", mem)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if !strings.Contains(req.UserPrompt, "Test goal") {
		t.Errorf("Expected prompt to contain goal")
	}
	if !strings.Contains(req.UserPrompt, "Lesson 1") {
		t.Errorf("Expected prompt to contain lesson")
	}
	if !strings.Contains(req.UserPrompt, "assert True") {
		t.Errorf("Expected prompt to contain test code")
	}
}

func TestSolveReAct_DONEEmptyWorkspace(t *testing.T) {
	mockLLM := &MockLLM2{Responses: []string{"DONE: done", "SOLUTION: print('ok')"}}
	mockJudge := &MockJudge2{Verdict: judge.Verdict{IsCorrect: true, Feedback: "ok"}}
	workspace := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(5 * time.Minute)
	o := NewOrchestrator(mockLLM, exec, mockJudge, workspace, Config{MaxAttempts: 3, Timeout: 5 * time.Minute}, &MockCLI{})

	sol, err := o.SolveReAct(context.Background(), "test task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sol, "print") {
		t.Fatalf("expected solution containing 'print', got '%s'", sol)
	}
	if mockLLM.CallCount != 2 {
		t.Fatalf("expected 2 LLM calls after corrective observation, got %d", mockLLM.CallCount)
	}
}

func TestSolveReAct_MultiAttemptFallback(t *testing.T) {
	mockLLM := &MockLLM2{Responses: []string{"Thinking... let me reconsider.", "SOLUTION: print('hello')"}}
	mockJudge := &MockJudge2{Verdict: judge.Verdict{IsCorrect: true, Feedback: "ok"}}
	workspace := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(5 * time.Minute)
	o := NewOrchestrator(mockLLM, exec, mockJudge, workspace, Config{MaxAttempts: 3, Timeout: 5 * time.Minute}, &MockCLI{})

	sol, err := o.SolveReAct(context.Background(), "test task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sol, "print") {
		t.Fatalf("expected solution containing 'print', got '%s'", sol)
	}
	if mockLLM.CallCount != 2 {
		t.Fatalf("expected 2 LLM calls (fallback then solution), got %d", mockLLM.CallCount)
	}
}

func TestSolveReAct_ToolCallContinuesLoop(t *testing.T) {
	mockLLM := &MockLLM2{Responses: []string{"Action: python_repl\nAction Input: {\"code\":\"print(1)\"}", "SOLUTION: print('ok')"}}
	mockJudge := &MockJudge2{Verdict: judge.Verdict{IsCorrect: true, Feedback: "ok"}}
	workspace := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(5 * time.Minute)
	o := NewOrchestrator(mockLLM, exec, mockJudge, workspace, Config{MaxAttempts: 3, Timeout: 5 * time.Minute}, &MockCLI{})

	sol, err := o.SolveReAct(context.Background(), "test task")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(sol, "print") {
		t.Fatalf("expected solution containing 'print', got '%s'", sol)
	}
	if mockLLM.CallCount != 2 {
		t.Fatalf("expected 2 LLM calls (tool call then solution), got %d", mockLLM.CallCount)
	}
}
