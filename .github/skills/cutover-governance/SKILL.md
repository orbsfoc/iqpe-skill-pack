---
name: cutover-governance
description: Evaluate and advance wave cutover governance status using deterministic readiness/remediation actions.
argument-hint: [wave] [target_state(optional)]
user-invokable: true
disable-model-invocation: false
---

# Cutover Governance

Use applicable cutover actions:

- `mcp.action.cutover_progress`
- `mcp.action.wave3_readiness`
- `mcp.action.wave3_remediation`
- `mcp.action.wave3_remove_shadows`
- `mcp.action.wave3_closure`
- `mcp.action.wave4_bootstrap`

Report status with clear PASS/FAIL/BLOCKED outcomes.
