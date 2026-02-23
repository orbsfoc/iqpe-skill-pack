package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type techDecisionCandidate struct {
	Value     string `json:"value"`
	File      string `json:"file"`
	Line      int    `json:"line"`
	MatchedOn string `json:"matched_on"`
}

type approvedTechBaseline struct {
	AuthoritySource string `json:"authority_source"`
	ApprovalOwner   string `json:"approval_owner"`
	ApprovalStatus  string `json:"approval_status"`
	Decisions       struct {
		BackendRuntime    string `json:"backend_runtime"`
		FrontendFramework string `json:"frontend_framework"`
		PersistentEngine  string `json:"persistent_engine"`
		MigrationTool     string `json:"migration_tool"`
		RedisVersion      string `json:"redis_version"`
	} `json:"decisions"`
}

type mcpConfig struct {
	Servers map[string]mcpServer `json:"servers"`
}

type mcpServer struct {
	Transport string   `json:"transport"`
	Command   string   `json:"command"`
	Args      []string `json:"args"`
}

func nowUTC() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func resolveBinary() string {
	if path, err := exec.LookPath("iqpe-localmcp"); err == nil {
		return path
	}
	home, err := os.UserHomeDir()
	if err == nil {
		candidates := []string{
			filepath.Join(home, "bin", "iqpe-localmcp"),
			filepath.Join(home, ".local", "bin", "iqpe-localmcp"),
		}
		for _, candidate := range candidates {
			if isExecutable(candidate) {
				return candidate
			}
		}
	}
	return "iqpe-localmcp"
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	return info.Mode().Perm()&0o111 != 0
}

