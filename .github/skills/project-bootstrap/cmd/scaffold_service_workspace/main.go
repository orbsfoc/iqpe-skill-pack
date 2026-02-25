package main

import (
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
}

func main() {
	targetRoot := flag.String("target-root", "", "target repository root (defaults to cwd)")
	workspaceDir := flag.String("workspace-dir", "repos", "relative workspace directory for service repos")
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
	writeIfMissing(filepath.Join(workspacePath, "README.md"), workspaceReadme())

	scaffoldGoModule(workspacePath, mkDir, writeIfMissing)
	scaffoldGoApp(workspacePath, mkDir, writeIfMissing)
	scaffoldTsReact(workspacePath, mkDir, writeIfMissing)
	scaffoldDemoCompose(workspacePath, mkDir, writeIfMissing)
	scaffoldNamingADR(absRoot, writeIfMissing)
	scaffoldRun5GovernanceArtifacts(absRoot, writeIfMissing)

	data, _ := json.Marshal(result)
	fmt.Println(string(data))
}

func scaffoldGoModule(workspace string, mkDir func(string), writeIfMissing func(string, string)) {
	repoRoot := filepath.Join(workspace, "go-library-module")
	scaffoldRepoDocs(repoRoot, "go-library-module", mkDir, writeIfMissing)
	writeIfMissing(filepath.Join(repoRoot, "go.mod"), "module example.com/go-library-module\n\ngo 1.24\n")
	writeIfMissing(filepath.Join(repoRoot, "pkg", "module", "module.go"), "package module\n\nfunc Name() string {\n\treturn \"go-library-module\"\n}\n")
	writeIfMissing(filepath.Join(repoRoot, "Dockerfile"), "FROM golang:1.24-alpine\nWORKDIR /workspace\nCOPY . .\nRUN go test ./...\nCMD [\"sh\",\"-c\",\"echo go-library-module ready\"]\n")
}

func scaffoldGoApp(workspace string, mkDir func(string), writeIfMissing func(string, string)) {
	repoRoot := filepath.Join(workspace, "go-application-service")
	scaffoldRepoDocs(repoRoot, "go-application-service", mkDir, writeIfMissing)
	writeIfMissing(filepath.Join(repoRoot, "go.mod"), "module example.com/go-application-service\n\ngo 1.24\n")
	writeIfMissing(filepath.Join(repoRoot, "cmd", "app", "main.go"), "package main\n\nimport (\n\t\"fmt\"\n\t\"net/http\"\n)\n\nfunc main() {\n\thttp.HandleFunc(\"/healthz\", func(w http.ResponseWriter, _ *http.Request) {\n\t\tw.WriteHeader(http.StatusOK)\n\t\t_, _ = w.Write([]byte(\"ok\"))\n\t})\n\tfmt.Println(\"go-application-service listening on :8080\")\n\t_ = http.ListenAndServe(\":8080\", nil)\n}\n")
	writeIfMissing(filepath.Join(repoRoot, "Dockerfile"), "FROM golang:1.24 AS build\nWORKDIR /src\nCOPY . .\nRUN CGO_ENABLED=0 go build -o /out/app ./cmd/app\n\nFROM gcr.io/distroless/static-debian12\nCOPY --from=build /out/app /app\nEXPOSE 8080\nENTRYPOINT [\"/app\"]\n")
}

