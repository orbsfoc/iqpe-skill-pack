# iqpe-skill-pack

Repository for reusable agent skills and skill version governance.

## Extracted package contents

- `.github/skills` (versioned skill definitions)
- `docs/README.md` (skill-pack docs index)

## Owns

- Skill definitions
- Skill packaging/release notes
- Skill version index and compatibility notes

## Required standards

- Track skill changes in CHANGELOG.
- Document required MCP actions per skill.
- Keep compatibility matrix with runtime/governance repos.

## CI and hosting

- GitHub Actions is the active CI host for early testing.
- Skill-pack quality checks should remain portable as command contracts.
- Follow `CI-HOSTING-PORTABILITY.md` in this repo after extraction.

## Extraction provenance

- See `EXTRACTION-MANIFEST.md` for source mapping from monorepo paths.
