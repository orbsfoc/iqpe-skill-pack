---
name: workflow-preflight-check
description: Validate target repo MCP readiness and SPEC_DIR before phase execution. Use after bootstrap and before phase 01.
argument-hint: [target_root] [spec_dir]
user-invokable: true
disable-model-invocation: false
---

# Workflow Preflight Check

1. Run `mcp.action.workflow_preflight_check` with:
   - `target_root`
   - `spec_dir`
2. Confirm `docs/tooling/workflow-preflight.json` exists.
3. Require `status: PASS` before proceeding.

If preflight is not PASS, keep workflow status `BLOCKED`.
