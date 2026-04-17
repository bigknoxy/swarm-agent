package executor

import (
	"strings"
	"testing"
	"time"
)

func TestExecutePython_Success(t *testing.T) {
	exec := NewExecutor(2 * time.Second)
	code := `print("Hello, World!")
print(1 + 1)`
	
	res, err := exec.RunPython(code)
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
	
	res, err := exec.RunPython(code)
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
	
	res, err := exec.RunPython(code)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if res.Status != "timeout" {
		t.Errorf("Expected timeout, got %s", res.Status)
	}
}
