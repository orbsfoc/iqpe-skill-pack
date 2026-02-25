package main

import (
	"errors"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type scaffoldResult struct {
	Status       string   `json:"status"`
	Workspace    string   `json:"workspace"`
	CreatedDirs  []string `json:"created_dirs"`
	CreatedFiles []string `json:"created_files"`
	Issues       []string `json:"issues,omitempty"`
}

type repoPlanRow struct {
	Action     string
	TargetRepo string
}

func main() {
	targetRoot := flag.String("target-root", "", "target repository root (defaults to cwd)")
	workspaceDir := flag.String("workspace-dir", "repos", "relative workspace directory for service repos")
	repoPlanFile := flag.String("repo-plan-file", "", "optional repo change plan markdown path (relative to target root). when set, create missing repos for create-actions and validate update-actions")
	flag.Parse()

	root := strings.TrimSpace(*targetRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			printBlocked("unable to determine working directory")
			return
		}
		root = cwd
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		printBlocked("invalid target root")
		return
	}

	workspaceRel := strings.TrimSpace(*workspaceDir)
	if workspaceRel == "" {
		printBlocked("workspace-dir cannot be empty")
		return
	}
	workspacePath := filepath.Join(absRoot, filepath.FromSlash(workspaceRel))

	result := scaffoldResult{Status: "PASS", Workspace: filepath.ToSlash(workspacePath)}

	mkDir := func(path string) {
		if info, statErr := os.Stat(path); statErr == nil && info.IsDir() {
			return
		}
		if err := os.MkdirAll(path, 0o755); err == nil {
			result.CreatedDirs = append(result.CreatedDirs, filepath.ToSlash(path))
		}
	}

	writeIfMissing := func(path, content string) {
		if _, statErr := os.Stat(path); statErr == nil {
			return
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err == nil {
			result.CreatedFiles = append(result.CreatedFiles, filepath.ToSlash(path))
		}
	}

	mkDir(workspacePath)
	writeIfMissing(filepath.Join(workspacePath, ".gitkeep"), "")
	writeIfMissing(filepath.Join(workspacePath, "README.md"), workspaceReadme())
	scaffoldNamingADR(absRoot, writeIfMissing)
	scaffoldRun5GovernanceArtifacts(absRoot, writeIfMissing)

	planFile := strings.TrimSpace(*repoPlanFile)
	if planFile != "" {
		rows, parseErr := loadRepoPlanRows(absRoot, planFile)
		if parseErr != nil {
			result.Status = "BLOCKED"
			result.Issues = append(result.Issues, parseErr.Error())
		} else {
			issues := applyRepoPlan(absRoot, rows, mkDir, writeIfMissing)
			if len(issues) > 0 {
				result.Status = "BLOCKED"
				result.Issues = append(result.Issues, issues...)
			}
		}
	}

	data, _ := json.Marshal(result)
	fmt.Println(string(data))
}

func loadRepoPlanRows(root, planFile string) ([]repoPlanRow, error) {
	planPath := filepath.Join(root, filepath.FromSlash(planFile))
	data, err := os.ReadFile(planPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read repo plan file: %s", filepath.ToSlash(planPath))
	}

	lines := strings.Split(string(data), "\n")
	rows := make([]repoPlanRow, 0)
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "|") || strings.Count(line, "|") < 4 {
			continue
		}
		if strings.Contains(line, "Repo Action") || strings.Contains(line, "---") {
			continue
		}
		parts := strings.Split(line, "|")
		if len(parts) < 6 {
			continue
		}
		action := strings.ToLower(strings.TrimSpace(parts[3]))
		targetRepo := strings.TrimSpace(parts[4])
		if action == "" || targetRepo == "" {
			continue
		}
		rows = append(rows, repoPlanRow{Action: action, TargetRepo: targetRepo})
	}

	if len(rows) == 0 {
		return nil, errors.New("no repo action rows found in repo plan file")
	}

	return rows, nil
}

func applyRepoPlan(root string, rows []repoPlanRow, mkDir func(string), writeIfMissing func(string, string)) []string {
	issues := make([]string, 0)
	for _, row := range rows {
		repoPath := filepath.Join(root, filepath.FromSlash(row.TargetRepo))
		repoName := filepath.Base(repoPath)
		_, statErr := os.Stat(repoPath)

		switch row.Action {
		case "create":
			if statErr == nil {
				continue
			}
			scaffoldRepoDocs(repoPath, repoName, mkDir, writeIfMissing)
		case "update":
			if statErr != nil {
				issues = append(issues, fmt.Sprintf("update action references missing repo: %s", filepath.ToSlash(repoPath)))
			}
		default:
			issues = append(issues, fmt.Sprintf("unsupported repo action '%s' for target %s", row.Action, filepath.ToSlash(repoPath)))
		}
	}
	return issues
}

