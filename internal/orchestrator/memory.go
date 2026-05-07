package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"agent_loop/internal/agent"
)

type Step struct {
	Thought    string
	Action     string
	Input      string
	Observation string
}

type Hypothesis struct {
	Description string
	Status      string // "Pending", "Failed", "Verified"
	Outcome     string
}

type SessionMemory struct {
	Goal               string
	ReActSteps         []Step     // History of the ReAct loop
	Lessons            []string   // High-level insights (e.g. "Tool X failed on empty input")
	Hypotheses         []Hypothesis
	RegisteredTools    []agent.Tool
	TestCode           string
}

// SetGoal updates the current goal (useful if the agent refines the goal).
func (m *SessionMemory) SetGoal(g string) {
	m.Goal = g
}

// --- Persistence ---

// Save writes the session memory to a JSON file.
func (m *SessionMemory) Save(ctx context.Context, path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal memory: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// Load reads session memory from a JSON file.
func (m *SessionMemory) Load(ctx context.Context, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read memory file: %w", err)
	}
	return json.Unmarshal(data, m)
}

// FormatTimestamp returns a string representation of the current time for logging.
func (m *SessionMemory) FormatTimestamp() string {
	return time.Now().Format(time.TimeOnly)
}

func NewSessionMemory(goal string) *SessionMemory {
	return &SessionMemory{
		Goal:               goal,
		Hypotheses:         []Hypothesis{},
		Lessons:            []string{},
		RegisteredTools:    []agent.Tool{},
	}
}

func (m *SessionMemory) AddHypothesis(desc string) {
	m.Hypotheses = append(m.Hypotheses, Hypothesis{
		Description: desc,
		Status:      "Pending",
	})
}

func (m *SessionMemory) UpdateHypothesis(index int, status, outcome string) {
	if index >= 0 && index < len(m.Hypotheses) {
		m.Hypotheses[index].Status = status
		m.Hypotheses[index].Outcome = outcome
	}
}

func (m *SessionMemory) AddLesson(lesson string) {
	m.Lessons = append(m.Lessons, lesson)
}

// AddTool registers a new tool.
func (m *SessionMemory) AddTool(t agent.Tool) {
	m.RegisteredTools = append(m.RegisteredTools, t)
}

func (m *SessionMemory) GetLog() string {
	log := fmt.Sprintf("Goal: %s\n\n--- Known Lessons ---\n", m.Goal)
	if len(m.Lessons) == 0 {
		log += "No lessons learned yet.\n"
	} else {
		for i, l := range m.Lessons {
			log += fmt.Sprintf("%d. %s\n", i+1, l)
		}
	}

	log += "\n--- Hypothesis Track ---\n"
	if len(m.Hypotheses) == 0 {
		log += "No hypotheses tracked.\n"
	} else {
		for i, h := range m.Hypotheses {
			log += fmt.Sprintf("%d. [%s] %s -> %s\n", i+1, h.Status, h.Description, h.Outcome)
		}
	}

	log += "\n--- ReAct Execution History ---\n\n### HISTORY ###\n"
	if len(m.ReActSteps) == 0 {
		log += "No history yet.\n"

	} else {
		for i, step := range m.ReActSteps {
			log += fmt.Sprintf("%d. Thought: %s\nAction: %s\nInput: %s\nObservation: %s\n---\n",
				i+1, step.Thought, step.Action, step.Input, step.Observation)
		}
	}

	return log
}
