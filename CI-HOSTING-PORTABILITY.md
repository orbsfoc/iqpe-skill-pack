# CI Hosting Portability (Skill Pack Repo)

## Current host

GitHub Actions is used for early testing.

## Command contract (host-neutral)

Every CI host must preserve these checks:

1. Required baseline docs exist (`README.md`, `CHANGELOG.md`, `OWNERS.md`, `docs/README.md`).
2. Skills root exists (`.github/skills`).
3. Required core skills exist with docs (`.github/skills/*/SKILL.md`).

## Migration rule

When moving away from GitHub, port checks with identical command semantics first, then optimize.
