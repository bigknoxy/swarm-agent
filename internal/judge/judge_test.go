package judge

import (
	"agent_loop/internal/executor"
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
