package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var blockedPathSubstrings = []string{
	"phase-gate",
	"implementation-plan",
	"technology-constraints",
	"traceability-matrix",
	"repo-topology-decision",
	"openapi-contract-plan",
	"product-intent",
	"requirements",
	"backlog",
	"severity-classification",
	"adr",
}

var blockedTitleSubstrings = []string{
	"# implementation plan",
	"# technology constraints",
	"# traceability matrix",
	"# repo topology decision",
	"# openapi contract plan",
	"# product intent",
	"# requirements",
	"# backlog",
	"# phase gate",
	"# severity classification",
	"# architecture decision record",
}

func main() {
	targetRoot := flag.String("target-root", "", "target repository root (defaults to cwd)")
	flag.Parse()

	root := strings.TrimSpace(*targetRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			printBlocked([]string{"unable to determine working directory"}, nil)
			return
		}
		root = cwd
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		printBlocked([]string{"invalid target root"}, nil)
		return
	}

	feedbackRoot := filepath.Join(absRoot, "docs", "feedback")
	if info, statErr := os.Stat(feedbackRoot); statErr != nil || !info.IsDir() {
		printPass(feedbackRoot)
		return
	}

	violations := []string{}
	walkErr := filepath.WalkDir(feedbackRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !isMarkdown(path) {
			return nil
		}
		if looksLikeFeedbackFile(path) {
			return nil
		}

		rel, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			rel = path
		}
		relSlash := strings.ToLower(filepath.ToSlash(rel))
		for _, token := range blockedPathSubstrings {
			if strings.Contains(relSlash, token) {
				violations = append(violations, filepath.ToSlash(rel))
				return nil
			}
		}

		head := readHead(path, 40)
		headLower := strings.ToLower(head)
		for _, token := range blockedTitleSubstrings {
			if strings.Contains(headLower, token) {
				violations = append(violations, filepath.ToSlash(rel))
				return nil
			}
		}
		return nil
	})
	if walkErr != nil {
		printBlocked([]string{fmt.Sprintf("failed to scan docs/feedback: %v", walkErr)}, nil)
		return
	}

	if len(violations) > 0 {
		sort.Strings(violations)
		printBlocked([]string{"docs/feedback contains non-feedback draft deliverables"}, violations)
		return
	}

	printPass(feedbackRoot)
}

func isMarkdown(path string) bool {
	name := strings.ToLower(path)
	return strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".markdown")
}

func looksLikeFeedbackFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	if base == "readme.md" {
		return true
	}
	feedbackTokens := []string{"feedback", "issue", "finding", "report"}
	for _, token := range feedbackTokens {
		if strings.Contains(base, token) {
			return true
		}
	}
	return false
}

func readHead(path string, maxLines int) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}
	return strings.Join(lines, "\n")
}

func printPass(root string) {
	payload, _ := json.Marshal(map[string]any{
		"status":        "PASS",
		"feedback_root": filepath.ToSlash(root),
	})
	fmt.Println(string(payload))
}

func printBlocked(issues []string, violations []string) {
	payload := map[string]any{
		"status": "BLOCKED",
		"issues": issues,
	}
	if len(violations) > 0 {
		payload["violations"] = violations
	}
	data, _ := json.Marshal(payload)
	fmt.Println(string(data))
}
