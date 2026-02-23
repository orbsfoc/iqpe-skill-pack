---
name: project-bootstrap
description: Bootstrap workflow prompts and baseline project artifacts for a fresh delivery run. Use when starting a new product/demo repository.
argument-hint: [target_root] [spec_dir(optional)]
user-invokable: true
disable-model-invocation: false
---

# Project Bootstrap

Use deterministic MCP actions to initialize workflow execution:

1. Run `mcp.action.bootstrap_workflow_pack` with `target_root` (and optional `spec_dir`).
2. Verify `docs/tooling/bootstrap-report.md` exists.
3. Continue with preflight checks before phase execution.

If actions cannot be invoked in this client session, use another MCP-capable client connected to the same servers/skills and record that in evidence.
