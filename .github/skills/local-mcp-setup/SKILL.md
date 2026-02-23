---
name: local-mcp-setup
description: Install and configure local MCP runtime for this project using deterministic actions. Use before bootstrap and preflight in local demo mode.
argument-hint: [target_root] [spec_dir]
user-invokable: true
disable-model-invocation: false
---

# Local MCP Setup

Run setup actions:

1. `mcp.action.local_mcp_install`
2. `mcp.action.local_mcp_configure` with `target_root`

Then verify `.vscode/mcp.json` is present in the target repo and proceed to bootstrap + preflight.

Self-service fallback (no `run_action` client path):

`go run ./.github/skills/local-mcp-setup/bootstrap_preflight.go --target-root <target_repo_root_abs_path> --spec-dir <spec_dir>`

Optional override baseline source:

`go run ./.github/skills/local-mcp-setup/bootstrap_preflight.go --target-root <target_repo_root_abs_path> --spec-dir <spec_dir> --corporate-tech-file <path_to_corporate_approved_tech.json>`

This generates:
- `docs/tooling/bootstrap-report.md`
- `docs/tooling/workflow-preflight.json`
- `docs/tooling/spec-tech-detect.json`

`spec-tech-detect.json` merges `SPEC_DIR` detection with the installed corporate approved tech baseline file:
- `./.github/skills/local-mcp-setup/corporate-approved-tech.json`

Use this when MCP server is healthy but current client session cannot invoke `run_action`.

Corporate ADR/tech docs availability through MCP:
- Corporate docs are bundled under `./.github/skills/local-mcp-setup/corporate-docs/`.
- `docs-graph` scans `Docs/` and this corporate-docs path by default.
- Optional override for docs roots: set `DOCS_GRAPH_ROOTS` (comma-separated workspace-relative paths).