func ensureMCPConfig(targetRoot string) (string, error) {
	vscodeDir := filepath.Join(targetRoot, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0o755); err != nil {
		return "", err
	}
	mcpPath := filepath.Join(vscodeDir, "mcp.json")
	if _, err := os.Stat(mcpPath); err == nil {
		return mcpPath, nil
	}

	command := resolveBinary()
	cfg := mcpConfig{Servers: map[string]mcpServer{
		"repo-read-local": {
			Transport: "stdio",
			Command:   command,
			Args:      []string{"--server", "repo-read"},
		},
		"docflow-actions-local": {
			Transport: "stdio",
			Command:   command,
			Args:      []string{"--server", "docflow-actions"},
		},
		"docs-graph-local": {
			Transport: "stdio",
			Command:   command,
			Args:      []string{"--server", "docs-graph"},
		},
		"policy-local": {
			Transport: "stdio",
			Command:   command,
			Args:      []string{"--server", "policy"},
		},
	}}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(mcpPath, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return mcpPath, nil
}

func writeBootstrapReport(targetRoot, specDir, mcpPath string) (string, error) {
	out := filepath.Join(targetRoot, "docs", "tooling", "bootstrap-report.md")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	if strings.TrimSpace(specDir) == "" {
		specDir = "<unset>"
	}
	content := strings.Join([]string{
		"# Workflow Bootstrap Report",
		"",
		fmt.Sprintf("- Timestamp (UTC): %s", nowUTC()),
		fmt.Sprintf("- Target root: %s", targetRoot),
		fmt.Sprintf("- SPEC_DIR: %s", specDir),
		"",
		"## Applied changes",
		"- Ensured `.vscode/mcp.json` exists",
		fmt.Sprintf("- MCP config path: `%s`", mcpPath),
		"- Generated this report",
		"",
		"## Next steps",
		"1) Run workflow preflight and require PASS",
		"2) Start with orchestrator prompt",
	}, "\n") + "\n"
	if err := os.WriteFile(out, []byte(content), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

func countSpecFiles(specDir string) int {
	allowed := map[string]bool{".md": true, ".yaml": true, ".yml": true, ".json": true, ".txt": true}
	count := 0
	_ = filepath.WalkDir(specDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if allowed[strings.ToLower(filepath.Ext(path))] {
			count++
		}
		return nil
	})
	return count
}

func commandRunnable(command string) bool {
	command = strings.TrimSpace(command)
	if command == "" {
		return false
	}
	if filepath.IsAbs(command) || strings.Contains(command, string(os.PathSeparator)) {
		return isExecutable(command)
	}
	_, err := exec.LookPath(command)
	return err == nil
}

func runPreflight(targetRoot, specDirArg, mcpPath string) (string, error) {
	specDir := specDirArg
	if !filepath.IsAbs(specDir) {
		specDir = filepath.Join(targetRoot, specDir)
	}
	specDir, _ = filepath.Abs(specDir)

	mcpServersPresent := false
	mcpUsesLocalBinary := false
	mcpConfigCommandRunnable := false
	mcpConfigCommand := ""
	mcpParseError := ""

	if data, err := os.ReadFile(mcpPath); err == nil {
		var cfg mcpConfig
		if err := json.Unmarshal(data, &cfg); err != nil {
			mcpParseError = err.Error()
		} else {
			required := []string{"repo-read-local", "docflow-actions-local"}
			present := 0
			local := 0
			runnable := 0
			for _, name := range required {
				server, ok := cfg.Servers[name]
				if !ok {
					continue
				}
				present++
				if strings.Contains(strings.ToLower(server.Command), "iqpe-localmcp") {
					local++
				}
				if commandRunnable(server.Command) {
					runnable++
				}
				if name == "repo-read-local" {
					mcpConfigCommand = strings.TrimSpace(server.Command)
				}
			}
			mcpServersPresent = present == len(required)
			mcpUsesLocalBinary = local == len(required)
			mcpConfigCommandRunnable = runnable == len(required)
		}
	}

	specCount := 0
	specReady := false
	if info, err := os.Stat(specDir); err == nil && info.IsDir() {
		specCount = countSpecFiles(specDir)
		specReady = specCount > 0
	}

	mcpReady := mcpServersPresent && mcpUsesLocalBinary && mcpConfigCommandRunnable
	status := "PASS"
	if !mcpReady || !specReady {
		status = "BLOCKED"
	}

	out := filepath.Join(targetRoot, "docs", "tooling", "workflow-preflight.json")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	payload := map[string]any{
		"status":                      status,
		"timestamp_utc":               nowUTC(),
		"target_root":                 targetRoot,
		"spec_dir":                    specDir,
		"mcp_config_path":             mcpPath,
		"mcp_ready":                   mcpReady,
		"mcp_servers_present":         mcpServersPresent,
		"mcp_uses_local_binary":       mcpUsesLocalBinary,
		"mcp_config_parse_error":      mcpParseError,
		"mcp_config_command":          mcpConfigCommand,
		"mcp_config_command_runnable": mcpConfigCommandRunnable,
		"spec_ready":                  specReady,
		"spec_file_count":             specCount,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

func loadApprovedTechBaseline(path string) (*approvedTechBaseline, string) {
	if strings.TrimSpace(path) == "" {
		return nil, ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err.Error()
	}
	var baseline approvedTechBaseline
	if err := json.Unmarshal(data, &baseline); err != nil {
		return nil, err.Error()
	}
	if strings.ToUpper(strings.TrimSpace(baseline.ApprovalStatus)) != "APPROVED" {
		return nil, "baseline approval status is not APPROVED"
	}
	return &baseline, ""
}

func runSpecTechDetect(targetRoot, specDirArg, corporateTechFile string) (string, error) {
	specDir := specDirArg
	if !filepath.IsAbs(specDir) {
		specDir = filepath.Join(targetRoot, specDir)
	}
	specDir, _ = filepath.Abs(specDir)

	type keyword struct {
		value  string
		regexp *regexp.Regexp
	}
	compile := func(pattern string) *regexp.Regexp {
		return regexp.MustCompile("(?i)" + pattern)
	}

	backendMatchers := []keyword{{"golang", compile(`\bgo(lang)?\b`)}, {"node", compile(`\bnode(js)?\b`)}, {"java", compile(`\bjava\b`)}, {"dotnet", compile(`\.net|dotnet`)}}
	frontendMatchers := []keyword{{"react", compile(`\breact\b`)}, {"vue", compile(`\bvue\b`)}, {"angular", compile(`\bangular\b`)}}
	dbMatchers := []keyword{{"postgres", compile(`\bpostgres(ql)?\b`)}, {"sqlite", compile(`\bsqlite\b`)}, {"mysql", compile(`\bmysql\b`)}, {"mssql", compile(`\bms\s*sql|sql\s*server\b`)}}
	migrationMatchers := []keyword{{"flyway", compile(`\bflyway\b`)}, {"liquibase", compile(`\bliquibase\b`)}, {"golang-migrate", compile(`\bmigrate\b`)}}

	findFirst := func(filePath, line string, lineNo int, patterns []keyword) *techDecisionCandidate {
		for _, item := range patterns {
			if item.regexp.MatchString(line) {
				return &techDecisionCandidate{Value: item.value, File: filepath.ToSlash(filePath), Line: lineNo, MatchedOn: strings.TrimSpace(line)}
			}
		}
		return nil
	}

	var backend, frontend, database, migration *techDecisionCandidate
	allowedExt := map[string]bool{".md": true, ".yaml": true, ".yml": true, ".json": true, ".txt": true}

	_ = filepath.WalkDir(specDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !allowedExt[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return nil
		}
		defer file.Close()

		rel, _ := filepath.Rel(specDir, path)
		scanner := bufio.NewScanner(file)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			text := scanner.Text()
			if backend == nil {
				backend = findFirst(rel, text, lineNo, backendMatchers)
			}
			if frontend == nil {
				frontend = findFirst(rel, text, lineNo, frontendMatchers)
			}
			if database == nil {
				database = findFirst(rel, text, lineNo, dbMatchers)
			}
			if migration == nil {
				migration = findFirst(rel, text, lineNo, migrationMatchers)
			}
			if backend != nil && frontend != nil && database != nil && migration != nil {
				return io.EOF
			}
		}
		return nil
	})

	baseline, baselineError := loadApprovedTechBaseline(corporateTechFile)
	if baseline != nil {
		if backend == nil && strings.TrimSpace(baseline.Decisions.BackendRuntime) != "" {
			backend = &techDecisionCandidate{Value: strings.TrimSpace(baseline.Decisions.BackendRuntime), File: filepath.ToSlash(corporateTechFile), Line: 1, MatchedOn: "corporate approved baseline"}
		}
		if frontend == nil && strings.TrimSpace(baseline.Decisions.FrontendFramework) != "" {
			frontend = &techDecisionCandidate{Value: strings.TrimSpace(baseline.Decisions.FrontendFramework), File: filepath.ToSlash(corporateTechFile), Line: 1, MatchedOn: "corporate approved baseline"}
		}
		if database == nil && strings.TrimSpace(baseline.Decisions.PersistentEngine) != "" {
			database = &techDecisionCandidate{Value: strings.TrimSpace(baseline.Decisions.PersistentEngine), File: filepath.ToSlash(corporateTechFile), Line: 1, MatchedOn: "corporate approved baseline"}
		}
		if migration == nil && strings.TrimSpace(baseline.Decisions.MigrationTool) != "" {
			migration = &techDecisionCandidate{Value: strings.TrimSpace(baseline.Decisions.MigrationTool), File: filepath.ToSlash(corporateTechFile), Line: 1, MatchedOn: "corporate approved baseline"}
		}
	}

	detected := map[string]any{}
	if backend != nil {
		detected["backend_runtime"] = backend
	}
	if frontend != nil {
		detected["frontend_framework"] = frontend
	}
	if database != nil {
		detected["persistent_engine"] = database
	}
	if migration != nil {
		detected["migration_tool"] = migration
	}

	status := "PASS"
	if backend == nil || frontend == nil || database == nil {
		status = "BLOCKED"
	}

	out := filepath.Join(targetRoot, "docs", "tooling", "spec-tech-detect.json")
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		return "", err
	}
	payload := map[string]any{
		"spec_dir":                  specDir,
		"detected":                  detected,
		"status":                    status,
		"timestamp_utc":             nowUTC(),
		"corporate_tech_file":       corporateTechFile,
		"corporate_baseline_loaded": baseline != nil,
		"corporate_baseline_error":  baselineError,
	}
	if baseline != nil {
		payload["authority"] = map[string]any{
			"authoritative_source": baseline.AuthoritySource,
			"approval_owner":       baseline.ApprovalOwner,
			"approval_status":      baseline.ApprovalStatus,
			"redis_version":        baseline.Decisions.RedisVersion,
		}
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(out, append(data, '\n'), 0o644); err != nil {
		return "", err
	}
	return out, nil
}

func main() {
	targetRoot := flag.String("target-root", "", "absolute path to target project repo root")
	specDir := flag.String("spec-dir", "", "SPEC_DIR path (absolute or relative to target-root)")
	corporateTechFile := flag.String("corporate-tech-file", "", "optional path to corporate approved tech baseline JSON")
	flag.Parse()

	if strings.TrimSpace(*targetRoot) == "" || strings.TrimSpace(*specDir) == "" {
		fmt.Fprintln(os.Stderr, "usage: go run ./.github/skills/local-mcp-setup/bootstrap_preflight.go --target-root <target_root_abs_path> --spec-dir <spec_dir_path> [--corporate-tech-file <path>]")
		os.Exit(2)
	}

	resolvedTarget, err := filepath.Abs(*targetRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	if info, err := os.Stat(resolvedTarget); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "target root not found: %s\n", resolvedTarget)
		os.Exit(2)
	}

	mcpPath, err := ensureMCPConfig(resolvedTarget)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	bootstrapPath, err := writeBootstrapReport(resolvedTarget, *specDir, mcpPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	preflightPath, err := runPreflight(resolvedTarget, *specDir, mcpPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}
	techFile := strings.TrimSpace(*corporateTechFile)
	if techFile == "" {
		techFile = filepath.Join(resolvedTarget, ".github", "skills", "local-mcp-setup", "corporate-approved-tech.json")
	}
	if !filepath.IsAbs(techFile) {
		techFile = filepath.Join(resolvedTarget, techFile)
	}
	techFile, _ = filepath.Abs(techFile)

	specTechPath, err := runSpecTechDetect(resolvedTarget, *specDir, techFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(2)
	}

	result := map[string]any{
		"status":             "PASS",
		"bootstrap_report":   bootstrapPath,
		"workflow_preflight": preflightPath,
		"spec_tech_detect":   specTechPath,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	fmt.Println(string(data))
}
