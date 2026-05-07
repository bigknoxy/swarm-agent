package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent_loop/internal/executor"
)

// Workspace represents a strictly isolated directory for the agent
type Workspace struct {
	Root string
}

func NewWorkspace() *Workspace {
	root := os.Getenv("SWARM_WORKSPACE")
	if root == "" {
		tmpDir, _ := os.MkdirTemp(os.TempDir(), "agent_loop_workspace_*")
		root = tmpDir
	}
	return &Workspace{Root: root}
}

// NewWorkspaceWithPath creates a workspace with a specific path.
func NewWorkspaceWithPath(path string) *Workspace {
	if path == "" {
		return NewWorkspace()
	}
	// Expand ~ to home directory
	if len(path) > 0 && path[0] == '~' {
		home, _ := os.UserHomeDir()
		path = filepath.Join(home, path[1:])
	}
	// Create directory if it doesn't exist
	os.MkdirAll(path, 0755)
	return &Workspace{Root: path}
}

// Path joins the root with the target path, ensuring no directory traversal
func (w *Workspace) Path(filename string) string {
	clean := filepath.Clean(filename)
	if strings.HasPrefix(clean, "..") {
		clean = filepath.Base(filename)
	}
	return filepath.Join(w.Root, clean)
}

// WriteFile safely writes content to the workspace
func (w *Workspace) WriteFile(filename string, content string) error {
	path := w.Path(filename)
	rel, err := filepath.Rel(w.Root, path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return fmt.Errorf("workspace: path %q escapes workspace root", filename)
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("workspace mkdir failed: %w", err)
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// ReadFile safely reads content from the workspace
func (w *Workspace) ReadFile(filename string) (string, error) {
	path := w.Path(filename)
	rel, err := filepath.Rel(w.Root, path)
	if err != nil || strings.HasPrefix(rel, "..") || filepath.IsAbs(rel) {
		return "", fmt.Errorf("workspace: path %q escapes workspace root", filename)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("workspace read failed: %w", err)
	}
	return string(content), nil
}

// ListFiles lists all files in the workspace, recursing into subdirectories.
func (w *Workspace) ListFiles() ([]string, error) {
	var files []string
	err := filepath.WalkDir(w.Root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(w.Root, path)
		if err != nil {
			return err
		}
		files = append(files, rel)
		return nil
	})
	return files, err
}

// Clean removes the workspace directory
func (w *Workspace) Clean() {
	os.RemoveAll(w.Root)
}

// --- FileSystemTool ---

type FileTool struct {
	Workspace *Workspace
}

func (f *FileTool) Name() string { return "filesystem" }

func (f *FileTool) Description() string {
	return `Read and write files in the workspace. Action Input must be JSON with an "action" field.
  Write: {"action":"write","path":"index.html","content":"<file content>"}
  Read:  {"action":"read","path":"index.html"}
  List:  {"action":"list"}`
}

func (f *FileTool) Execute(ctx context.Context, arg string) (string, error) {
	// Single quotes never need escaping in JSON; \' is invalid JSON that LLMs sometimes emit.
	arg = strings.ReplaceAll(arg, `\'`, `'`)
	var args map[string]string
	if err := json.Unmarshal([]byte(arg), &args); err != nil {
		return "", fmt.Errorf("invalid arguments JSON: %w", err)
	}

	action, ok := args["action"]
	if !ok {
		return "", fmt.Errorf("missing 'action' argument")
	}

	switch action {
	case "write":
		path, ok := args["path"]
		if !ok {
			return "", fmt.Errorf("missing 'path' argument for write")
		}
		content, ok := args["content"]
		if !ok {
			return "", fmt.Errorf("missing 'content' argument for write")
		}
		if err := f.Workspace.WriteFile(path, content); err != nil {
			return "", err
		}
		return fmt.Sprintf("Successfully wrote %d bytes to %s", len(content), path), nil

	case "read":
		path, ok := args["path"]
		if !ok {
			return "", fmt.Errorf("missing 'path' argument for read")
		}
		content, err := f.Workspace.ReadFile(path)
		if err != nil {
			return "", err
		}
		return content, nil

	case "list":
		files, err := f.Workspace.ListFiles()
		if err != nil {
			return "", err
		}
		return strings.Join(files, "\n"), nil

	default:
		return "", fmt.Errorf("unknown action: %s", action)
	}
}

// --- ShellTool ---

var destructivePatterns = []string{
	"rm -rf /", "rm -rf ~",
	"curl | bash", "curl|bash",
	"wget | sh", "wget|sh",
	"git push",
	"chmod 777",
	":(){:|:&};:",
}

type ShellTool struct {
	Exec      *executor.Executor
	Workspace *Workspace
	Timeout   time.Duration
}

func NewShellTool(exec *executor.Executor, ws *Workspace, timeout time.Duration) *ShellTool {
	return &ShellTool{Exec: exec, Workspace: ws, Timeout: timeout}
}

func (s *ShellTool) Name() string { return "shell" }

func (s *ShellTool) Description() string {
	return `Run a shell command in the workspace directory.
Input: {"command": "npm install"}
Output: stdout+stderr combined with exit code.
Timeout: 5m default. Working directory: workspace root.
Use for: go build, go test, npm install, node app.js, pip install, pytest, cargo build, make.
Do NOT use for: writing files (use filesystem tool), running Python snippets (use python_repl).
Blocked patterns: rm -rf /, curl | bash, git push, wget | sh, chmod 777.`
}

func (s *ShellTool) Execute(ctx context.Context, arg string) (string, error) {
	var args map[string]string
	if err := json.Unmarshal([]byte(arg), &args); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	command, ok := args["command"]
	if !ok {
		return "", fmt.Errorf("missing 'command' field")
	}
	for _, p := range destructivePatterns {
		if strings.Contains(command, p) {
			return "", fmt.Errorf("blocked: command matches destructive pattern %q", p)
		}
	}
	result, err := s.Exec.RunShell(ctx, command, s.Workspace.Root, s.Timeout)
	if err != nil {
		return "", err
	}
	if result.Status == "timeout" {
		return fmt.Sprintf("Command '%s' timed out after %s. Last output:\n%s", command, s.Timeout, result.Stdout), nil
	}
	return result.Stdout, nil
}
