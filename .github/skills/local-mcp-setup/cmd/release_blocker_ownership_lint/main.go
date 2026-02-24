package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

func main() {
	targetRoot := flag.String("target-root", "", "target repository root (defaults to cwd)")
	file := flag.String("file", "", "optional severity classification markdown file path")
	flag.Parse()

	root := strings.TrimSpace(*targetRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			printBlocked("", []string{"unable to determine working directory"}, nil)
			return
		}
		root = cwd
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		printBlocked("", []string{"invalid target root"}, nil)
		return
	}

	path, err := resolveSeverityFile(absRoot, strings.TrimSpace(*file))
	if err != nil {
		printBlocked("", []string{err.Error()}, nil)
		return
	}

	findings, ownership, err := parseSeveritySections(path)
	if err != nil {
		printBlocked(path, []string{fmt.Sprintf("failed to parse severity file: %v", err)}, nil)
		return
	}

	blockerRefs := requiredBlockers(findings)
	issues := []string{}
	missing := []string{}
	for _, blockerID := range blockerRefs {
		row, ok := ownership[blockerID]
		if !ok {
			missing = append(missing, blockerID)
			continue
		}
		if !rowComplete(row) {
			issues = append(issues, fmt.Sprintf("blocker ownership incomplete for %s", blockerID))
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		issues = append(issues, "missing blocker ownership rows")
	}

	if len(issues) > 0 {
		printBlocked(path, issues, map[string]any{
			"required_blockers": blockerRefs,
			"missing_blockers":  missing,
		})
		return
	}

	payload, _ := json.Marshal(map[string]any{
		"status":            "PASS",
		"severity_file":     filepath.ToSlash(path),
		"required_blockers": blockerRefs,
	})
	fmt.Println(string(payload))
}

type finding struct {
	severity  string
	blockerID string
}

func resolveSeverityFile(root, explicit string) (string, error) {
	if explicit != "" {
		if filepath.IsAbs(explicit) {
			if exists(explicit) {
				return explicit, nil
			}
			return "", fmt.Errorf("severity file not found: %s", explicit)
		}
		joined := filepath.Join(root, filepath.FromSlash(explicit))
		if exists(joined) {
			return joined, nil
		}
		return "", fmt.Errorf("severity file not found: %s", joined)
	}

	candidates := []string{
		filepath.Join(root, "docs", "handoffs", "release", "severity-classification.md"),
		filepath.Join(root, "docs", "release", "severity-classification.md"),
		filepath.Join(root, "docs", "severity-classification.md"),
	}
	for _, candidate := range candidates {
		if exists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("severity classification file not found in default locations")
}

func exists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func parseSeveritySections(path string) ([]finding, map[string][]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	section := ""
	findings := []finding{}
	ownership := map[string][]string{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		lower := strings.ToLower(line)
		switch lower {
		case "## findings":
			section = "findings"
			continue
		case "## blocker ownership":
			section = "ownership"
			continue
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if strings.Contains(line, "---") {
			continue
		}
		cells := parseTableRow(line)
		if section == "findings" {
			if len(cells) < 5 || strings.EqualFold(cells[0], "Finding ID") {
				continue
			}
			findings = append(findings, finding{severity: strings.TrimSpace(cells[2]), blockerID: strings.TrimSpace(cells[4])})
		}
		if section == "ownership" {
			if len(cells) < 5 || strings.EqualFold(cells[0], "blocker_id") {
				continue
			}
			blockerID := strings.TrimSpace(cells[0])
			ownership[blockerID] = []string{
				strings.TrimSpace(cells[1]),
				strings.TrimSpace(cells[2]),
				strings.TrimSpace(cells[3]),
				strings.TrimSpace(cells[4]),
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}
	return findings, ownership, nil
}

func parseTableRow(line string) []string {
	trimmed := strings.TrimSpace(line)
	trimmed = strings.TrimPrefix(trimmed, "|")
	trimmed = strings.TrimSuffix(trimmed, "|")
	parts := strings.Split(trimmed, "|")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		out = append(out, strings.TrimSpace(part))
	}
	return out
}

func requiredBlockers(findings []finding) []string {
	set := map[string]bool{}
	out := []string{}
	for _, row := range findings {
		sev := strings.ToLower(strings.TrimSpace(row.severity))
		blockerID := strings.TrimSpace(row.blockerID)
		if blockerID == "" || blockerID == "-" {
			continue
		}
		if sev == "sev-1" || sev == "sev-2" || strings.Contains(sev, "blocked") {
			if !set[blockerID] {
				set[blockerID] = true
				out = append(out, blockerID)
			}
		}
	}
	sort.Strings(out)
	return out
}

func rowComplete(values []string) bool {
	if len(values) < 4 {
		return false
	}
	for _, v := range values {
		s := strings.TrimSpace(strings.ToLower(v))
		if s == "" || s == "-" || strings.Contains(s, "<") || strings.Contains(s, ">") || strings.Contains(s, "todo") {
			return false
		}
	}
	return true
}

func printBlocked(path string, issues []string, details map[string]any) {
	payload := map[string]any{
		"status": "BLOCKED",
		"issues": issues,
	}
	if strings.TrimSpace(path) != "" {
		payload["severity_file"] = filepath.ToSlash(path)
	}
	for k, v := range details {
		payload[k] = v
	}
	data, _ := json.Marshal(payload)
	fmt.Println(string(data))
}
