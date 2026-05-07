package orchestrator

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewSessionMemory(t *testing.T) {
	goal := "Test goal"
	mem := NewSessionMemory(goal)

	if mem.Goal != goal {
		t.Errorf("Expected goal %q, got %q", goal, mem.Goal)
	}
	if len(mem.Hypotheses) != 0 {
		t.Errorf("Expected empty Hypotheses, got %d", len(mem.Hypotheses))
	}
	if len(mem.Lessons) != 0 {
		t.Errorf("Expected empty Lessons, got %d", len(mem.Lessons))
	}
	if len(mem.ReActSteps) != 0 {
		t.Errorf("Expected empty ReActSteps, got %d", len(mem.ReActSteps))
	}
}

func TestSessionMemory_SetGoal(t *testing.T) {
	mem := NewSessionMemory("Initial goal")
	newGoal := "Updated goal"
	mem.SetGoal(newGoal)

	if mem.Goal != newGoal {
		t.Errorf("Expected goal %q, got %q", newGoal, mem.Goal)
	}
}

func TestSessionMemory_AddHypothesis(t *testing.T) {
	mem := NewSessionMemory("Test")
	mem.AddHypothesis("Test hypothesis")

	if len(mem.Hypotheses) != 1 {
		t.Fatalf("Expected 1 hypothesis, got %d", len(mem.Hypotheses))
	}
	if mem.Hypotheses[0].Description != "Test hypothesis" {
		t.Errorf("Expected description 'Test hypothesis', got %q", mem.Hypotheses[0].Description)
	}
	if mem.Hypotheses[0].Status != "Pending" {
		t.Errorf("Expected status 'Pending', got %q", mem.Hypotheses[0].Status)
	}
}

func TestSessionMemory_UpdateHypothesis(t *testing.T) {
	mem := NewSessionMemory("Test")
	mem.AddHypothesis("H1")
	mem.AddHypothesis("H2")

	mem.UpdateHypothesis(0, "Failed", "Analysis 1")
	mem.UpdateHypothesis(1, "Verified", "Passed all tests")

	if mem.Hypotheses[0].Status != "Failed" {
		t.Errorf("Expected status 'Failed' for H1, got %q", mem.Hypotheses[0].Status)
	}
	if mem.Hypotheses[0].Outcome != "Analysis 1" {
		t.Errorf("Expected outcome 'Analysis 1' for H1, got %q", mem.Hypotheses[0].Outcome)
	}
	if mem.Hypotheses[1].Status != "Verified" {
		t.Errorf("Expected status 'Verified' for H2, got %q", mem.Hypotheses[1].Status)
	}
}

func TestSessionMemory_UpdateHypothesis_InvalidIndex(t *testing.T) {
	mem := NewSessionMemory("Test")
	mem.AddHypothesis("H1")

	// Should not panic
	mem.UpdateHypothesis(-1, "Failed", "Should not crash")
	mem.UpdateHypothesis(100, "Failed", "Should not crash")
}

func TestSessionMemory_AddLesson(t *testing.T) {
	mem := NewSessionMemory("Test")
	mem.AddLesson("Lesson 1")
	mem.AddLesson("Lesson 2")

	if len(mem.Lessons) != 2 {
		t.Fatalf("Expected 2 lessons, got %d", len(mem.Lessons))
	}
	if mem.Lessons[0] != "Lesson 1" {
		t.Errorf("Expected 'Lesson 1', got %q", mem.Lessons[0])
	}
}

func TestSessionMemory_AddTool(t *testing.T) {
	mem := NewSessionMemory("Test")
	mem.AddTool(nil) // Simplified for test

	if len(mem.RegisteredTools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(mem.RegisteredTools))
	}
}

func TestSessionMemory_GetLog(t *testing.T) {
	mem := NewSessionMemory("Test goal")
	mem.AddHypothesis("H1")
	mem.AddLesson("Lesson 1")
	mem.ReActSteps = append(mem.ReActSteps, Step{
		Thought:    "Think",
		Action:     "Act",
		Input:      "Input",
		Observation: "Obs",
	})

	log := mem.GetLog()
	if !strings.Contains(log, "Goal: Test goal") {
		t.Errorf("Expected log to contain goal")
	}
	if !strings.Contains(log, "H1") {
		t.Errorf("Expected log to contain hypothesis")
	}
	if !strings.Contains(log, "Lesson 1") {
		t.Errorf("Expected log to contain lesson")
	}
	if !strings.Contains(log, "Think") {
		t.Errorf("Expected log to contain thought")
	}
}

func TestSessionMemory_SaveAndLoad(t *testing.T) {
	tmpFile := "/tmp/test_memory.json"
	defer os.Remove(tmpFile)

	mem := NewSessionMemory("Test goal")
	mem.AddHypothesis("Test hypothesis")
	mem.AddLesson("Test lesson")

	ctx := context.Background()
	err := mem.Save(ctx, tmpFile)
	if err != nil {
		t.Fatalf("Failed to save: %v", err)
	}

	// Create new memory and load
	loaded := NewSessionMemory("")
	err = loaded.Load(ctx, tmpFile)
	if err != nil {
		t.Fatalf("Failed to load: %v", err)
	}

	if loaded.Goal != "Test goal" {
		t.Errorf("Expected goal 'Test goal', got %q", loaded.Goal)
	}
	if len(loaded.Hypotheses) != 1 {
		t.Errorf("Expected 1 hypothesis, got %d", len(loaded.Hypotheses))
	}
	if len(loaded.Lessons) != 1 {
		t.Errorf("Expected 1 lesson, got %d", len(loaded.Lessons))
	}
}

func TestSessionMemory_SaveError(t *testing.T) {
	mem := NewSessionMemory("Test")
	// Try to save to invalid path
	err := mem.Save(context.Background(), "/nonexistent/dir/file.json")
	if err == nil {
		t.Error("Expected error for invalid path")
	}
}

func TestSessionMemory_LoadError(t *testing.T) {
	mem := NewSessionMemory("Test")
	err := mem.Load(context.Background(), "/nonexistent/file.json")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestSessionMemory_FormatTimestamp(t *testing.T) {
	mem := NewSessionMemory("Test")
	ts := mem.FormatTimestamp()

	// Should be able to parse it as time
	_, err := time.Parse(time.TimeOnly, ts)
	if err != nil {
		t.Errorf("FormatTimestamp returned invalid time format: %v", err)
	}
}
