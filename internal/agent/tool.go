package agent

import (
	"context"
	"fmt"
	"strings"

	"agent_loop/internal/executor"
)

// Tool represents a capability the agent can use to extend its capabilities (calculator, repl, etc.)
type Tool interface {
	Name() string
	Description() string
	Execute(ctx context.Context, arg string) (string, error)
}

// ToolManager registers and executes tools.
type ToolManager struct {
	Tools []Tool
}

func NewToolManager() *ToolManager {
	return &ToolManager{}
}

func (tm *ToolManager) Register(t ...Tool) {
	tm.Tools = append(tm.Tools, t...)
}

func (tm *ToolManager) Execute(ctx context.Context, name string, arg string) (string, error) {
	for _, t := range tm.Tools {
		if t.Name() == name {
			return t.Execute(ctx, arg)
		}
	}
	return "", fmt.Errorf("tool not found: %s", name)
}

func (tm *ToolManager) GetToolsSummary() string {
	var sb strings.Builder
	for _, t := range tm.Tools {
		sb.WriteString(fmt.Sprintf("- %s: %s\n", t.Name(), t.Description()))
	}
	return sb.String()
}

// --- Concrete Tools ---

// PythonREPL is a tool that allows the agent to execute arbitrary python code for verification.
type PythonREPL struct {
	Executor *executor.Executor
}

func (r *PythonREPL) Name() string {
	return "python_repl"
}

func (r *PythonREPL) Description() string {
	return "You can execute python code here. Use this to calculate specific values or test snippets."
}

func (r *PythonREPL) Execute(ctx context.Context, input string) (string, error) {
	res, err := r.Executor.RunPython(ctx, input)
	if err != nil {
		return "", err
	}
	// Standardize output for the prompt
	output := res.Stdout
	if res.Stderr != "" {
		output += "\n[STDERR]: " + res.Stderr
	}
	return output, nil
}

// PerformanceChecker is a specific tool to check the time limit constraint explicitly.
type PerformanceChecker struct {
	Executor *executor.Executor
}

func (t *PerformanceChecker) Name() string {
	return "performance_checker"
}

func (t *PerformanceChecker) Description() string {
	return "Measures the execution time of a python script. Use when efficiency constraints are critical."
}

func (t *PerformanceChecker) Execute(ctx context.Context, input string) (string, error) {
	// Wrap the input in a timing script
	wrapper := fmt.Sprintf("import time; s=time.time(); %s; print('TIME:', time.time()-s, 's')\n", input)
	res, err := t.Executor.RunPython(ctx, wrapper)
	if err != nil {
		return "", err
	}
	return res.Stdout, nil
}
