package orchestrator

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent_loop/internal/executor"
)

func TestBuildContext_NonGitDir(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 1; i <= 3; i++ {
		content := []byte(fmt.Sprintf("package main\n\nfunc F%d() {}\n", i))
		if err := os.WriteFile(filepath.Join(tmpDir, fmt.Sprintf("f%d.go", i)), content, 0644); err != nil {
			t.Fatal(err)
		}
	}

	exec := executor.NewExecutor(5 * time.Second)
	ws := NewWorkspaceWithPath(tmpDir)

	pc, err := BuildContext("test goal", tmpDir, exec, ws)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(pc.TopFiles) < 3 {
		t.Errorf("expected TopFiles >= 3, got %d", len(pc.TopFiles))
	}
	if pc.GitStatus != "" {
		t.Errorf("expected empty GitStatus for non-git dir, got %q", pc.GitStatus)
	}
}

func TestBuildContext_KeywordMatching(t *testing.T) {
	tmpDir := t.TempDir()
	for name, content := range map[string]string{
		"login.go":  "package main\n",
		"auth.go":   "package main\n",
		"README.md": "# Docs\n",
	} {
		if err := os.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	exec := executor.NewExecutor(5 * time.Second)
	ws := NewWorkspaceWithPath(tmpDir)

	pc, err := BuildContext("add a test for the login function", tmpDir, exec, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	hasLogin := false
	for _, f := range pc.RelevantFiles {
		if strings.Contains(f, "login") {
			hasLogin = true
			break
		}
	}
	if !hasLogin {
		t.Error("expected RelevantFiles to contain a 'login' entry")
	}
}

func TestBuildContext_FileSnippet(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmpDir, "login.go"), []byte("package main\nfunc Login() {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	exec := executor.NewExecutor(5 * time.Second)
	ws := NewWorkspaceWithPath(tmpDir)

	pc, err := BuildContext("fix the login function", tmpDir, exec, ws)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(pc.FileSnippet, "Login") {
		t.Errorf("expected FileSnippet to contain 'Login', got %q", pc.FileSnippet)
	}
}

func TestBuildContext_EmptyWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	exec := executor.NewExecutor(5 * time.Second)
	ws := NewWorkspaceWithPath(tmpDir)

	pc, err := BuildContext("hello world", tmpDir, exec, ws)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if len(pc.WorkspaceFiles) > 0 {
		t.Errorf("expected empty WorkspaceFiles, got %v", pc.WorkspaceFiles)
	}
}

func TestProjectContext_Format_Empty(t *testing.T) {
	pc := &ProjectContext{}
	result := pc.Format()
	if !strings.Contains(result, "## Project Context") {
		t.Errorf("expected Format() to contain '## Project Context', got %q", result)
	}
}

func TestContextScanSummary_NoFiles(t *testing.T) {
	pc := &ProjectContext{}
	if result := ContextScanSummary(pc); result != "" {
		t.Errorf("expected empty string for empty TopFiles, got %q", result)
	}
}

func TestContextScanSummary_WithMatches(t *testing.T) {
	pc := &ProjectContext{
		TopFiles:      []string{"login.go"},
		RelevantFiles: []string{"login.go"},
		Keywords:      []string{"login"},
	}
	result := ContextScanSummary(pc)
	if !strings.Contains(result, "[Context]") {
		t.Errorf("expected '[Context]' in result, got %q", result)
	}
	if !strings.Contains(result, "login") {
		t.Errorf("expected 'login' in result, got %q", result)
	}
}
