package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"text/template"

	"agent_loop/internal/agent"
	"agent_loop/internal/llm"
)

type PromptAssembler struct {
	reAct *template.Template
	solve *template.Template
}

func NewPromptAssembler() *PromptAssembler {
	_ = agent.NewToolManager
	tmplReAct := `# Goal: {{.Goal}}

## History
{{range .History}}### Step: {{.}}
{{end}}

## Tools Available
{{.Tools}}
`
	tmplSolve := `# Goal: {{.Goal}}

## Lessons Learned
{{range .Lessons}}- {{.}}
{{end}}

## Test Code
{{.TestCode}}
`
	return &PromptAssembler{
		reAct: template.Must(template.New("reAct").Parse(tmplReAct)),
		solve: template.Must(template.New("solve").Parse(tmplSolve)),
	}
}

func (p *PromptAssembler) AssembleReAct(ctx context.Context, goal string, memory *SessionMemory) (llm.Request, error) {
	// Format History
	var history []string
	for _, step := range memory.ReActSteps {
		history = append(history, fmt.Sprintf("Thought: %s\nAction: %s\nObservation: %s", step.Thought, step.Action, step.Observation))
	}

	// Format Tools
	var tools []string
	for _, t := range memory.RegisteredTools {
		tools = append(tools, fmt.Sprintf("- %s: %s", t.Name(), t.Description()))
	}

	// Assemble
	var buf strings.Builder
	err := p.reAct.Execute(&buf, map[string]interface{}{
		"Goal":    goal,
		"History": history,
		"Tools":   tools,
	})
	if err != nil {
		return llm.Request{}, err
	}

	return llm.Request{
		UserPrompt: buf.String(),
	}, nil
}

func (p *PromptAssembler) AssembleTDD(goal string, memory *SessionMemory) (llm.Request, error) {
	var lessons []string
	for _, l := range memory.Lessons {
		lessons = append(lessons, l)
	}

	var buf strings.Builder
	err := p.solve.Execute(&buf, map[string]interface{}{
		"Goal":       goal,
		"Lessons":    lessons,
		"TestCode":   memory.TestCode,
	})
	if err != nil {
		return llm.Request{}, err
	}

	return llm.Request{
		UserPrompt: buf.String(),
	}, nil
}
