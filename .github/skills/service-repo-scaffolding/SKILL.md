---
name: service-repo-scaffolding
description: Initialize an empty multi-repo workspace boundary, then materialize repositories from approved repo planning actions.
argument-hint: [target_root] [workspace_dir(optional)] [repo_plan_file(optional)]
user-invokable: true
disable-model-invocation: false
---

# Service Repo Scaffolding

Use deterministic actions:

1. Run `mcp.action.scaffold_service_workspace` with `target_root`.
2. Confirm `repos/` exists as an empty workspace boundary (no starter repos created automatically).
3. Confirm naming ADR exists at `docs/adr/ADR-0001-repo-naming-conventions.md`.
4. Confirm governance baseline artifacts exist:
   - `docs/data-architecture-decision.md`
   - `docs/handoffs/routing-matrix.md`
   - `docs/plans/index.md`
   - `docs/plans/planning-signoff.md`

After planning signoff (`Approval Status: APPROVED`):
5. Run `mcp.action.materialize_repos_from_plan`.
6. Behavior:
   - `create` actions: create target repo if missing and scaffold baseline docs.
   - `update` actions: do not create; require repo to already exist.
7. If any `update` target is missing, status is `BLOCKED` until corrected in plan or repository state.
