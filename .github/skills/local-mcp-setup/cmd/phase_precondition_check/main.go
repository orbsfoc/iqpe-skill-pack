package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	phase := flag.String("phase", "01", "workflow phase to validate (01-05)")
	targetRoot := flag.String("target-root", "", "target repository root (defaults to cwd)")
	flag.Parse()

	root := strings.TrimSpace(*targetRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			printBlocked(*phase, []string{"unable to determine working directory"})
			return
		}
		root = cwd
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		printBlocked(*phase, []string{"invalid target root"})
		return
	}

	missing := []string{}
	require := func(rel string) {
		target := filepath.Join(absRoot, filepath.FromSlash(rel))
		if info, statErr := os.Stat(target); statErr != nil || info.IsDir() {
			missing = append(missing, rel)
		}
	}

	switch strings.TrimSpace(*phase) {
	case "01":
		require("docs/tooling/workflow-preflight.json")
		require("docs/tooling/spec-tech-detect.json")
		require("docs/planning-behavior-resolution.md")
	case "02":
		require("docs/requirements.md")
		require("docs/repo-topology-decision.md")
		require("docs/openapi-contract-plan.md")
		require("docs/data-architecture-decision.md")
	case "03":
		require("docs/implementation-plan.md")
		require("docs/technology-constraints.md")
		require("docs/handoffs/architect/phase-gate.md")
		if requiresOpenAPISpec(absRoot) {
			if !hasOpenAPISpec(absRoot) {
				missing = append(missing, "docs/openapi/*.yaml")
			}
		}
	case "04":
		require("docs/handoffs/dev/phase-gate.md")
		require("docs/tooling/mcp-usage-evidence.md")
		require("docs/integration/compose-mode-decision.md")
	case "05":
		require("docs/handoffs/release/phase-gate.md")
		require("docs/handoffs/routing-matrix.md")
		require("docs/data-architecture-decision.md")
	default:
		printBlocked(*phase, []string{"unsupported phase value"})
		return
	}

	if len(missing) > 0 {
		printBlocked(*phase, missing)
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"status": "PASS",
		"phase":  strings.TrimSpace(*phase),
	})
	fmt.Println(string(payload))
}

func requiresOpenAPISpec(root string) bool {
	contractPlan := filepath.Join(root, "docs", "openapi-contract-plan.md")
	data, err := os.ReadFile(contractPlan)
	if err != nil {
		return false
	}
	text := strings.ToLower(string(data))
	return strings.Contains(text, "openapi") || strings.Contains(text, "http")
}

func hasOpenAPISpec(root string) bool {
	openapiRoot := filepath.Join(root, "docs", "openapi")
	entries, err := os.ReadDir(openapiRoot)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := strings.ToLower(entry.Name())
		if strings.HasSuffix(name, ".yaml") || strings.HasSuffix(name, ".yml") {
			return true
		}
	}
	return false
}

func printBlocked(phase string, missing []string) {
	payload, _ := json.Marshal(map[string]any{
		"status":  "BLOCKED",
		"phase":   strings.TrimSpace(phase),
		"missing": missing,
	})
	fmt.Println(string(payload))
}
