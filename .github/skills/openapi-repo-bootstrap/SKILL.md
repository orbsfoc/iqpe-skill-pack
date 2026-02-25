---
name: openapi-repo-bootstrap
description: Create a dedicated OpenAPI contract repository only when planning approves a create action and the repo does not already exist.
argument-hint: [target_root] [repo_path(optional)] [repo_plan_file(optional)]
user-invokable: true
disable-model-invocation: false
---

# OpenAPI Repo Bootstrap

Use deterministic action:

1. Run `mcp.action.bootstrap_openapi_repo_if_missing` with `target_root`.
2. Default target path is `repos/openapi-contracts` (override with `repo_path`).
3. Creation gate is enforced:
   - `docs/plans/planning-signoff.md` must contain `Approval Status: APPROVED`.
   - `docs/plans/repo-change-plan.md` must contain a matching `create` action for the target repo path.
4. Behavior:
   - If repo exists: returns `PASS` with `created=false`.
   - If missing and approved `create` action exists: creates minimal OpenAPI contract repo baseline.
   - If plan/signoff conditions are missing: returns `BLOCKED`.

Created baseline (when missing):
- `README.md`
- `CHANGELOG.md`
- `openapi/openapi.yaml`
- `docs/plans/`
- `docs/current-state/`
- `docs/diagrams/`
- `docs/handoffs/`
- `docs/tooling/`
