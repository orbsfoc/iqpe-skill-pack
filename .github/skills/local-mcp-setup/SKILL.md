---
name: local-mcp-setup
description: Install and configure local MCP runtime for this project using deterministic actions. Use before bootstrap and preflight in local demo mode.
argument-hint: "target_root spec_dir"
user-invokable: true
disable-model-invocation: false
---

# Local MCP Setup

Run setup actions:

1. `mcp.action.local_mcp_install`
2. `mcp.action.local_mcp_configure` with `target_root`

Then verify `.vscode/mcp.json` is present in the target repo and proceed to bootstrap + preflight.

Binary compatibility note (clean-laptop installs):
- Some installers may place the runtime as `localmcp` instead of `iqpe-localmcp`.
- Fallback bootstrap/preflight supports both names and can normalize unresolved local MCP commands in `.vscode/mcp.json` to a discovered executable path.

Self-service fallback (no `run_action` client path):

`go run ./.github/skills/local-mcp-setup/bootstrap_preflight.go --target-root <target_repo_root_abs_path> --spec-dir <spec_dir>`

Optional override baseline source:

`go run ./.github/skills/local-mcp-setup/bootstrap_preflight.go --target-root <target_repo_root_abs_path> --spec-dir <spec_dir> --corporate-tech-file <path_to_corporate_approved_tech.json>`

This generates:
- `docs/tooling/bootstrap-report.md`
- `docs/tooling/workflow-preflight.json`
- `docs/tooling/spec-tech-detect.json`

Planning behavior resolution (MCP-configurable):

`go run ./.github/skills/local-mcp-setup/cmd/planning_behavior_resolve/main.go --target-root <target_repo_root_abs_path> --out docs/planning-behavior-resolution.md`

Phase precondition check (cross-platform Go checker):

`go run ./.github/skills/local-mcp-setup/cmd/phase_precondition_check/main.go --target-root <target_repo_root_abs_path> --phase 01`

Default profile fallback path is bundled locally:
- `./.github/skills/local-mcp-setup/corporate-docs/planning-behavior-profile.yaml`

`spec-tech-detect.json` merges `SPEC_DIR` detection with the installed corporate approved tech baseline file:
- `./.github/skills/local-mcp-setup/corporate-approved-tech.json`

Use this when MCP server is healthy but current client session cannot invoke `run_action`.

Corporate ADR/tech docs availability through MCP:
- Corporate docs are bundled under `./.github/skills/local-mcp-setup/corporate-docs/`.
- `docs-graph` scans `Docs/` and this corporate-docs path by default.
- Optional override for docs roots: set `DOCS_GRAPH_ROOTS` (comma-separated workspace-relative paths).
