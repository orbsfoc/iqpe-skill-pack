package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var adapterLinePattern = regexp.MustCompile(`(?i)(adapter_id|adaptor_id)\s*:\s*([a-zA-Z0-9._-]+)`)

func main() {
	targetRoot := flag.String("target-root", "", "target repository root (defaults to cwd)")
	tcFile := flag.String("tc-file", "docs/technology-constraints.md", "technology constraints file path")
	flag.Parse()

	root := strings.TrimSpace(*targetRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			printBlocked([]string{"unable to determine working directory"}, nil, nil)
			return
		}
		root = cwd
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		printBlocked([]string{"invalid target root"}, nil, nil)
		return
	}

	absTC := strings.TrimSpace(*tcFile)
	if absTC == "" {
		printBlocked([]string{"--tc-file cannot be empty"}, nil, nil)
		return
	}
	if !filepath.IsAbs(absTC) {
		absTC = filepath.Join(absRoot, filepath.FromSlash(absTC))
	}

	expected, err := parseExpectedAdapters(absTC)
	if err != nil {
		printBlocked([]string{fmt.Sprintf("failed to parse expected adapters: %v", err)}, nil, nil)
		return
	}
	if len(expected) == 0 {
		printBlocked([]string{"no adapter_id/adaptor_id entries found in technology constraints"}, nil, nil)
		return
	}

	implemented := discoverImplementedAdapters(absRoot)

	missing := diff(expected, implemented)
	undeclared := diff(implemented, expected)

	if len(missing) > 0 {
		issues := []string{"declared adapters missing implementation"}
		printBlocked(issues, missing, undeclared)
		return
	}

	printPass(expected, implemented, undeclared)
}

func parseExpectedAdapters(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	seen := map[string]bool{}
	out := []string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		match := adapterLinePattern.FindStringSubmatch(line)
		if len(match) < 3 {
			continue
		}
		id := strings.ToLower(strings.TrimSpace(match[2]))
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	sort.Strings(out)
	return out, nil
}

func discoverImplementedAdapters(root string) []string {
	candidateRoots := []string{
		filepath.Join(root, "adapters"),
		filepath.Join(root, "src", "adapters"),
		filepath.Join(root, "internal", "adapters"),
		filepath.Join(root, "pkg", "adapters"),
	}

	reposRoot := filepath.Join(root, "repos")
	if repoEntries, repoErr := os.ReadDir(reposRoot); repoErr == nil {
		for _, repoEntry := range repoEntries {
			if !repoEntry.IsDir() {
				continue
			}
			repoRoot := filepath.Join(reposRoot, repoEntry.Name())
			candidateRoots = append(candidateRoots,
				filepath.Join(repoRoot, "adapters"),
				filepath.Join(repoRoot, "src", "adapters"),
				filepath.Join(repoRoot, "internal", "adapters"),
				filepath.Join(repoRoot, "pkg", "adapters"),
			)
		}
	}

	seen := map[string]bool{}
	out := []string{}
	for _, candidate := range candidateRoots {
		entries, err := os.ReadDir(candidate)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := strings.ToLower(strings.TrimSpace(entry.Name()))
			if name == "" || seen[name] {
				continue
			}
			seen[name] = true
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

func diff(a, b []string) []string {
	bset := map[string]bool{}
	for _, item := range b {
		bset[item] = true
	}
	out := []string{}
	for _, item := range a {
		if !bset[item] {
			out = append(out, item)
		}
	}
	return out
}

func printBlocked(issues, missing, undeclared []string) {
	payload, _ := json.Marshal(map[string]any{
		"status":              "BLOCKED",
		"issues":              issues,
		"missing_adapters":    missing,
		"undeclared_adapters": undeclared,
	})
	fmt.Println(string(payload))
}

func printPass(expected, implemented, undeclared []string) {
	payload, _ := json.Marshal(map[string]any{
		"status":               "PASS",
		"expected_adapters":    expected,
		"implemented_adapters": implemented,
		"undeclared_adapters":  undeclared,
	})
	fmt.Println(string(payload))
}
