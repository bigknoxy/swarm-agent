package orchestrator

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"agent_loop/internal/executor"
)

func TestNewWorkspace(t *testing.T) {
	// Test with default (temp dir)
	ws := NewWorkspace()
	if ws.Root == "" {
		t.Error("Expected non-empty root")
	}
	if !strings.Contains(ws.Root, "agent_loop_workspace") {
		t.Errorf("Expected workspace path to contain 'agent_loop_workspace', got %q", ws.Root)
	}
}

func TestNewWorkspaceWithPath(t *testing.T) {
	// Test with custom path
	customPath := "/tmp/test_workspace"
	defer os.RemoveAll(customPath)

	ws := NewWorkspaceWithPath(customPath)
	if ws.Root != customPath {
		t.Errorf("Expected root %q, got %q", customPath, ws.Root)
	}

	// Directory should be created
	_, err := os.Stat(customPath)
	if err != nil {
		t.Errorf("Expected directory to be created, got error: %v", err)
	}
}

func TestNewWorkspaceWithPath_Empty(t *testing.T) {
	ws := NewWorkspaceWithPath("")
	// Should fall back to NewWorkspace
	if ws.Root == "" {
		t.Error("Expected non-empty root")
	}
}

func TestNewWorkspaceWithPath_TildeExpansion(t *testing.T) {
	ws := NewWorkspaceWithPath("~/test_workspace_tilde")
	defer os.RemoveAll(ws.Root)

	if !strings.Contains(ws.Root, "test_workspace_tilde") {
		t.Errorf("Expected path to contain 'test_workspace_tilde', got %q", ws.Root)
	}
	// Should not start with ~
	if strings.HasPrefix(ws.Root, "~") {
		t.Errorf("Expected tilde to be expanded, got %q", ws.Root)
	}
}

func TestWorkspace_Path(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_ws")
	defer os.RemoveAll(ws.Root)

	// Test safe path joining
	p := ws.Path("test.txt")
	if !strings.Contains(p, "test.txt") {
		t.Errorf("Expected path to contain filename, got %q", p)
	}
	if !strings.HasPrefix(p, ws.Root) {
		t.Errorf("Expected path to be within workspace, got %q", p)
	}
}

func TestWorkspace_Path_DirectoryTraversal(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_ws")
	defer os.RemoveAll(ws.Root)

	// Test that directory traversal is prevented
	p := ws.Path("../../etc/passwd")
	if strings.Contains(p, "..") {
		t.Errorf("Path should not contain '..', got %q", p)
	}
	if !strings.HasPrefix(p, ws.Root) {
		t.Errorf("Expected path to be within workspace, got %q", p)
	}
}

func TestWorkspace_WriteAndReadFile(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_ws_fileops")
	defer os.RemoveAll(ws.Root)

	// Write file
	content := "Hello, World!"
	err := ws.WriteFile("test.txt", content)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Read file
	read, err := ws.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if read != content {
		t.Errorf("Expected %q, got %q", content, read)
	}
}

func TestWorkspace_ReadFile_NotFound(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_ws_readerr")
	defer os.RemoveAll(ws.Root)

	_, err := ws.ReadFile("nonexistent.txt")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
}

func TestWorkspace_ListFiles(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_ws_list")
	defer os.RemoveAll(ws.Root)

	// Create some files
	ws.WriteFile("file1.txt", "content1")
	ws.WriteFile("file2.txt", "content2")

	files, err := ws.ListFiles()
	if err != nil {
		t.Fatalf("Failed to list files: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, got %d", len(files))
	}
}

func TestWorkspace_Clean(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_ws_clean")
	// Create a file
	ws.WriteFile("test.txt", "content")

	// Clean
	ws.Clean()

	// Directory should be removed
	_, err := os.Stat(ws.Root)
	if !os.IsNotExist(err) {
		t.Errorf("Expected directory to be removed after Clean()")
	}
}

func TestFileTool_Name(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_filetool")
	defer os.RemoveAll(ws.Root)

	tool := &FileTool{Workspace: ws}
	if tool.Name() != "filesystem" {
		t.Errorf("Expected name 'filesystem', got %q", tool.Name())
	}
}

func TestFileTool_Description(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_filetool_desc")
	defer os.RemoveAll(ws.Root)

	tool := &FileTool{Workspace: ws}
	if tool.Description() == "" {
		t.Error("Expected non-empty description")
	}
}

func TestFileTool_Execute_Write(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_filetool_write")
	defer os.RemoveAll(ws.Root)

	tool := &FileTool{Workspace: ws}

	// Write action
	args := `{"action": "write", "path": "test.txt", "content": "Hello!"}`
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Failed to execute write: %v", err)
	}
	if !strings.Contains(result, "Successfully wrote") {
		t.Errorf("Expected success message, got %q", result)
	}

	// Verify file was written
	content, err := ws.ReadFile("test.txt")
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if content != "Hello!" {
		t.Errorf("Expected 'Hello!', got %q", content)
	}
}

