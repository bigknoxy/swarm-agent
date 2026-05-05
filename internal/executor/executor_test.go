package executor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestExecutePython_Success(t *testing.T) {
	exec := NewExecutor(2 * time.Second)
	code := `print("Hello, World!")
print(1 + 1)`
	
	res, err := exec.RunPython(context.Background(), code)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if res.Status != "success" {
		t.Errorf("Expected success, got %s (stderr: %s)", res.Status, res.Stderr)
	}
	
	if !strings.Contains(res.Stdout, "Hello, World!") || !strings.Contains(res.Stdout, "2") {
		t.Errorf("Unexpected output: %s", res.Stdout)
	}
}

func TestExecutePython_Failure(t *testing.T) {
	exec := NewExecutor(2 * time.Second)
	code := `print(1 / 0)`
	
	res, err := exec.RunPython(context.Background(), code)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if res.Status != "failed" {
		t.Errorf("Expected failed, got %s", res.Status)
	}
	
	if !strings.Contains(res.Stderr, "ZeroDivisionError") {
		t.Errorf("Expected ZeroDivisionError in stderr, got: %s", res.Stderr)
	}
}

func TestExecutePython_Timeout(t *testing.T) {
	exec := NewExecutor(500 * time.Millisecond)
	code := `import time
time.sleep(2)`
	
	res, err := exec.RunPython(context.Background(), code)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if res.Status != "timeout" {
		t.Errorf("Expected timeout, got %s", res.Status)
	}
	if res.ExitCode != 124 {
		t.Errorf("Expected exit code 124 for timeout, got %d", res.ExitCode)
	}
}

func TestExecutePython_ExitCode(t *testing.T) {
	exec := NewExecutor(2 * time.Second)
	
	tests := []struct {
		name     string
		code     string
		expected int
	}{
		{"exit 0", "exit(0)", 0},
		{"exit 1", "exit(1)", 1},
		{"exit 42", "exit(42)", 42},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := exec.RunPython(context.Background(), tt.code)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if res.ExitCode != tt.expected {
				t.Errorf("Expected exit code %d, got %d", tt.expected, res.ExitCode)
			}
		})
	}
}

func TestExecutePython_InvalidSyntax(t *testing.T) {
	exec := NewExecutor(2 * time.Second)
	code := `this is not valid python`
	
	res, err := exec.RunPython(context.Background(), code)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if res.Status != "failed" {
		t.Errorf("Expected failed, got %s", res.Status)
	}
	if !strings.Contains(res.Stderr, "SyntaxError") {
		t.Errorf("Expected SyntaxError in stderr, got: %s", res.Stderr)
	}
}

func TestRunTDD_Success(t *testing.T) {
	exec := NewExecutor(5 * time.Second)
	
	solutionCode := `def add(a, b):
    return a + b`
	
	testCode := `from solution import add
assert add(1, 2) == 3
assert add(-1, 1) == 0
assert add(0, 0) == 0
print("All tests passed!")`
	
	res, err := exec.RunTDD(context.Background(), solutionCode, testCode)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if res.Status != "success" {
		t.Errorf("Expected success, got %s (stderr: %s)", res.Status, res.Stderr)
	}
	if !strings.Contains(res.Stdout, "All tests passed!") {
		t.Errorf("Expected 'All tests passed!' in stdout, got: %s", res.Stdout)
	}
}

func TestRunTDD_Failure(t *testing.T) {
	exec := NewExecutor(5 * time.Second)
	
	solutionCode := `def add(a, b):
    return a - b  # Wrong implementation`
	
	testCode := `from solution import add
assert add(1, 2) == 3`
	
	res, err := exec.RunTDD(context.Background(), solutionCode, testCode)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if res.Status != "failed" {
		t.Errorf("Expected failed, got %s", res.Status)
	}
}

func TestRunTDD_Timeout(t *testing.T) {
	exec := NewExecutor(500 * time.Millisecond)
	
	solutionCode := `def dummy():
    pass`
	
	testCode := `import time
time.sleep(2)`
	
	res, err := exec.RunTDD(context.Background(), solutionCode, testCode)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if res.Status != "timeout" {
		t.Errorf("Expected timeout, got %s", res.Status)
	}
	if res.ExitCode != 124 {
		t.Errorf("Expected exit code 124 for timeout, got %d", res.ExitCode)
	}
}

func TestNewExecutor(t *testing.T) {
	timeout := 5 * time.Minute
	exec := NewExecutor(timeout)
	
	if exec.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, exec.Timeout)
	}
}

func TestExecutePython_WriteFileError(t *testing.T) {
	// This is hard to test directly since os.TempDir() is used
	// In a real scenario, you'd use a writable path or mock os.WriteFile
	// For now, we'll skip this test
	t.Skip("Skipping write file error test - requires file system mocking")
}

func TestRunTDD_WriteFileError(t *testing.T) {
	// Similar to above - hard to test without mocking
	t.Skip("Skipping write file error test - requires file system mocking")
}

func TestRunShell_Basic(t *testing.T) {
	exec := NewExecutor(2 * time.Second)
	res, err := exec.RunShell(context.Background(), "echo hello", "", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "success" {
		t.Errorf("expected status success, got %s (stdout: %s)", res.Status, res.Stdout)
	}
	if res.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", res.ExitCode)
	}
	if !strings.Contains(res.Stdout, "hello") {
		t.Errorf("expected stdout to contain 'hello', got %q", res.Stdout)
	}
}

func TestRunShell_ExitCode(t *testing.T) {
	exec := NewExecutor(2 * time.Second)
	res, err := exec.RunShell(context.Background(), "exit 42", "", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "failed" {
		t.Errorf("expected status failed, got %s", res.Status)
	}
	if res.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", res.ExitCode)
	}
}

func TestRunShell_Timeout(t *testing.T) {
	exec := NewExecutor(2 * time.Second)
	res, err := exec.RunShell(context.Background(), "sleep 10", "", 100*time.Millisecond)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Status != "timeout" {
		t.Errorf("expected status timeout, got %s", res.Status)
	}
	if res.ExitCode != 124 {
		t.Errorf("expected exit code 124, got %d", res.ExitCode)
	}
}

func TestRunShell_WorksInCwd(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "testfile.txt"), []byte("hi"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}
	exec := NewExecutor(2 * time.Second)
	res, err := exec.RunShell(context.Background(), "ls", dir, 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Stdout, "testfile.txt") {
		t.Errorf("expected stdout to contain 'testfile.txt', got %q", res.Stdout)
	}
}

func TestRunShell_CombinedOutput(t *testing.T) {
	exec := NewExecutor(2 * time.Second)
	res, err := exec.RunShell(context.Background(), "echo out; echo err >&2", "", 5*time.Second)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(res.Stdout, "out") {
		t.Errorf("expected stdout to contain 'out', got %q", res.Stdout)
	}
	if !strings.Contains(res.Stdout, "err") {
		t.Errorf("expected stdout to contain 'err', got %q", res.Stdout)
	}
}
