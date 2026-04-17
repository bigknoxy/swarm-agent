package executor

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Result holds the outcome of the code execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Status   string // "success", "failed", "timeout", "error"
}

// Executor manages the execution of external code.
type Executor struct {
	Timeout time.Duration
}

func NewExecutor(timeout time.Duration) *Executor {
	return &Executor{
		Timeout: timeout,
	}
}

// Run executes a Python script from a string.
func (e *Executor) RunPython(code string) (*Result, error) {
	tmpDir := os.TempDir()
	tmpFile := filepath.Join(tmpDir, fmt.Sprintf("agent_code_%d.py", time.Now().UnixNano()))
	err := os.WriteFile(tmpFile, []byte(code), 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(tmpFile)
	ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", tmpFile)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	result := &Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "timeout"
		result.ExitCode = 124
		return result, nil
	}
	if err != nil {
		result.Status = "failed"
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result, nil
	}
	result.Status = "success"
	result.ExitCode = 0
	return result, nil
}

// RunTDD executes a solution script against a test script in a temporary directory.
func (e *Executor) RunTDD(solutionCode, testCode string) (*Result, error) {
	tmpDir, err := os.MkdirTemp("", "agent_tdd_")
	if err != nil {
		return nil, fmt.Errorf("failed to create tmp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)
	solutionPath := filepath.Join(tmpDir, "solution.py")
	testPath := filepath.Join(tmpDir, "verify.py")
	if err := os.WriteFile(solutionPath, []byte(solutionCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write solution file: %w", err)
	}
	if err := os.WriteFile(testPath, []byte(testCode), 0644); err != nil {
		return nil, fmt.Errorf("failed to write test file: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), e.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "verify.py")
	cmd.Dir = tmpDir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	result := &Result{Stdout: stdout.String(), Stderr: stderr.String()}
	if ctx.Err() == context.DeadlineExceeded {
		result.Status = "timeout"
		result.ExitCode = 124
		return result, nil
	}
	if err != nil {
		result.Status = "failed"
		if exitError, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitError.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result, nil
	}
	result.Status = "success"
	result.ExitCode = 0
	return result, nil
}
