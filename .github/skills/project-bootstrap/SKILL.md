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
3. Run `mcp.action.scaffold_service_workspace` to create `repos/` workspace scaffolds.
4. Verify `docs/adr/ADR-0001-repo-naming-conventions.md` exists.
5. Verify governance baseline artifacts exist:
	- `docs/data-architecture-decision.md`
	- `docs/handoffs/routing-matrix.md`
	- `docs/integration/compose-mode-decision.md`
6. Run `mcp.action.context_promotion_publish` with shared repo roots to publish reusable context automatically.
7. Continue with preflight checks before phase execution.

Optional after planning signoff:
- Run `mcp.action.bootstrap_openapi_repo_if_missing` when planning calls for a dedicated OpenAPI contract repository.
- The action creates the target repo only when:
	- `docs/plans/planning-signoff.md` is `APPROVED`, and
	- `docs/plans/repo-change-plan.md` contains a matching `create` action for the target repo path.
- If repo already exists, action returns `PASS` without creating anything.

Service workspace scaffold initializes only the workspace boundary and governance/planning artifacts.

Automated context promotion action:
- `mcp.action.context_promotion_publish`
- Required args for non-manual publish:
	- `architecture_repo_root`
	- `catalog_repo_root`
	- optional `project_slug`

If actions cannot be invoked in this client session, use another MCP-capable client connected to the same servers/skills and record that in evidence.
