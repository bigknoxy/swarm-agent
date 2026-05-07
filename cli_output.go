package main

import (
	"fmt"

	"github.com/fatih/color"
)

// CLIOutput handles all formatted output
type CLIOutput struct {
	Verbose bool
	JSON    bool
	Quiet   bool
}

// NewCLIOutput creates a new CLI output handler
func NewCLIOutput(verbose, jsonOutput, quiet bool) *CLIOutput {
	return &CLIOutput{
		Verbose: verbose,
		JSON:    jsonOutput,
		Quiet:   quiet,
	}
}

// PrintStart prints the initial startup message
func (c *CLIOutput) PrintStart(goal string) {
	if c.JSON {
		fmt.Printf(`{"event": "start", "goal": "%s"}\n`, goal)
		return
	}
	if !c.Quiet {
		color.Cyan("🚀 Starting AI Agent Loop (Phase 6: Golden Path)...")
		color.Yellow("Goal: %s\n", goal)
	}
}

// PrintThought prints a thought message
func (c *CLIOutput) PrintThought(thought string) {
	if c.JSON {
		fmt.Printf(`{"event": "thought", "content": "%s"}\n`, thought)
		return
	}
	if !c.Quiet {
		color.Yellow("💭 Thought: %s", thought)
	}
}

// PrintAction prints an action message
func (c *CLIOutput) PrintAction(action string) {
	if c.JSON {
		fmt.Printf(`{"event": "action", "tool": "%s"}\n`, action)
		return
	}
	if !c.Quiet {
		color.Blue("🔧 Action: %s", action)
	}
}

// PrintObservation prints an observation message
func (c *CLIOutput) PrintObservation(obs string) {
	if c.JSON {
		fmt.Printf(`{"event": "observation", "content": "%s"}\n`, obs)
		return
	}
	if !c.Quiet {
		color.Green("👀 Observation: %s", obs)
	}
}

// PrintSuccess prints a success message
func (c *CLIOutput) PrintSuccess(msg string) {
	if c.JSON {
		fmt.Printf(`{"event": "success", "message": "%s"}\n`, msg)
		return
	}
	if !c.Quiet {
		color.New(color.FgGreen, color.Bold).Printf("✅ %s\n", msg)
	}
}

// PrintFailure prints a failure message
func (c *CLIOutput) PrintFailure(msg string) {
	if c.JSON {
		fmt.Printf(`{"event": "failure", "message": "%s"}\n`, msg)
		return
	}
	if !c.Quiet {
		color.New(color.FgRed, color.Bold).Printf("❌ %s\n", msg)
	}
}

// PrintInfo prints an info message
func (c *CLIOutput) PrintInfo(msg string) {
	if c.JSON {
		fmt.Printf(`{"event": "info", "message": "%s"}\n`, msg)
		return
	}
	if !c.Quiet {
		fmt.Println(msg)
	}
}

// PrintVerbose prints verbose output (prompts, responses)
func (c *CLIOutput) PrintVerbose(label, content string) {
	if c.Verbose && !c.JSON {
		fmt.Printf("[VERBOSE] %s:\n%s\n", label, content)
	}
}

// NewProgressBar creates a new progress bar for LLM generation (placeholder)
func (c *CLIOutput) NewProgressBar(max int) interface{} {
	return nil
}

// PrintJudgeVerdict prints the judge's verdict
func (c *CLIOutput) PrintJudgeVerdict(verdict string) {
	if c.JSON {
		fmt.Printf(`{"event": "judge", "verdict": "%s"}\n`, verdict)
		return
	}
	if !c.Quiet {
		if verdict == "passed" {
			color.Green("✅ Judge: %s", verdict)
		} else {
			color.Red("❌ Judge: %s", verdict)
		}
	}
}
