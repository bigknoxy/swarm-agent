# Swarm Agent — Quickstart

**Get a working AI coding agent in 30 seconds.**

## Install

```bash
curl -sSL https://swarm.sh | sh
```

That's it. The script will:
- Detect your OS (macOS, Linux, Windows/WSL2)
- Install Ollama if missing
- Pull the recommended model (`qwen2.5-coder:32b`)
- Build and install the `swarm` binary
- Create a default config at `~/.swarm/config.json`

## Verify Installation

```bash
swarm --version
# Output: swarm 0.1.0 (built with Go 1.21+)
```

## Your First Task

```bash
swarm "Write a Python function that calculates fibonacci numbers"
```

The agent will:
1. Think about the problem (ReAct loop)
2. Write the code
3. **Verify it works** (runs the code in a sandbox)
4. Return the working solution

## How It Works

Swarm uses a **"Code → Verify → Reflect"** loop:

```
Goal → Orchestrator → LLM (qwen2.5-coder:32b)
                     ↓
               Tools (PythonREPL, FileTool)
                     ↓
               Executor (sandboxed, timeout-protected)
                     ↓
               Judge (verifies output)
                     ↓
               Success! (or Reflector tries again)
```

**Key principle:** Swarm never outputs code it hasn't verified.

## Common Usage

### Custom Model
```bash
swarm --model qwen3.6:35b-a3b "Write a REST API in FastAPI"
```

### Custom Timeout (for complex tasks)
```bash
swarm --timeout 10m "Implement a distributed cache"
```

### Custom Workspace
```bash
swarm --workspace ./my-project "Refactor the authentication module"
```

### Use a Config File
```bash
swarm --config ./my-config.json "Goal"
```

## Configuration

View your config:
```bash
cat ~/.swarm/config.json
```

Edit to customize:
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

Environment variables override config:
```bash
SWARM_MODEL=qwen3.6:35b-a3b swarm "Goal"
```

## Examples

Check out the `examples/` directory for more complex use cases:
- `examples/hello_world.txt` — Simplest possible task
- `examples/fibonacci.py` — Basic Python with verification
- `examples/prime_sieve.py` — Performance checking
- `examples/file_io.py` — FileTool usage
- `examples/learning_demo.py` — LTM learning demo

## Uninstall

```bash
curl -sSL https://swarm.sh | sh -s uninstall
```

Or if you have the repo:
```bash
./scripts/uninstall.sh
```

## Next Steps

1. Read the [README.md](README.md) for full documentation
2. Check the [blueprint.md](blueprint.md) for the development roadmap
3. Join the community (TODO: add Discord/Slack link)

---

**Having issues?** Run with `SWARM_DEBUG=1` for verbose output.
