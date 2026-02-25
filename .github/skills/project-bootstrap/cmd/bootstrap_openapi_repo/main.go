package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type result struct {
	Status       string   `json:"status"`
	TargetRepo   string   `json:"target_repo"`
	Created      bool     `json:"created"`
	CreatedDirs  []string `json:"created_dirs,omitempty"`
	CreatedFiles []string `json:"created_files,omitempty"`
	Issues       []string `json:"issues,omitempty"`
}

type planRow struct {
	Action string
	Target string
}

func main() {
	targetRoot := flag.String("target-root", "", "target repository root (defaults to cwd)")
	repoPath := flag.String("repo-path", "repos/openapi-contracts", "relative repo path to create if missing")
	repoPlanFile := flag.String("repo-plan-file", "docs/plans/repo-change-plan.md", "repo change plan markdown path relative to target root")
	requireApprovedSignoff := flag.Bool("require-approved-signoff", true, "require planning signoff approval before creating repo")
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

	targetRel := strings.TrimSpace(*repoPath)
	if targetRel == "" {
		printBlocked("repo-path cannot be empty")
		return
	}
	targetAbs := filepath.Join(absRoot, filepath.FromSlash(targetRel))

	res := result{Status: "PASS", TargetRepo: filepath.ToSlash(targetAbs), Created: false}

	if info, statErr := os.Stat(targetAbs); statErr == nil && info.IsDir() {
		emit(res)
		return
	}

	if *requireApprovedSignoff && !hasApprovedPlanningSignoff(absRoot) {
		res.Status = "BLOCKED"
		res.Issues = append(res.Issues, "planning signoff is not APPROVED: docs/plans/planning-signoff.md")
		emit(res)
		return
	}

	rows, rowErr := loadPlanRows(absRoot, strings.TrimSpace(*repoPlanFile))
	if rowErr != nil {
		res.Status = "BLOCKED"
		res.Issues = append(res.Issues, rowErr.Error())
		emit(res)
		return
	}

	if !hasCreateActionForTarget(rows, targetRel) {
		res.Status = "BLOCKED"
		res.Issues = append(res.Issues, fmt.Sprintf("repo plan does not contain create action for target: %s", filepath.ToSlash(targetRel)))
		emit(res)
		return
	}

	mkDir := func(path string) {
		if err := os.MkdirAll(path, 0o755); err == nil {
			res.CreatedDirs = append(res.CreatedDirs, filepath.ToSlash(path))
		}
	}
	writeFile := func(path, content string) {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err == nil {
			res.CreatedFiles = append(res.CreatedFiles, filepath.ToSlash(path))
		}
	}

	mkDir(targetAbs)
	mkDir(filepath.Join(targetAbs, "openapi"))
	mkDir(filepath.Join(targetAbs, "docs", "plans"))
	mkDir(filepath.Join(targetAbs, "docs", "current-state"))
	mkDir(filepath.Join(targetAbs, "docs", "diagrams"))
	mkDir(filepath.Join(targetAbs, "docs", "handoffs"))
	mkDir(filepath.Join(targetAbs, "docs", "tooling"))

	writeFile(filepath.Join(targetAbs, "README.md"), "# openapi-contracts\n\n## Purpose\n- Canonical interface contract repository.\n\n## Scope\n- In scope: API contract definitions and compatibility governance.\n- Out of scope: service runtime implementation.\n\n## Traceability\n- REQ IDs:\n- PLAN IDs:\n- DIAG IDs:\n")
	writeFile(filepath.Join(targetAbs, "CHANGELOG.md"), "# Changelog\n\n## [Unreleased]\n")
	writeFile(filepath.Join(targetAbs, "openapi", "openapi.yaml"), "openapi: 3.0.3\ninfo:\n  title: Contract Placeholder\n  version: 0.1.0\npaths: {}\n")
	writeFile(filepath.Join(targetAbs, "docs", "current-state", "implementation-summary.md"), "# Implementation Summary\n\n## Plan intent\n-\n\n## Key implementation details\n-\n")
	writeFile(filepath.Join(targetAbs, "docs", "handoffs", "traceability-pack.md"), "# Handoff Traceability Pack - OpenAPI Repo\n\n## ID Inventory\n- REQ:\n- PLAN:\n- DIAG:\n- TEST:\n- DEF:\n- TC:\n")

	res.Created = true
	emit(res)
}

func hasApprovedPlanningSignoff(root string) bool {
	path := filepath.Join(root, "docs", "plans", "planning-signoff.md")
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToUpper(string(data)), "APPROVAL STATUS: APPROVED")
}

func loadPlanRows(root, planFile string) ([]planRow, error) {
	if strings.TrimSpace(planFile) == "" {
		return nil, errors.New("repo-plan-file cannot be empty")
	}
	path := filepath.Join(root, filepath.FromSlash(planFile))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("unable to read repo plan file: %s", filepath.ToSlash(path))
	}

	rows := make([]planRow, 0)
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "|") || strings.Count(line, "|") < 5 {
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
		target := strings.TrimSpace(parts[4])
		if action == "" || target == "" {
			continue
		}
		rows = append(rows, planRow{Action: action, Target: target})
	}

	if len(rows) == 0 {
		return nil, errors.New("no repo action rows found in repo plan file")
	}
	return rows, nil
}

func hasCreateActionForTarget(rows []planRow, target string) bool {
	normalized := strings.Trim(filepath.ToSlash(target), "/")
	for _, row := range rows {
		if row.Action != "create" {
			continue
		}
		candidate := strings.Trim(filepath.ToSlash(row.Target), "/")
		if candidate == normalized {
			return true
		}
	}
	return false
}

func printBlocked(message string) {
	res := result{Status: "BLOCKED", Issues: []string{message}}
	emit(res)
}

func emit(res result) {
	data, _ := json.Marshal(res)
	fmt.Println(string(data))
}
