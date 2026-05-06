package watch

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a directory for source file changes and runs a build command.
// When the build fails it calls FixFunc with the error output so the caller can invoke the agent.
type Watcher struct {
	Dir      string
	BuildCmd string        // e.g. "go build ./..."
	Exts     []string      // extensions to watch, e.g. [".go", ".ts", ".py"]
	MaxFixes int           // max agent fix attempts per change event, default 3
	FixFunc  func(ctx context.Context, errorOutput string) error
}

// Run starts watching Dir until ctx is cancelled.
func (w *Watcher) Run(ctx context.Context) error {
	exts := w.Exts
	if len(exts) == 0 {
		exts = []string{".go", ".ts", ".tsx", ".js", ".py", ".rs"}
	}
	maxFixes := w.MaxFixes
	if maxFixes == 0 {
		maxFixes = 3
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("watch: %w", err)
	}
	defer watcher.Close()

	// Walk dir and add all subdirectories
	if err := filepath.Walk(w.Dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && !strings.Contains(path, "node_modules") && !strings.Contains(path, ".git") && !strings.Contains(path, "vendor") {
			return watcher.Add(path)
		}
		return nil
	}); err != nil {
		return fmt.Errorf("watch setup: %w", err)
	}

	fmt.Printf("👀 Watching %s (build: %s)\n", w.Dir, w.BuildCmd)

	var debounce *time.Timer

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			// Only react to write/create events for watched extensions
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			matched := false
			for _, ext := range exts {
				if strings.HasSuffix(event.Name, ext) {
					matched = true
					break
				}
			}
			if !matched {
				continue
			}

			// Debounce: reset timer on each event
			if debounce != nil {
				debounce.Stop()
			}
			debounce = time.AfterFunc(2*time.Second, func() {
				w.handleChange(ctx, maxFixes)
			})
		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
		}
	}
}

func (w *Watcher) handleChange(ctx context.Context, maxFixes int) {
	stdout, stderr, exitOK := w.runBuild()
	if exitOK {
		fmt.Println("✅ Build OK")
		return
	}

	errorOutput := strings.TrimSpace(stdout + "\n" + stderr)
	fmt.Printf("❌ Build failed — invoking agent fix (attempt 1/%d)\n", maxFixes)

	for attempt := 1; attempt <= maxFixes; attempt++ {
		if err := w.FixFunc(ctx, errorOutput); err != nil {
			fmt.Fprintf(os.Stderr, "fix error: %v\n", err)
			return
		}
		// Re-run build to check if fixed
		_, _, exitOK = w.runBuild()
		if exitOK {
			fmt.Printf("✅ Fixed in %d attempt(s)\n", attempt)
			return
		}
		if attempt < maxFixes {
			fmt.Printf("⚠️  Still failing — retrying (attempt %d/%d)\n", attempt+1, maxFixes)
		}
	}
	fmt.Printf("❌ Could not fix after %d attempt(s)\n", maxFixes)
}

func (w *Watcher) runBuild() (stdout, stderr string, ok bool) {
	parts := strings.Fields(w.BuildCmd)
	if len(parts) == 0 {
		return "", "", false
	}
	cmd := exec.Command(parts[0], parts[1:]...)
	cmd.Dir = w.Dir
	var outBuf, errBuf strings.Builder
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err := cmd.Run()
	return outBuf.String(), errBuf.String(), err == nil
}

// DetectBuildCmd returns a sensible default build command for the project in dir.
func DetectBuildCmd(dir string) string {
	checks := []struct{ file, cmd string }{
		{"go.mod", "go build ./..."},
		{"package.json", "npm run build"},
		{"Cargo.toml", "cargo build"},
		{"pyproject.toml", "python3 -m pytest --tb=short -q"},
		{"requirements.txt", "python3 -m pytest --tb=short -q"},
	}
	for _, c := range checks {
		if _, err := os.Stat(filepath.Join(dir, c.file)); err == nil {
			return c.cmd
		}
	}
	return "make"
}
