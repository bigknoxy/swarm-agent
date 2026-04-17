package judge

import (
	"strings"
	"agent_loop/internal/executor"
)

// Expectation defines what the judge looks for in the execution result.
type Expectation struct {
	ExpectedOutput []string // Must contain these strings
	ForbiddenOutput []string // Must NOT contain these strings
	RequireSuccess  bool     // Exit code must be 0
}

// Verdict represents the outcome of the judgment.
type Verdict struct {
	IsCorrect    bool
	Feedback     string
	ValueMismatch *ValueMismatch // Present if an assertion error occurred
}

// ValueMismatch represents a specific value disagreement between code and test
type ValueMismatch struct {
	Actual   string
	Expected string
	Location string // e.g. "assertion line 10"
}

// Judge analyzes the executor result against the expectations.
type Judge struct{}

func NewJudge() *Judge {
	return &Judge{}
}

func (j *Judge) Evaluate(res *executor.Result, exp Expectation) Verdict {
	if exp.RequireSuccess && res.Status != "success" {
		return Verdict{
			IsCorrect: false,
			Feedback:  "Code failed to execute successfully. Error: " + res.Stderr,
		}
	}

	for _, expected := range exp.ExpectedOutput {
		if !strings.Contains(res.Stdout, expected) {
			return Verdict{
				IsCorrect: false,
				Feedback:  "Output missing expected string: '" + expected + "'",
			}
		}
	}

	for _, forbidden := range exp.ForbiddenOutput {
		if strings.Contains(res.Stdout, forbidden) {
			return Verdict{
				IsCorrect: false,
				Feedback:  "Output contains forbidden string: '" + forbidden + "'",
			}
		}
	}

	return Verdict{
		IsCorrect: true,
		Feedback:  "Verification passed!",
	}
}

func (j *Judge) EvaluateTDD(res *executor.Result) Verdict {
	if res.ExitCode == 0 {
		return Verdict{
			IsCorrect: true,
			Feedback:  "TDD Tests Passed!",
		}
	}
	
	// Extract value mismatch info from the traceback if present
	var mismatch *ValueMismatch
	if strings.Contains(res.Stderr, "AssertionError:") {
		mismatch = &ValueMismatch{
			Actual:   extractActual(res.Stderr),
			Expected: extractExpected(res.Stderr),
			Location: "assertion line in verify.py",
		}
	}
	
	return Verdict{
		IsCorrect: false,
		Feedback:  "TDD Tests Failed. Stderr: " + res.Stderr,
		ValueMismatch: mismatch,
	}
}

func extractActual(stderr string) string {
	// Simple heuristic: look for the first number in the "assert ... == ..." error
	// Usually stdout has the actual value if the code prints it, or we can parse stderr
	if len(stderr) > 0 {
		return "... (see stderr) ..."
	}
	return ""
}

func extractExpected(stderr string) string {
	// Look for "should be <value>" in the error message
	if strings.Contains(stderr, "should be ") {
		parts := strings.Split(stderr, "should be ")
		if len(parts) > 1 {
			fields := strings.Fields(strings.TrimSpace(parts[len(parts)-1]))
			if len(fields) > 0 {
				return fields[0]
			}
		}
	}
	return ""
}
