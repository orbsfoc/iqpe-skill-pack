---
name: skill-version-check
description: Validate required skill versions before role execution. Use during provisioning and gate initialization.
argument-hint: [skill_id] [expected_version(optional)]
user-invokable: true
disable-model-invocation: false
---

# Skill Version Check

1. Run `list_skill_versions` or `mcp.action.skill_version_list`.
2. Run `check_skill_version` or `mcp.action.skill_version_check` with:
   - `skill_id`
   - `expected_version` (optional)
3. Record results in MCP evidence.