func TestFileTool_Execute_Read(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_filetool_read")
	defer os.RemoveAll(ws.Root)

	tool := &FileTool{Workspace: ws}

	// Write a file first
	ws.WriteFile("test.txt", "Content to read")

	// Read action
	args := `{"action": "read", "path": "test.txt"}`
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Failed to execute read: %v", err)
	}
	if result != "Content to read" {
		t.Errorf("Expected 'Content to read', got %q", result)
	}
}

func TestFileTool_Execute_List(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_filetool_list")
	defer os.RemoveAll(ws.Root)

	tool := &FileTool{Workspace: ws}

	// Create files
	ws.WriteFile("a.txt", "a")
	ws.WriteFile("b.txt", "b")

	// List action
	args := `{"action": "list"}`
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Failed to execute list: %v", err)
	}
	if !strings.Contains(result, "a.txt") || !strings.Contains(result, "b.txt") {
		t.Errorf("Expected file names in result, got %q", result)
	}
}

func TestFileTool_Execute_InvalidJSON(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_filetool_json")
	defer os.RemoveAll(ws.Root)

	tool := &FileTool{Workspace: ws}

	_, err := tool.Execute(context.Background(), "not json")
	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

func TestFileTool_Execute_MissingAction(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_filetool_action")
	defer os.RemoveAll(ws.Root)

	tool := &FileTool{Workspace: ws}

	_, err := tool.Execute(context.Background(), `{"path": "test"}`)
	if err == nil {
		t.Error("Expected error for missing action")
	}
}

func TestFileTool_Execute_UnknownAction(t *testing.T) {
	ws := NewWorkspaceWithPath("/tmp/test_filetool_unknown")
	defer os.RemoveAll(ws.Root)

	tool := &FileTool{Workspace: ws}

	_, err := tool.Execute(context.Background(), `{"action": "unknown"}`)
	if err == nil {
		t.Error("Expected error for unknown action")
	}
}

func TestShellTool_Name(t *testing.T) {
	ws := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(2 * time.Second)
	tool := NewShellTool(exec, ws, 5*time.Second)
	if tool.Name() != "shell" {
		t.Errorf("expected name 'shell', got %q", tool.Name())
	}
}

func TestShellTool_Execute_Basic(t *testing.T) {
	ws := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(2 * time.Second)
	tool := NewShellTool(exec, ws, 5*time.Second)

	out, err := tool.Execute(context.Background(), `{"command": "echo hello"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "hello") {
		t.Errorf("expected output to contain 'hello', got %q", out)
	}
}

func TestShellTool_Execute_InvalidJSON(t *testing.T) {
	ws := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(2 * time.Second)
	tool := NewShellTool(exec, ws, 5*time.Second)

	_, err := tool.Execute(context.Background(), `not json`)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
}

func TestShellTool_Execute_MissingCommand(t *testing.T) {
	ws := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(2 * time.Second)
	tool := NewShellTool(exec, ws, 5*time.Second)

	_, err := tool.Execute(context.Background(), `{"action": "list"}`)
	if err == nil {
		t.Fatal("expected error for missing 'command' field, got nil")
	}
}

func TestShellTool_Execute_BlockedPattern(t *testing.T) {
	ws := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(2 * time.Second)
	tool := NewShellTool(exec, ws, 5*time.Second)

	_, err := tool.Execute(context.Background(), `{"command": "rm -rf /"}`)
	if err == nil {
		t.Fatal("expected error for blocked pattern, got nil")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got %q", err.Error())
	}
}

func TestShellTool_Execute_BlockedGitPush(t *testing.T) {
	ws := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(2 * time.Second)
	tool := NewShellTool(exec, ws, 5*time.Second)

	_, err := tool.Execute(context.Background(), `{"command": "git push origin main"}`)
	if err == nil {
		t.Fatal("expected error for git push, got nil")
	}
	if !strings.Contains(err.Error(), "blocked") {
		t.Errorf("expected 'blocked' in error, got %q", err.Error())
	}
}

func TestShellTool_Execute_WorksInWorkspace(t *testing.T) {
	ws := NewWorkspaceWithPath(t.TempDir())
	exec := executor.NewExecutor(2 * time.Second)
	tool := NewShellTool(exec, ws, 5*time.Second)

	testFile := filepath.Join(ws.Root, "test_file.txt")
	if err := os.WriteFile(testFile, []byte("hi"), 0644); err != nil {
		t.Fatalf("setup failed: %v", err)
	}

	out, err := tool.Execute(context.Background(), `{"command": "ls"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "test_file.txt") {
		t.Errorf("expected output to contain 'test_file.txt', got %q", out)
	}
}
