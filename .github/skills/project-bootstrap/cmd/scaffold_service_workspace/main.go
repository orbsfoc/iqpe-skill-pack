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

	data, _ := json.Marshal(result)
	fmt.Println(string(data))
}

func scaffoldGoModule(workspace string, mkDir func(string), writeIfMissing func(string, string)) {
	repoRoot := filepath.Join(workspace, "go-module-service")
	scaffoldRepoDocs(repoRoot, "go-module-service", mkDir, writeIfMissing)
	writeIfMissing(filepath.Join(repoRoot, "go.mod"), "module example.com/go-module-service\n\ngo 1.24\n")
	writeIfMissing(filepath.Join(repoRoot, "pkg", "module", "module.go"), "package module\n\nfunc Name() string {\n\treturn \"go-module-service\"\n}\n")
	writeIfMissing(filepath.Join(repoRoot, "Dockerfile"), "FROM golang:1.24-alpine\nWORKDIR /workspace\nCOPY . .\nRUN go test ./...\nCMD [\"sh\",\"-c\",\"echo go-module-service ready\"]\n")
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
	writeIfMissing(filepath.Join(repoRoot, "scripts", "checkout-repos.sh"), "#!/usr/bin/env bash\nset -euo pipefail\n\nROOT=\"$(cd \"$(dirname \"$0\")/..\" && pwd)\"\nWORKSPACE=\"$ROOT/workspace\"\nmkdir -p \"$WORKSPACE\"\n\nclone_or_update() {\n  local name=\"$1\"\n  local repo_url=\"$2\"\n  local branch=\"${3:-main}\"\n  if [[ -z \"$repo_url\" ]]; then\n    echo \"skip $name (repo URL not set)\"\n    return\n  fi\n  local target=\"$WORKSPACE/$name\"\n  if [[ -d \"$target/.git\" ]]; then\n    git -C \"$target\" fetch --all\n    git -C \"$target\" checkout \"$branch\"\n    git -C \"$target\" pull --ff-only\n  else\n    git clone --branch \"$branch\" \"$repo_url\" \"$target\"\n  fi\n}\n\nclone_or_update \"go-module-service\" \"${GO_MODULE_REPO_URL:-}\" \"${GO_MODULE_REPO_REF:-main}\"\nclone_or_update \"go-application-service\" \"${GO_APPLICATION_REPO_URL:-}\" \"${GO_APPLICATION_REPO_REF:-main}\"\nclone_or_update \"ts-react-service\" \"${TS_REACT_REPO_URL:-}\" \"${TS_REACT_REPO_REF:-main}\"\n\necho \"workspace checkout complete: $WORKSPACE\"\n")
	writeIfMissing(filepath.Join(repoRoot, "docker-compose.yml"), "services:\n  go-module-service:\n    build:\n      context: ./workspace/go-module-service\n      dockerfile: Dockerfile\n    command: [\"sh\",\"-c\",\"echo go-module-service integrated\"]\n\n  go-application-service:\n    build:\n      context: ./workspace/go-application-service\n      dockerfile: Dockerfile\n    ports:\n      - \"8080:8080\"\n\n  ts-react-service:\n    build:\n      context: ./workspace/ts-react-service\n      dockerfile: Dockerfile\n    ports:\n      - \"4173:4173\"\n")
	writeIfMissing(filepath.Join(repoRoot, "README.md"), "# demo-compose\n\nIntegration workspace for bringing multiple service repos together.\n\n## Usage\n\n1. Set repo URLs as environment variables:\n   - `GO_MODULE_REPO_URL`\n   - `GO_APPLICATION_REPO_URL`\n   - `TS_REACT_REPO_URL`\n2. Run `./scripts/checkout-repos.sh` to clone/update repos into `workspace/`.\n3. Start integration compose stack:\n   - `docker compose up --build`\n")
}

func scaffoldRepoDocs(repoRoot, repoName string, mkDir func(string), writeIfMissing func(string, string)) {
	mkDir(repoRoot)
	mkDir(filepath.Join(repoRoot, "docs"))
	mkDir(filepath.Join(repoRoot, "docs", "plans"))
	mkDir(filepath.Join(repoRoot, "docs", "current-state"))
	mkDir(filepath.Join(repoRoot, "docs", "diagrams"))

	writeIfMissing(filepath.Join(repoRoot, "README.md"), fmt.Sprintf("# %s\n\nStarter scaffold repository generated by project bootstrap.\n", repoName))
	writeIfMissing(filepath.Join(repoRoot, "CHANGELOG.md"), "# Changelog\n\n## Unreleased\n- Initial scaffold\n")
	writeIfMissing(filepath.Join(repoRoot, "docs", "README.md"), docsReadme(repoName))
	writeIfMissing(filepath.Join(repoRoot, "docs", "plans", "README.md"), "# Plans\n\nStore current implementation plans and story-linked execution artifacts.\n")
	writeIfMissing(filepath.Join(repoRoot, "docs", "current-state", "README.md"), "# Current State\n\nSummarize current runtime behavior, open risks, and known constraints.\n")
	writeIfMissing(filepath.Join(repoRoot, "docs", "diagrams", "high-level.mmd"), "flowchart TD\n    A[Client] --> B[Service Boundary]\n    B --> C[Core Logic]\n")
}

func scaffoldNamingADR(targetRoot string, writeIfMissing func(string, string)) {
	writeIfMissing(filepath.Join(targetRoot, "docs", "adr", "ADR-0001-repo-naming-conventions.md"), namingADRTemplate())
}

func workspaceReadme() string {
	return "# Multi-Repo Workspace\n\nThis folder contains scaffolded service repositories and a demo integration compose workspace.\n\n## Starter repositories\n- `go-module-service`\n- `go-application-service`\n- `ts-react-service`\n- `demo-compose`\n\nEach repository includes baseline docs structure (`docs/plans`, `docs/current-state`, `docs/diagrams`), plus `README.md` and `CHANGELOG.md`.\n"
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
		"     - acme-svc-catalog-go-module\n\n" +
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

func printBlocked(message string) {
	payload, _ := json.Marshal(map[string]any{
		"status": "BLOCKED",
		"issues": []string{message},
	})
	fmt.Println(string(payload))
}
