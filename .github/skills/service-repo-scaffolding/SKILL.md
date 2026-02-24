---
name: service-repo-scaffolding
description: Scaffold a multi-repo service workspace (Go module, Go app, TS/React app, demo compose) with baseline docs structure and naming ADR.
argument-hint: [target_root] [workspace_dir(optional)]
user-invokable: true
disable-model-invocation: false
---

# Service Repo Scaffolding

Use deterministic action:

1. Run `mcp.action.scaffold_service_workspace` with `target_root`.
2. Confirm `repos/` exists with starter repos:
   - `go-module-service`
   - `go-application-service`
   - `ts-react-service`
   - `demo-compose`
3. Confirm naming ADR exists at `docs/adr/ADR-0001-repo-naming-conventions.md`.
4. Confirm governance baseline artifacts exist:
   - `docs/data-architecture-decision.md`
   - `docs/handoffs/routing-matrix.md`
   - `docs/integration/compose-mode-decision.md`

Expected baseline in each starter repo:
- `README.md`
- `CHANGELOG.md`
- `docs/plans/`
- `docs/current-state/`
- `docs/diagrams/`
- `docs/tooling/go-bin-convention.md`