func scaffoldTsReact(workspace string, mkDir func(string), writeIfMissing func(string, string)) {
	repoRoot := filepath.Join(workspace, "ts-react-service")
	scaffoldRepoDocs(repoRoot, "ts-react-service", mkDir, writeIfMissing)
	writeIfMissing(filepath.Join(repoRoot, "package.json"), "{\n  \"name\": \"ts-react-service\",\n  \"private\": true,\n  \"version\": \"0.1.0\",\n  \"type\": \"module\",\n  \"scripts\": {\n    \"dev\": \"vite\",\n    \"build\": \"tsc && vite build\",\n    \"preview\": \"vite preview --host 0.0.0.0 --port 4173\"\n  },\n  \"dependencies\": {\n    \"react\": \"^18.3.1\",\n    \"react-dom\": \"^18.3.1\"\n  },\n  \"devDependencies\": {\n    \"@types/react\": \"^18.3.12\",\n    \"@types/react-dom\": \"^18.3.1\",\n    \"typescript\": \"^5.7.3\",\n    \"vite\": \"^6.0.7\"\n  }\n}\n")
	writeIfMissing(filepath.Join(repoRoot, "tsconfig.json"), "{\n  \"compilerOptions\": {\n    \"target\": \"ES2020\",\n    \"module\": \"ESNext\",\n    \"moduleResolution\": \"Bundler\",\n    \"jsx\": \"react-jsx\",\n    \"strict\": true\n  },\n  \"include\": [\"src\"]\n}\n")
	writeIfMissing(filepath.Join(repoRoot, "vite.config.ts"), "import { defineConfig } from 'vite'\n\nexport default defineConfig({\n  server: {\n    host: '0.0.0.0',\n    port: 4173\n  }\n})\n")
	writeIfMissing(filepath.Join(repoRoot, "index.html"), "<!doctype html>\n<html lang=\"en\">\n  <head>\n    <meta charset=\"UTF-8\" />\n    <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\" />\n    <title>ts-react-service</title>\n  </head>\n  <body>\n    <div id=\"root\"></div>\n    <script type=\"module\" src=\"/src/main.tsx\"></script>\n  </body>\n</html>\n")
	writeIfMissing(filepath.Join(repoRoot, "src", "main.tsx"), "import React from 'react'\nimport ReactDOM from 'react-dom/client'\nimport { App } from './App'\n\nReactDOM.createRoot(document.getElementById('root')!).render(\n  <React.StrictMode>\n    <App />\n  </React.StrictMode>\n)\n")
	writeIfMissing(filepath.Join(repoRoot, "src", "App.tsx"), "export function App() {\n  return (\n    <main>\n      <h1>ts-react-service</h1>\n      <p>Starter scaffold for UI/service integration.</p>\n    </main>\n  )\n}\n")
	writeIfMissing(filepath.Join(repoRoot, "Dockerfile"), "FROM node:20-alpine\nWORKDIR /workspace\nCOPY . .\nRUN npm install\nEXPOSE 4173\nCMD [\"npm\",\"run\",\"dev\",\"--\",\"--host\",\"0.0.0.0\",\"--port\",\"4173\"]\n")
}

