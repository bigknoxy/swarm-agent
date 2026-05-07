package judge

import (
	"agent_loop/internal/executor"
	"strings"
	"testing"
)

func TestJudge_Evaluate(t *testing.T) {
	j := NewJudge()

	tests := []struct {
		name     string
		res      *executor.Result
		exp      Expectation
		expected bool
	}{
		{
			name: "SuccessCase",
			res: &executor.Result{
				Stdout: "Result is 42",
				Status: "success",
			},
			exp: Expectation{
				ExpectedOutput: []string{"42"},
				RequireSuccess: true,
			},
			expected: true,
		},
		{
			name: "MissingOutput",
			res: &executor.Result{
				Stdout: "Result is 0",
				Status: "success",
			},
			exp: Expectation{
				ExpectedOutput: []string{"42"},
				RequireSuccess: true,
			},
			expected: false,
		},
		{
			name: "FailedExecution",
			res: &executor.Result{
				Stderr: "ZeroDivisionError",
				Status: "failed",
			},
			exp: Expectation{
				RequireSuccess: true,
			},
			expected: false,
		},
		{
			name: "ForbiddenOutput",
			res: &executor.Result{
				Stdout: "Error: something went wrong",
				Status: "success",
			},
			exp: Expectation{
				ForbiddenOutput: []string{"Error"},
				RequireSuccess: true,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verdict := j.Evaluate(tt.res, tt.exp)
			if verdict.IsCorrect != tt.expected {
				t.Errorf("Expected correctness %v, got %v. Feedback: %s", tt.expected, verdict.IsCorrect, verdict.Feedback)
			}
		})
	}
}

func TestJudge_EvaluateTDD(t *testing.T) {
	j := NewJudge()

	tests := []struct {
		name        string
		res         *executor.Result
		expected    bool
		hasMismatch bool
	}{
		{
			name: "TDD_Success",
			res: &executor.Result{
				ExitCode: 0,
				Stdout:   "All tests passed!",
				Status:   "success",
			},
			expected:    true,
			hasMismatch: false,
		},
		{
			name: "TDD_Failure_NoAssertion",
			res: &executor.Result{
				ExitCode: 1,
				Stderr:   "Some error occurred",
				Status:   "failed",
			},
			expected:    false,
			hasMismatch: false,
		},
		{
			name: "TDD_Failure_WithAssertion",
			res: &executor.Result{
				ExitCode: 1,
				Stderr:   "AssertionError: assert 1 == 2",
				Status:   "failed",
			},
			expected:    false,
			hasMismatch: true,
		},
		{
			name: "TDD_Timeout",
			res: &executor.Result{
				ExitCode: 124,
				Status:   "timeout",
			},
			expected:    false,
			hasMismatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			verdict := j.EvaluateTDD(tt.res)
			if verdict.IsCorrect != tt.expected {
				t.Errorf("Expected correctness %v, got %v. Feedback: %s", tt.expected, verdict.IsCorrect, verdict.Feedback)
			}
			if tt.hasMismatch && verdict.ValueMismatch == nil {
				t.Errorf("Expected ValueMismatch, got nil")
			}
			if !tt.hasMismatch && verdict.ValueMismatch != nil {
				t.Errorf("Expected no ValueMismatch, got %+v", verdict.ValueMismatch)
			}
		})
	}
}

func TestJudge_Evaluate_RequireSuccessFalse(t *testing.T) {
	j := NewJudge()

	// When RequireSuccess is false, failed execution should still pass if output matches
	res := &executor.Result{
		Stdout: "Some output 42",
		Status: "failed", // Failed but we don't require success
	}

	exp := Expectation{
		ExpectedOutput: []string{"42"},
		RequireSuccess: false,
	}

	verdict := j.Evaluate(res, exp)
	if !verdict.IsCorrect {
		t.Errorf("Expected correct when RequireSuccess=false, got false. Feedback: %s", verdict.Feedback)
	}
}

func TestJudge_Evaluate_MultipleExpectedOutputs(t *testing.T) {
	j := NewJudge()

	res := &executor.Result{
		Stdout: "Hello World 42",
		Status: "success",
	}

	exp := Expectation{
		ExpectedOutput: []string{"Hello", "World", "42"},
		RequireSuccess: true,
	}

	verdict := j.Evaluate(res, exp)
	if !verdict.IsCorrect {
		t.Errorf("Expected correct with multiple outputs, got false. Feedback: %s", verdict.Feedback)
	}
}

func TestJudge_Evaluate_MissingOneOfMultiple(t *testing.T) {
	j := NewJudge()

	res := &executor.Result{
		Stdout: "Hello 42",
		Status: "success",
	}

	exp := Expectation{
		ExpectedOutput: []string{"Hello", "World"}, // "World" is missing
		RequireSuccess: true,
	}

	verdict := j.Evaluate(res, exp)
	if verdict.IsCorrect {
		t.Errorf("Expected incorrect when missing output, got true")
	}
	if !strings.Contains(verdict.Feedback, "World") {
		t.Errorf("Expected feedback about missing 'World', got: %s", verdict.Feedback)
	}
}

func TestExtractActual(t *testing.T) {
	tests := []struct {
		stderr   string
		expected string
	}{
		{"", ""},
		{"Some error", "... (see stderr) ..."},
		{"AssertionError", "... (see stderr) ..."},
	}

	for _, tt := range tests {
		result := extractActual(tt.stderr)
		if result != tt.expected {
			t.Errorf("extractActual(%q) = %q, expected %q", tt.stderr, result, tt.expected)
		}
	}
}

func TestExtractExpected(t *testing.T) {
	tests := []struct {
		stderr   string
		expected string
	}{
		{"", ""},
		{"assert 1 == 2", ""}, // No "should be" in this
		{"AssertionError: should be 42", "42"},
		{"Error: value should be 100", "100"},
	}

	for _, tt := range tests {
		result := extractExpected(tt.stderr)
		if result != tt.expected {
			t.Errorf("extractExpected(%q) = %q, expected %q", tt.stderr, result, tt.expected)
		}
	}
}

func TestNewJudge(t *testing.T) {
	j := NewJudge()
	if j == nil {
		t.Error("Expected non-nil Judge")
	}
}