func scaffoldRepoDocs(repoRoot, repoName string, mkDir func(string), writeIfMissing func(string, string)) {
	mkDir(repoRoot)
	mkDir(filepath.Join(repoRoot, "docs"))
	mkDir(filepath.Join(repoRoot, "docs", "plans"))
	mkDir(filepath.Join(repoRoot, "docs", "current-state"))
	mkDir(filepath.Join(repoRoot, "docs", "diagrams"))
	mkDir(filepath.Join(repoRoot, "docs", "tooling"))
	mkDir(filepath.Join(repoRoot, "docs", "handoffs"))

	writeIfMissing(filepath.Join(repoRoot, "README.md"), repoReadmeTemplate(repoName))
	writeIfMissing(filepath.Join(repoRoot, "CHANGELOG.md"), "# Changelog\n\n## [Unreleased]\n\n## [YYYY-MM-DD] - <version/tag>\n### Plan Reference\n- Plan artifact: <path or PLAN-id>\n\n### Added\n- <change summary> (REQ-xxx, PLAN-xxx)\n\n### Changed\n- <change summary> (DEF-xxx, TEST-xxx)\n\n### Fixed\n- <change summary> (DEF-xxx)\n\n### Migration/Operations Notes\n- <required actions, if any>\n")
	writeIfMissing(filepath.Join(repoRoot, "docs", "README.md"), docsReadme(repoName))
	writeIfMissing(filepath.Join(repoRoot, "docs", "plans", "README.md"), "# Plans\n\nStore current implementation plans and story-linked execution artifacts.\n")
	writeIfMissing(filepath.Join(repoRoot, "docs", "current-state", "README.md"), "# Current State\n\nSummarize current runtime behavior, open risks, and known constraints.\n")
	writeIfMissing(filepath.Join(repoRoot, "docs", "current-state", "implementation-summary.md"), "# Implementation Summary\n\n## Plan intent\n-\n\n## Key implementation details\n-\n\n## Deferred/not delivered\n-\n")
	writeIfMissing(filepath.Join(repoRoot, "docs", "diagrams", "high-level.mmd"), "flowchart TD\n    A[Client] --> B[Service Boundary]\n    B --> C[Core Logic]\n")
	writeIfMissing(filepath.Join(repoRoot, "docs", "handoffs", "traceability-pack.md"), "# Handoff Traceability Pack - Repo\n\n## ID Inventory\n- REQ:\n- PLAN:\n- DIAG:\n- TEST:\n- DEF:\n- TC:\n\n## Mapping\n- REQ-xxx -> PLAN-xxx -> DIAG-xxx -> TEST-xxx/DEF-xxx\n\n## Planning Behavior\n- profile_id:\n- profile_source:\n- profile_version:\n- resolved_controls_snapshot:\n\n## Workflow Decisions Applied\n-\n\n## ADR Ledger\n| ADR ID | Title | Applied Scope | Approval Status |\n|---|---|---|---|\n| ADR-xxxx | <title> | <repo/system> | APPROVED/BLOCKED |\n\n## System Description\n-\n\n## Diagram Index\n- [DIAG-xxx] <name> -> <path>\n")
	writeIfMissing(filepath.Join(repoRoot, "docs", "tooling", "go-bin-convention.md"), "# Go Binary Convention\n\n- Resolve `go` from `PATH` first.\n- Fallback probe order: `/usr/local/go/bin/go`, `/opt/homebrew/bin/go`, `/snap/bin/go`.\n- If unresolved, fail with explicit `go not found` and execution context evidence.\n")
}

func scaffoldNamingADR(targetRoot string, writeIfMissing func(string, string)) {
	writeIfMissing(filepath.Join(targetRoot, "docs", "adr", "ADR-0001-repo-naming-conventions.md"), namingADRTemplate())
}

