package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(fn func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestCLIOutput_QuietMode(t *testing.T) {
	out := NewCLIOutput(false, false, true)
	got := captureStdout(func() {
		out.PrintStart("goal")
		out.PrintSuccess("done")
	})
	if got != "" {
		t.Errorf("expected empty output in quiet mode, got %q", got)
	}
}

func TestCLIOutput_JSONMode_Start(t *testing.T) {
	out := NewCLIOutput(false, true, false)
	got := captureStdout(func() {
		out.PrintStart("my goal")
	})
	if !strings.Contains(got, `"event": "start"`) || !strings.Contains(got, "my goal") {
		t.Errorf("expected JSON start output, got %q", got)
	}
}

func TestCLIOutput_JSONMode_Success(t *testing.T) {
	out := NewCLIOutput(false, true, false)
	got := captureStdout(func() {
		out.PrintSuccess("done")
	})
	if !strings.Contains(got, `"event": "success"`) {
		t.Errorf("expected JSON success output, got %q", got)
	}
}

func TestCLIOutput_JSONMode_Failure(t *testing.T) {
	out := NewCLIOutput(false, true, false)
	got := captureStdout(func() {
		out.PrintFailure("oops")
	})
	if !strings.Contains(got, `"event": "failure"`) {
		t.Errorf("expected JSON failure output, got %q", got)
	}
}

func TestCLIOutput_VerboseMode(t *testing.T) {
	out := NewCLIOutput(true, false, false)
	got := captureStdout(func() {
		out.PrintVerbose("PROMPT", "content")
	})
	if !strings.Contains(got, "PROMPT") {
		t.Errorf("expected verbose output, got %q", got)
	}
}

func TestCLIOutput_VerboseSuppressedInJSON(t *testing.T) {
	out := NewCLIOutput(true, true, false)
	got := captureStdout(func() {
		out.PrintVerbose("PROMPT", "content")
	})
	if got != "" {
		t.Errorf("expected empty output when verbose+json, got %q", got)
	}
}

func TestCLIOutput_PrintThought_Normal(t *testing.T) {
	// color lib writes to terminal directly; verify no panic
	NewCLIOutput(false, false, false).PrintThought("my thought")
}

func TestCLIOutput_PrintThought_JSON(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, true, false).PrintThought("my thought") })
	if !strings.Contains(out, `"thought"`) || !strings.Contains(out, "my thought") {
		t.Errorf("expected JSON thought output, got %q", out)
	}
}

func TestCLIOutput_PrintThought_Quiet(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, false, true).PrintThought("my thought") })
	if out != "" {
		t.Errorf("expected empty output in quiet mode, got %q", out)
	}
}

func TestCLIOutput_PrintAction_Normal(t *testing.T) {
	// color lib writes to terminal directly; verify no panic
	NewCLIOutput(false, false, false).PrintAction("tool_name")
}

func TestCLIOutput_PrintAction_JSON(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, true, false).PrintAction("tool_name") })
	if !strings.Contains(out, `"action"`) || !strings.Contains(out, "tool_name") {
		t.Errorf("expected JSON action output, got %q", out)
	}
}

func TestCLIOutput_PrintAction_Quiet(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, false, true).PrintAction("tool_name") })
	if out != "" {
		t.Errorf("expected empty output in quiet mode, got %q", out)
	}
}

func TestCLIOutput_PrintObservation_Normal(t *testing.T) {
	// color lib writes to terminal directly; verify no panic
	NewCLIOutput(false, false, false).PrintObservation("obs text")
}

func TestCLIOutput_PrintObservation_JSON(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, true, false).PrintObservation("obs text") })
	if !strings.Contains(out, `"observation"`) || !strings.Contains(out, "obs text") {
		t.Errorf("expected JSON observation output, got %q", out)
	}
}

func TestCLIOutput_PrintObservation_Quiet(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, false, true).PrintObservation("obs text") })
	if out != "" {
		t.Errorf("expected empty output in quiet mode, got %q", out)
	}
}

func TestCLIOutput_PrintInfo_Normal(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, false, false).PrintInfo("info msg") })
	if !strings.Contains(out, "info msg") {
		t.Errorf("expected 'info msg' in output, got %q", out)
	}
}

func TestCLIOutput_PrintInfo_JSON(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, true, false).PrintInfo("info msg") })
	if !strings.Contains(out, `"info"`) || !strings.Contains(out, "info msg") {
		t.Errorf("expected JSON info output, got %q", out)
	}
}

func TestCLIOutput_PrintInfo_Quiet(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, false, true).PrintInfo("info msg") })
	if out != "" {
		t.Errorf("expected empty output in quiet mode, got %q", out)
	}
}

func TestCLIOutput_PrintJudgeVerdict_Passed_JSON(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, true, false).PrintJudgeVerdict("passed") })
	if !strings.Contains(out, `"judge"`) || !strings.Contains(out, "passed") {
		t.Errorf("expected JSON judge output, got %q", out)
	}
}

func TestCLIOutput_PrintJudgeVerdict_Rejected_Normal(t *testing.T) {
	// color lib writes to terminal directly; verify no panic
	NewCLIOutput(false, false, false).PrintJudgeVerdict("rejected")
}

func TestCLIOutput_PrintJudgeVerdict_Quiet(t *testing.T) {
	out := captureStdout(func() { NewCLIOutput(false, false, true).PrintJudgeVerdict("passed") })
	if out != "" {
		t.Errorf("expected empty output in quiet mode, got %q", out)
	}
}

func TestCLIOutput_NewProgressBar_ReturnsNil(t *testing.T) {
	cli := NewCLIOutput(false, false, false)
	if cli.NewProgressBar(10) != nil {
		t.Error("expected NewProgressBar to return nil")
	}
}