func scaffoldDemoCompose(workspace string, mkDir func(string), writeIfMissing func(string, string)) {
	repoRoot := filepath.Join(workspace, "demo-compose")
	scaffoldRepoDocs(repoRoot, "demo-compose", mkDir, writeIfMissing)
	mkDir(filepath.Join(repoRoot, "workspace"))
	mkDir(filepath.Join(repoRoot, "scripts"))
	writeIfMissing(filepath.Join(repoRoot, "workspace", ".gitkeep"), "")
	writeIfMissing(filepath.Join(repoRoot, "scripts", "checkout-repos.sh"), "#!/usr/bin/env bash\nset -euo pipefail\n\nROOT=\"$(cd \"$(dirname \"$0\")/..\" && pwd)\"\nWORKSPACE=\"$ROOT/workspace\"\nmkdir -p \"$WORKSPACE\"\n\nclone_or_update() {\n  local name=\"$1\"\n  local repo_url=\"$2\"\n  local branch=\"${3:-main}\"\n  if [[ -z \"$repo_url\" ]]; then\n    echo \"skip $name (repo URL not set)\"\n    return\n  fi\n  local target=\"$WORKSPACE/$name\"\n  if [[ -d \"$target/.git\" ]]; then\n    git -C \"$target\" fetch --all\n    git -C \"$target\" checkout \"$branch\"\n    git -C \"$target\" pull --ff-only\n  else\n    git clone --branch \"$branch\" \"$repo_url\" \"$target\"\n  fi\n}\n\nclone_or_update \"go-library-module\" \"${GO_LIBRARY_REPO_URL:-${GO_MODULE_REPO_URL:-}}\" \"${GO_LIBRARY_REPO_REF:-${GO_MODULE_REPO_REF:-main}}\"\nclone_or_update \"go-application-service\" \"${GO_APPLICATION_REPO_URL:-}\" \"${GO_APPLICATION_REPO_REF:-main}\"\nclone_or_update \"ts-react-service\" \"${TS_REACT_REPO_URL:-}\" \"${TS_REACT_REPO_REF:-main}\"\n\necho \"workspace checkout complete: $WORKSPACE\"\n")
	writeIfMissing(filepath.Join(repoRoot, "docker-compose.yml"), "services:\n  go-library-module:\n    build:\n      context: ./workspace/go-library-module\n      dockerfile: Dockerfile\n    command: [\"sh\",\"-c\",\"echo go-library-module integrated\"]\n\n  go-application-service:\n    build:\n      context: ./workspace/go-application-service\n      dockerfile: Dockerfile\n    ports:\n      - \"8080:8080\"\n\n  ts-react-service:\n    build:\n      context: ./workspace/ts-react-service\n      dockerfile: Dockerfile\n    ports:\n      - \"4173:4173\"\n")
	writeIfMissing(filepath.Join(repoRoot, "README.md"), "# demo-compose\n\n## Purpose\n- Integrate and validate cross-repo behavior across service and module repositories.\n\n## Scope\n- In scope: local integration orchestration and smoke validation.\n- Out of scope: production deployment orchestration.\n\n## Dependencies\n- Docker and Docker Compose\n- Checked-out repository workspace under `workspace/`\n\n## Runbook\n- Build: `docker compose build`\n- Run: `docker compose up`\n- Test: run repo-specific smoke checks after compose startup\n\n## Interfaces\n- Consumes service interfaces exposed by checked-out repos.\n\n## Ownership\n- Team: integration-platform\n- Primary owner: <owner>\n- Escalation: <contact>\n\n## Traceability\n- REQ IDs: REQ-\n- PLAN IDs: PLAN-\n- DIAG IDs: DIAG-\n- ADR IDs applied: ADR-\n\n## Plan-to-Implementation Summary\n- Plan intent:\n- Key implementation details delivered:\n- Deferred/not delivered:\n")
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
	writeIfMissing(filepath.Join(targetRoot, "docs", "data-architecture-decision.md"), "# Data Architecture Decision\n\n- Decision ID: DA-0001\n- Status: Proposed\n- Primary database engine:\n- Primary cache engine:\n- Approved deviations: none\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "handoffs", "routing-matrix.md"), "# Handoff Routing Matrix\n\n| Handoff ID | From Phase | To Phase | Artifact Bundle Path | Receiver Role | Receiver Name | Ack Status | Ack Timestamp (UTC) | Ack Evidence Path |\n|---|---|---|---|---|---|---|---|---|\n| HO-001 | 01 | 02 | docs/handoffs/po/ | architect | <name> | PENDING |  |  |\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "handoffs", "traceability-pack.md"), "# Handoff Traceability Pack - System\n\n## ID Inventory\n- REQ:\n- PLAN:\n- DIAG:\n- TEST:\n- DEF:\n- TC:\n\n## Mapping\n- REQ-xxx -> PLAN-xxx -> DIAG-xxx -> TEST-xxx/DEF-xxx\n\n## Planning Behavior\n- profile_id:\n- profile_source:\n- profile_version:\n- resolved_controls_snapshot:\n\n## Workflow Decisions Applied\n-\n\n## ADR Ledger\n| ADR ID | Title | Applied Scope | Approval Status |\n|---|---|---|---|\n| ADR-xxxx | <title> | <repo/system> | APPROVED/BLOCKED |\n\n## System Description\n-\n\n## Diagram Index\n- [DIAG-xxx] <name> -> <path>\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "integration", "compose-mode-decision.md"), "# Compose Mode Decision\n\n- Compose mode: local-dev\n- Evidence paths:\n")
	writeIfMissing(filepath.Join(targetRoot, "docs", "tooling", "skill-capability-gap.md"), "# Skill Capability Gap\n\nUse only when planning intent exceeds available skill/action capability.\n\n- Status: CLOSED\n")
}

func workspaceReadme() string {
	return "# Multi-Repo Workspace\n\nThis folder contains scaffolded repositories and a demo integration compose workspace.\n\n## Starter repositories\n- `go-library-module`\n- `go-application-service`\n- `ts-react-service`\n- `demo-compose`\n\nEach repository includes baseline docs structure (`docs/plans`, `docs/current-state`, `docs/diagrams`, `docs/handoffs`), plus operational README and changelog templates.\n"
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