func scaffoldRun5GovernanceArtifacts(targetRoot string, writeIfMissing func(string, string)) {
	writeIfMissing(filepath.Join(targetRoot, "docs", "plans", "index.md"), "# Plans Index\n\n| REQ ID | PLAN ID | Plan File | Target Repo | Owner | Status |\n|---|---|---|---|---|---|\n| REQ-001 | PLAN-001 | docs/plans/PLAN-001-<slug>.md | <repo> | <owner> | DRAFT/APPROVED |\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "plans", "planning-signoff.md"), "# Planning Signoff\n\n- Approval Owner: <name/role>\n- Approval Status: DRAFT\n- Approved Timestamp (UTC):\n- Plan Index Path: docs/plans/index.md\n- Notes:\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "data-architecture-decision.md"), "# Data Architecture Decision\n\n- Decision ID: DA-0001\n- Status: Proposed\n- Primary database engine:\n- Primary cache engine:\n- Approved deviations: none\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "handoffs", "routing-matrix.md"), "# Handoff Routing Matrix\n\n| Handoff ID | From Phase | To Phase | Artifact Bundle Path | Receiver Role | Receiver Name | Ack Status | Ack Timestamp (UTC) | Ack Evidence Path |\n|---|---|---|---|---|---|---|---|---|\n| HO-001 | 01 | 02 | docs/handoffs/po/ | architect | <name> | PENDING |  |  |\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "handoffs", "traceability-pack.md"), "# Handoff Traceability Pack - System\n\n## ID Inventory\n- REQ:\n- PLAN:\n- DIAG:\n- TEST:\n- DEF:\n- TC:\n\n## Mapping\n- REQ-xxx -> PLAN-xxx -> DIAG-xxx -> TEST-xxx/DEF-xxx\n\n## Planning Behavior\n- profile_id:\n- profile_source:\n- profile_version:\n- resolved_controls_snapshot:\n\n## Workflow Decisions Applied\n-\n\n## ADR Ledger\n| ADR ID | Title | Applied Scope | Approval Status |\n|---|---|---|---|\n| ADR-xxxx | <title> | <repo/system> | APPROVED/BLOCKED |\n\n## System Description\n-\n\n## Diagram Index\n- [DIAG-xxx] <name> -> <path>\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "integration", "compose-mode-decision.md"), "# Integration Mode Decision\n\n- Selected mode:\n- Evidence paths:\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "tooling", "skill-capability-gap.md"), "# Skill Capability Gap\n\nUse only when planning intent exceeds available skill/action capability.\n\n- Status: CLOSED\n")
}

func workspaceReadme() string {
	return "# Multi-Repo Workspace\n\nThis folder is intentionally initialized as an empty workspace boundary.\n\n## Policy\n- Repositories are created or updated only from approved planning outcomes (`docs/plans/index.md`, `docs/plans/repo-change-plan.md`, `docs/plans/planning-signoff.md`).\n- Bootstrap/scaffold steps do not create service or integration repositories by default.\n\n## Expected workflow\n1. Product Owner and Architect produce approved planning artifacts.\n2. Repository create/update actions are executed from those approved decisions.\n3. Traceability maps each `PLAN-*` to target repository paths.\n"
}

func docsReadme(repoName string) string {
	return fmt.Sprintf("# %s docs\n\n## Structure\n- `docs/plans/` planned and active delivery slices\n- `docs/current-state/` current architecture/runtime notes\n- `docs/diagrams/` high-level service and dependency diagrams\n", repoName)
}

func namingADRTemplate() string {
	return "# ADR-0001: Repository naming conventions\n\n" +
		"- Status: Proposed\n" +
		"- Date: YYYY-MM-DD\n" +
		"- Decision owners: <owner-role-or-team>\n\n" +
		"## Context\n\n" +
		"A multi-repo workspace needs deterministic naming so planning, traceability, and integration automation remain stable.\n\n" +
		"## Decision\n\n" +
		"Use these conventions:\n\n" +
		"1. Service repositories\n" +
		"   - Pattern: <product>-svc-<bounded-context>-<runtime>\n" +
		"   - Examples:\n" +
		"     - acme-svc-orders-go-app\n" +
		"     - acme-lib-catalog-go-module\n\n" +
		"2. UI repositories\n" +
		"   - Pattern: <product>-web-<bounded-context>-ts-react\n" +
		"   - Example: acme-web-portal-ts-react\n\n" +
		"3. Integration/demo compose repository\n" +
		"   - Pattern: <product>-demo-compose\n" +
		"   - Example: acme-demo-compose\n\n" +
		"4. Local workspace directory layout\n" +
		"   - Keep checked out repos under repos/.\n" +
		"   - Keep compose integration checkout under repos/demo-compose/workspace/.\n\n" +
		"## Consequences\n\n" +
		"- Build and orchestration scripts can infer repo purpose from names.\n" +
		"- Planning artifacts can map IDs to deterministic repo paths.\n" +
		"- New service onboarding uses a consistent scaffold baseline.\n"
}

func repoReadmeTemplate(repoName string) string {
	return fmt.Sprintf("# %s\n\n## Purpose\n- What this repo is responsible for.\n\n## Scope\n- In scope:\n- Out of scope:\n\n## Dependencies\n- Runtime dependencies (DB/cache/services)\n- Build/test toolchain\n\n## Runbook\n- Build:\n- Run:\n- Test:\n\n## Interfaces\n- API/contracts/events exposed and consumed.\n\n## Ownership\n- Team:\n- Primary owner:\n- Escalation:\n\n## Traceability\n- REQ IDs:\n- PLAN IDs:\n- DIAG IDs:\n- ADR IDs applied:\n\n## Plan-to-Implementation Summary\n- Plan intent:\n- Key implementation details delivered:\n- Deferred/not delivered:\n", repoName)
}

func printBlocked(message string) {
	payload, _ := json.Marshal(map[string]any{
		"status": "BLOCKED",
		"issues": []string{message},
	})
	fmt.Println(string(payload))
}
