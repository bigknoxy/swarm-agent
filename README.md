# Swarm Agent Loop

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?logo=go)](https://golang.org)
[![Ollama](https://img.shields.io/badge/Ollama-qwen2.5--coder%3A32b-000000?logo=ollama)](https://ollama.com)

**An AI coding agent that verifies its own work, learns from mistakes, and gets better over time.**

---

## Quickstart

Install in 30 seconds:

```bash
curl -sSL https://swarm.sh | sh
```

Run your first task:

```bash
swarm "Write a Python function that calculates fibonacci numbers"
```

That's it. The agent will think, code, verify, and return working code.

**Having issues?** See the full [QUICKSTART.md](QUICKSTART.md).

---

## Table of Contents

- [What is Swarm?](#what-is-swarm)
- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Architecture](#architecture)
- [Development](#development)
- [Testing](#testing)
- [Uninstall](#uninstall)

---

## What is Swarm?

Swarm is a local AI coding agent built on the **ReAct+TDD** architecture. Unlike linear pipelines, Swarm implements a **"Code → Verify → Reflect"** loop that ensures the agent learns from mistakes and uses tooling to verify its own logic.

### Key Features

- **Verified Output**: Never outputs code it hasn't proven correct through execution
- **Self-Learning**: Long-Term Memory (LTM) persists lessons across sessions
- **Tool Ecosystem**: Python REPL, Performance Checker, FileTool
- **Role System**: Architect → Developer → Utility workflow with specialized models
- **Zero-Config**: Works out of the box, highly customizable when needed

---

## Installation

### One-Line Install (Recommended)

```bash
curl -sSL https://swarm.sh | sh
```

The script will:
- Detect your OS (macOS, Linux, Windows/WSL2)
- Install Ollama if missing
- Pull the recommended model (`qwen2.5-coder:32b`)
- Build and install the `swarm` binary
- Create default config at `~/.swarm/config.json`

### Manual Install

```bash
# Prerequisites: Go 1.21+, Ollama running locally
git clone https://github.com/yourusername/swarm2.git
cd swarm2/agent_loop
go build -o swarm .
sudo mv swarm /usr/local/bin/
```

### Prerequisites

- **Go 1.21+** (for building)
- **Ollama** running at `http://localhost:11434`
- **Models**: `qwen2.5-coder:32b` (recommended), `qwen3.6:35b-a3b`, `qwen3.5`

Pull models:
```bash
ollama pull qwen2.5-coder:32b
ollama pull qwen3.6:35b-a3b
```

---

## Usage

### Basic Usage

```bash
swarm "Your goal here"
```

### Examples

```bash
# Simple function
swarm "Write a Python function to check if a number is prime"

# With verification
swarm "Create a sorting algorithm and verify it works for 1000 random numbers"

# File I/O
swarm "Write a script that reads a CSV and calculates summary statistics"

# Complex task with custom timeout
swarm --timeout 10m "Implement a Redis-backed cache with LRU eviction"
```

### CLI Flags

```bash
swarm --help
# Usage:
#   --config     Path to config file (default: ~/.swarm/config.json)
#   --model      Override model (e.g., qwen3.6:35b-a3b)
#   --timeout    Override timeout (e.g., 5m)
#   --workspace  Override workspace path
#   --version    Print version and exit
```

---

## Configuration

Swarm is highly customizable via config file, environment variables, or CLI flags.

### Config File

Location: `~/.swarm/config.json`

```json
{
  "ollama_url": "http://localhost:11434/v1",
  "default_model": "qwen2.5-coder:32b",
  "timeout": "5m",
  "max_attempts": 5,
  "workspace": "~/.swarm/workspace",
  "ltm_enabled": true,
  "roles": {
    "architect": "qwen3.6:35b-a3b",
    "developer": "qwen2.5-coder:32b",
    "utility": "qwen3.5"
  }
}
```

### Environment Variables

```bash
export SWARM_MODEL="qwen3.6:35b-a3b"
export SWARM_OLLAMA_URL="http://custom:11434/v1"
export SWARM_TIMEOUT="10m"
export SWARM_WORKSPACE="/tmp/custom-workspace"
export SWARM_MAX_ATTEMPTS="10"
```

### Precedence

CLI flags > Environment variables > Config file > Defaults

---

## Architecture

```
main.go → Orchestrator → LLM (Ollama)
                   ↓
             ToolManager → PythonREPL, PerformanceChecker, FileTool
                   ↓
             Executor (sandboxed execution)
                   ↓
             Judge (verifies output)
                   ↓
             Reflector (root cause analysis)
                   ↓
             SessionMemory → LTM (knowledge_manager.py)
```

### Core Components

1. **Orchestrator**: State machine managing ReAct/TDD loop flow
2. **Executor**: Strict process management with timeouts (2s per step, 5m for TDD)
3. **Judge**: Evaluates execution output against expectations
4. **Reflector**: Analyzes why verification failed, extracts lessons
5. **SessionMemory**: Persistent state with hypotheses, lessons, ReAct history
6. **Tools**: PythonREPL, PerformanceChecker, FileTool

---

## Development

### Building from Source

```bash
cd agent_loop
go build -o swarm .
```

### Running Tests

```bash
# All tests
go test ./...

# Specific package
go test ./internal/config/... -v

# With coverage
go test -cover ./...
```

### Project Structure

```
agent_loop/
├── main.go                    # Entry point
├── internal/
│   ├── config/               # Configuration system
│   ├── executor/             # Sandboxed execution
│   ├── judge/                # Output verification
│   ├── llm/                  # LLM client (Ollama)
│   ├── orchestrator/         # ReAct+TDD loop
│   └── agent/                # Agent tools
├── scripts/
│   ├── install.sh            # One-line install
│   ├── uninstall.sh          # Clean uninstall
│   └── test_install.sh       # Install script tests
├── examples/                 # Example goals
├── blueprint.md              # Development roadmap
├── QUICKSTART.md             # Quick start guide
└── README.md                 # This file
```

---

## Testing

We follow the principle: **"We do NOT assume, we verify."**

### Test Coverage

- **Config package**: 6 tests (load from file, env vars, CLI flags)
- **Executor**: Timeout handling, sandboxing
- **Judge**: Output verification logic
- **Orchestrator**: ReAct loop, TDD mode
- **Install scripts**: 9 tests (cross-platform compatibility)

Run all tests:
```bash
go test ./... -v
```

---

## Uninstall

```bash
# Option 1: Using the script
curl -sSL https://swarm.sh | sh -s uninstall

# Option 2: From source
cd swarm2/agent_loop
./scripts/uninstall.sh
```

The uninstall script will:
- Remove the `swarm` binary
- Ask if you want to keep config and workspace
- Clean up all traces (if you choose)

---

## Examples

Check the `examples/` directory for more complex use cases:
- `examples/hello_world.txt` — Simplest possible task
- `examples/fibonacci.py` — Basic Python with verification
- `examples/prime_sieve.py` — Performance checking
- `examples/file_io.py` — FileTool usage
- `examples/learning_demo.py` — LTM learning demo

---

## Contributing

1. Read the [blueprint.md](blueprint.md) to understand the roadmap
2. Check [QUICKSTART.md](QUICKSTART.md) for development setup
3. Run tests before submitting: `go test ./...`
4. Follow the "Code → Verify → Reflect" principle in your contributions

---

## License

[Add your license here]

---

## Acknowledgments

- Built on [@mariozechner/pi-coding-agent](https://github.com/mariozechner/pi-coding-agent)
- Inspired by the ReAct paper and Test-Driven Development practices
