package orchestrator

import (
	"context"
	"fmt"
	"strings"

	"agent_loop/internal/executor"
	"agent_loop/internal/judge"
	"agent_loop/internal/llm"
)

type Reflector struct {
	client *llm.Client
}

func NewReflector(client *llm.Client) *Reflector {
	return &Reflector{client: client}
}

// Analyze performs a root-cause analysis on a failed execution.
func (r *Reflector) Analyze(ctx context.Context, goal string, code string, res *executor.Result, verdict judge.Verdict) (string, string, string, error) {
	prompt := fmt.Sprintf(
		"You are a Senior Software Engineer. A coder tried to solve the following goal:\nGoal: %s\n\n"+
			"The code they wrote:\n%s\n\n"+
			"The actual output was:\n%s\n"+
			"The error output was:\n%s\n\n"+
			"The judge's verdict was: %s\n\n"+
			"Please provide three things:\n"+
			"1. FAULT: Either 'CODE' (the implementation is wrong) or 'TEST' (the test case/assertion is wrong).\n"+
			"2. ANALYSIS: Why did this fail? Be surgical.\n"+
			"3. LESSON: A general rule to avoid this mistake in the future.\n"+
			"Format your response exactly as:\nFAULT: <CODE|TEST>\nANALYSIS: <analysis>\nLESSON: <lesson>",
		goal, code, res.Stdout, res.Stderr, verdict.Feedback,
	)

	req := llm.Request{
		SystemPrompt: "You are a critical code reviewer. You must attribute the fault to either the code or the test suite.",
		UserPrompt:   prompt,
	}

	resp, err := r.client.Generate(ctx, req)
	if err != nil {
		return "", "", "", err
	}

	return parseReflection(resp.Text)
}

func parseReflection(text string) (string, string, string, error) {
	var fault, analysis, lesson string
	lines := strings.Split(text, "\n")
	current := ""
	for _, line := range lines {
		lowerLine := strings.ToLower(line)
		if strings.HasPrefix(lowerLine, "fault:") {
			current = "fault"
			fault = strings.TrimPrefix(line, "FAULT:")
			fault = strings.TrimPrefix(fault, "fault:")
			continue
		}
		if strings.HasPrefix(lowerLine, "analysis:") {
			current = "analysis"
			analysis = strings.TrimPrefix(line, "ANALYSIS:")
			analysis = strings.TrimPrefix(analysis, "analysis:")
			continue
		}
		if strings.HasPrefix(lowerLine, "lesson:") {
			current = "lesson"
			lesson = strings.TrimPrefix(line, "LESSON:")
			lesson = strings.TrimPrefix(lesson, "lesson:")
			continue
		}
		if current == "fault" {
			fault += "\n" + line
		} else if current == "analysis" {
			analysis += "\n" + line
		} else if current == "lesson" {
			lesson += "\n" + line
		}
	}
	return strings.TrimSpace(fault), strings.TrimSpace(analysis), strings.TrimSpace(lesson), nil
}
