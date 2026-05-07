# Swarm TODO

Deferred items from CEO plan 2026-05-06 (see `~/.gstack/projects/agent_loop/ceo-plans/2026-05-06-swarm-3tracks.md`).

## Deferred

### --dry-run mode
`swarm fix --dry-run` shows what would be changed without writing files.
Use case: CI audit, PR preview.
Promote after fix mode is battle-tested (used in production builds, no regressions over a sprint).

### SWARM_FIX_WORKSPACE_CWD
Env var to opt fix mode into an isolated workspace instead of CWD.
Note: currently fix mode always writes to CWD. The Phase 1 path escape guard in `workspace.WriteFile` is independent and should be implemented first.
