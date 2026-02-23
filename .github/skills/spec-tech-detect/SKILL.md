---
name: spec-tech-detect
description: Detect backend, frontend, database, and migration decisions from SPEC_DIR before declaring unresolved technology constraints.
argument-hint: [spec_dir]
user-invokable: true
disable-model-invocation: false
---

# Spec Technology Detection

1. Run `mcp.action.spec_tech_detect` with `spec_dir` and target output path.
2. Review `docs/tooling/spec-tech-detect.json`.
3. Materialize detected decisions into technology constraints and ADR artifacts.

Do not declare core TC items unresolved until this check is executed.
