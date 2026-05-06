package orchestrator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent_loop/internal/executor"
)

type ProjectContext struct {
	GitStatus      string
	TopFiles       []string
	RelevantFiles  []string
	WorkspaceFiles []string
	FileSnippet    string
	Keywords       []string
	ProjectType    string
	RecentCommits  string
	ConfigSnippet  string
}

func detectProjectType(dir string) string {
	checks := []struct{ file, ptype string }{
		{"go.mod", "go"}, {"package.json", "node"}, {"pyproject.toml", "python"},
		{"requirements.txt", "python"}, {"setup.py", "python"}, {"Cargo.toml", "rust"}, {"Makefile", "make"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(dir, c.file)); err == nil {
			return c.ptype
		}
	}
	return "unknown"
}

func readConfigSnippet(projectType, dir string) string {
	var candidates []string
	switch projectType {
	case "go":
		candidates = []string{"go.mod"}
	case "node":
		candidates = []string{"package.json"}
	case "python":
		candidates = []string{"pyproject.toml", "requirements.txt", "setup.py"}
	case "rust":
		candidates = []string{"Cargo.toml"}
	default:
		return ""
	}
	for _, name := range candidates {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			continue
		}
		lines := strings.Split(string(content), "\n")
		if len(lines) > 30 {
			lines = lines[:30]
		}
		return fmt.Sprintf("(%s)\n%s", name, strings.Join(lines, "\n"))
	}
	return ""
}

func BuildContext(goal, cwd string, exec *executor.Executor, ws *Workspace) (*ProjectContext, error) {
	ctx := context.Background()
	pc := &ProjectContext{}

	// GitStatus — skip gracefully if not a git repo
	if res, err := exec.RunShell(ctx, "git status --short", cwd, 5*time.Second); err == nil && res.Status == "success" {
		pc.GitStatus = strings.TrimSpace(res.Stdout)
	}

	// Project type detection
	pc.ProjectType = detectProjectType(cwd)
	pc.ConfigSnippet = readConfigSnippet(pc.ProjectType, cwd)

	// Recent git log
	if res, err := exec.RunShell(ctx, "git log --oneline -5", cwd, 5*time.Second); err == nil && res.Status == "success" {
		pc.RecentCommits = strings.TrimSpace(res.Stdout)
	}

	// TopFiles — find top-level files, exclude build artifacts and vendor dirs
	excludePatterns := []string{".git/", "node_modules/", "__pycache__/", ".venv/", "dist/", "build/", "vendor/", ".DS_Store"}
	if res, err := exec.RunShell(ctx, "find . -maxdepth 2 -type f", cwd, 5*time.Second); err == nil && res.Status == "success" {
		lines := strings.Split(strings.TrimSpace(res.Stdout), "\n")
		var filtered []string
		for _, f := range lines {
			f = strings.TrimSpace(f)
			if f == "" {
				continue
			}
			skip := false
			for _, p := range excludePatterns {
				if strings.Contains(f, p) {
					skip = true
					break
				}
			}
			if !skip {
				filtered = append(filtered, f)
			}
		}
		if len(filtered) > 50 {
			filtered = filtered[:50]
		}
		pc.TopFiles = filtered
	}

	// Keyword extraction — simple noun filter, no LLM call
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "to": true, "for": true, "in": true, "of": true, "that": true,
		"with": true, "and": true, "or": true, "is": true, "are": true, "on": true, "at": true, "by": true,
		"from": true, "it": true, "its": true, "this": true, "add": true, "make": true, "write": true,
		"create": true, "fix": true, "update": true, "get": true, "set": true, "run": true, "use": true,
	}
	for _, w := range strings.Fields(goal) {
		lower := strings.ToLower(w)
		if len(w) >= 4 && !stopWords[lower] {
			pc.Keywords = append(pc.Keywords, lower)
		}
	}

	// RelevantFiles — files matching goal keywords
	if len(pc.Keywords) > 0 {
		seen := make(map[string]bool)
		for _, kw := range pc.Keywords {
			if len(pc.RelevantFiles) >= 10 {
				break
			}
			for _, f := range pc.TopFiles {
				if !seen[f] && strings.Contains(strings.ToLower(f), kw) {
					seen[f] = true
					pc.RelevantFiles = append(pc.RelevantFiles, f)
				}
			}
		}
	}

	// FileSnippet — up to 3 keyword-matched files, 80 lines each
	var snippetParts []string
	for i, rel := range pc.RelevantFiles {
		if i >= 3 {
			break
		}
		relPath := strings.TrimPrefix(rel, "./")
		if content, err := os.ReadFile(filepath.Join(cwd, relPath)); err == nil {
			lines := strings.Split(string(content), "\n")
			if len(lines) > 80 {
				lines = lines[:80]
			}
			snippetParts = append(snippetParts, fmt.Sprintf("=== %s ===\n%s", relPath, strings.Join(lines, "\n")))
		}
	}
	if len(snippetParts) > 0 {
		pc.FileSnippet = strings.Join(snippetParts, "\n\n")
	}

	// WorkspaceFiles — files already written in the workspace
	if files, err := ws.ListFiles(); err == nil {
		pc.WorkspaceFiles = files
	}

	return pc, nil
}

func (pc *ProjectContext) Format() string {
	var sb strings.Builder
	sb.WriteString("## Project Context\n")

	if pc.ProjectType != "" && pc.ProjectType != "unknown" {
		sb.WriteString(fmt.Sprintf("Project type: %s\n\n", pc.ProjectType))
	}

	if pc.RecentCommits != "" {
		sb.WriteString("Recent commits:\n")
		sb.WriteString(pc.RecentCommits)
		sb.WriteString("\n\n")
	}

	if pc.ConfigSnippet != "" {
		sb.WriteString("Config:\n")
		sb.WriteString(pc.ConfigSnippet)
		sb.WriteString("\n\n")
	}

	if pc.GitStatus != "" {
		sb.WriteString("Git status:\n")
		sb.WriteString(pc.GitStatus)
		sb.WriteString("\n\n")
	}

	if len(pc.TopFiles) > 0 {
		sb.WriteString("Top files:\n")
		for _, f := range pc.TopFiles {
			sb.WriteString(strings.TrimPrefix(f, "./"))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if len(pc.RelevantFiles) > 0 {
		sb.WriteString("Relevant files (matching goal keywords):\n")
		for _, f := range pc.RelevantFiles {
			sb.WriteString(strings.TrimPrefix(f, "./"))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	if pc.FileSnippet != "" {
		sb.WriteString("File contents:\n")
		sb.WriteString(pc.FileSnippet)
		sb.WriteString("\n\n")
	}

	if len(pc.WorkspaceFiles) > 0 {
		sb.WriteString("Workspace files:\n")
		for _, f := range pc.WorkspaceFiles {
			sb.WriteString(f)
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// ContextScanSummary returns a one-line summary for the CLI output.
func ContextScanSummary(pc *ProjectContext) string {
	if len(pc.TopFiles) == 0 {
		return ""
	}
	if len(pc.RelevantFiles) == 0 {
		return fmt.Sprintf("[Context] %d files scanned", len(pc.TopFiles))
	}
	firstKw := pc.Keywords[0]
	return fmt.Sprintf("[Context] %d files, %d matches for '%s'", len(pc.TopFiles), len(pc.RelevantFiles), firstKw)
}
