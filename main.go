package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"agent_loop/internal/config"
	"agent_loop/internal/executor"
	"agent_loop/internal/judge"
	"agent_loop/internal/llm"
	"agent_loop/internal/orchestrator"
	"agent_loop/internal/roles"
	"agent_loop/internal/watch"
)


func main() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "swarm — local AI coding agent\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  swarm \"<goal>\" [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  swarm \"write fibonacci.py and verify it works\"\n")
		fmt.Fprintf(os.Stderr, "  swarm --role architect \"design a rate-limiting system\"\n")
		fmt.Fprintf(os.Stderr, "  swarm --role developer --verbose \"implement the auth module\"\n")
		fmt.Fprintf(os.Stderr, "  swarm --project /path/to/repo \"refactor the auth module\"\n")
		fmt.Fprintf(os.Stderr, "  swarm --json \"write hello.py\" | jq .\n\n")
		fmt.Fprintf(os.Stderr, "Fix mode (pipe compiler errors):\n")
		fmt.Fprintf(os.Stderr, "  go build ./... 2>&1 | swarm fix\n")
		fmt.Fprintf(os.Stderr, "  npm run build 2>&1 | swarm fix\n")
		fmt.Fprintf(os.Stderr, "  swarm fix < errors.txt\n\n")
		fmt.Fprintf(os.Stderr, "Watch mode (auto-fix on save):\n")
		fmt.Fprintf(os.Stderr, "  swarm --watch src/\n")
		fmt.Fprintf(os.Stderr, "  swarm --watch . --build-cmd \"go build ./...\"\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	var fixMode bool
	var fixGoal string

	// Detect 'fix' subcommand before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "fix" {
		fixMode = true
		os.Args = append(os.Args[:1], os.Args[2:]...)
	}

	configPath := flag.String("config", "", "Path to config file (default: ~/.swarm/config.json)")
	modelFlag := flag.String("model", "", "Override model (e.g., qwen3.6:35b-a3b)")
	roleFlag := flag.String("role", "", "Override role (architect, developer, utility)")
	timeoutFlag := flag.Duration("timeout", 0, "Override timeout (e.g., 5m)")
	workspaceFlag := flag.String("workspace", "", "Override workspace path")
	projectFlag := flag.String("project", "", "Project directory to scan for context (default: current directory)")
	watchFlag := flag.String("watch", "", "Directory to watch for changes and auto-fix on build failure")
	buildCmd := flag.String("build-cmd", "", "Build command to run in watch mode (default: auto-detected)")
	versionFlag := flag.Bool("version", false, "Print version and exit")
	verboseFlag := flag.Bool("verbose", false, "Show full LLM prompts/responses")
	jsonFlag := flag.Bool("json", false, "Machine-readable JSON output")
	quietFlag := flag.Bool("quiet", false, "Suppress non-essential output")
	flag.Parse()

	if *versionFlag {
		fmt.Println("swarm 0.1.0 (built with Go 1.21+)")
		os.Exit(0)
	}

	if len(flag.Args()) == 0 && !fixMode {
		flag.Usage()
		os.Exit(1)
	}

	// fix mode: read stdin and build goal from compiler errors
	if fixMode {
		stdinBytes, err := io.ReadAll(os.Stdin)
		if err != nil || len(stdinBytes) == 0 {
			fmt.Fprintln(os.Stderr, "swarm fix: no input on stdin. Usage: go build 2>&1 | swarm fix")
			os.Exit(1)
		}
		errorText := strings.TrimSpace(string(stdinBytes))

		// Detect build tool from error format to add verification hint
		verifyHint := ""
		switch {
		case strings.Contains(errorText, ".go:") || strings.Contains(errorText, "go build"):
			verifyHint = " After fixing, run 'go build ./...' to verify."
		case strings.Contains(errorText, "npm") || strings.Contains(errorText, "node_modules"):
			verifyHint = " After fixing, run 'npm run build' to verify."
		case strings.Contains(errorText, "Traceback") || strings.Contains(errorText, "SyntaxError"):
			verifyHint = " After fixing, run 'python3 -m pytest' or re-run the script to verify."
		case strings.Contains(errorText, "error[E") || strings.Contains(errorText, "cargo"):
			verifyHint = " After fixing, run 'cargo build' to verify."
		}

		fixGoal = "Fix these build/compiler errors in the source files:\n\n" + errorText + "\n\nFix the errors in the relevant source files." + verifyHint
	}

	// Create CLI output handler
	cli := NewCLIOutput(*verboseFlag, *jsonFlag, *quietFlag)

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		cli.PrintFailure(fmt.Sprintf("Warning: %v", err))
		cfg = config.DefaultConfig()
	}

	// Override with CLI flags
	if *modelFlag != "" {
		cfg.DefaultModel = *modelFlag
	}
	if *timeoutFlag != 0 {
		cfg.Timeout = *timeoutFlag
	}
	if *workspaceFlag != "" {
		cfg.Workspace = *workspaceFlag
	}

	// In fix mode, default workspace to CWD so writes land in the real project
	if fixMode && cfg.Workspace == "" {
		if cwd, err := os.Getwd(); err == nil {
			cfg.Workspace = cwd
		}
	}

	projectDir := *projectFlag
	if projectDir == "" {
		if cwd, err := os.Getwd(); err == nil {
			projectDir = cwd
		}
	}

	// Setup
	exec := executor.NewExecutor(cfg.Timeout)
	j := judge.NewJudge()
	workspace := orchestrator.NewWorkspaceWithPath(cfg.Workspace)
	fileTool := &orchestrator.FileTool{Workspace: workspace}
	shellTimeout := cfg.ShellTimeout
	if shellTimeout == 0 {
		shellTimeout = 5 * time.Minute
	}
	shellTool := orchestrator.NewShellTool(exec, workspace, shellTimeout)

	// Config for orchestrator
	orchCfg := orchestrator.Config{MaxAttempts: cfg.MaxAttempts, Timeout: cfg.Timeout}

	// Create LLM client
	llmClient := llm.NewClient(llm.NewOllamaProvider(cfg.OlamaURL, cfg.DefaultModel))

	// Create orchestrator
	orch := orchestrator.NewOrchestrator(llmClient, exec, j, workspace, projectDir, orchCfg, cli)
	orch.RegisterTool(fileTool)
	orch.RegisterTool(shellTool)

	// Watch mode
	if *watchFlag != "" {
		watchDir := *watchFlag
		if !filepath.IsAbs(watchDir) {
			if cwd, err := os.Getwd(); err == nil {
				watchDir = filepath.Join(cwd, watchDir)
			}
		}
		cmd := *buildCmd
		if cmd == "" {
			cmd = watch.DetectBuildCmd(watchDir)
		}
		w := &watch.Watcher{
			Dir:      watchDir,
			BuildCmd: cmd,
			FixFunc: func(ctx context.Context, errorOutput string) error {
				fixGoal := "Fix these build/compiler errors in the source files:\n\n" + errorOutput + "\n\nFix the errors in the relevant source files."
				_, err := orch.SolveReAct(ctx, fixGoal)
				return err
			},
		}
		ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer stop()
		if err := w.Run(ctx); err != nil {
			cli.PrintFailure(fmt.Sprintf("Watch error: %v", err))
			os.Exit(1)
		}
		os.Exit(0)
	}

	var goal string
	if fixMode {
		goal = fixGoal
	} else {
		goal = flag.Args()[0]
	}

	if *roleFlag != "" {
		roleCfg, err := roles.GetRoleConfig(*roleFlag)
		if err != nil {
			cli.PrintFailure(fmt.Sprintf("Error: %v", err))
			os.Exit(1)
		}
		// You can extend this to pass prompt/temp to LLM client or Orchestrator
		// For now, we use it as a way to verify the role system is working.
		cli.PrintStart(fmt.Sprintf("[Role: %s] %s", roleCfg.Name, goal))
	} else {
		cli.PrintStart(goal)
	}

	// Execute ReAct Loop
	ctx := context.Background()
	result, err := orch.SolveReAct(ctx, goal)
	if err != nil {
		cli.PrintFailure(fmt.Sprintf("Agent failed: %v", err))
		if *jsonFlag {
			fmt.Printf(`{"event": "error", "message": "%v"}\n`, err)
		}
		return
	}

	cli.PrintSuccess("Success! Final Solution:")
	if *jsonFlag {
		fmt.Printf(`{"event": "success", "solution": "%s", "workspace": "%s"}\n`, result, workspace.Root)
	} else {
		fmt.Printf("✅ Success! Final Solution:\n%s\n", result)
		if files, err := workspace.ListFiles(); err == nil && len(files) > 0 {
			fmt.Printf("✅ Files in workspace (%s):\n", workspace.Root)
			for _, f := range files {
				if f != "session_memory.json" {
					fmt.Printf("   %s\n", f)
				}
			}
		}
	}
}
