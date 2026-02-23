---
name: template-access
description: Retrieve versioned templates from MCP registries by name and optional version. Use when creating gates, evidence blocks, and provenance artifacts.
argument-hint: [template_name] [template_version(optional)]
user-invokable: true
disable-model-invocation: false
---

# Template Access

Use MCP template endpoints/actions:

- `list_templates` / `mcp.action.template_list`
- `get_template` / `mcp.action.template_get`

If version is omitted, use latest.
