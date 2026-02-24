package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	targetRoot := flag.String("target-root", "", "absolute path to target repository root")
	profileFile := flag.String("profile-file", "", "optional explicit profile file path")
	outPath := flag.String("out", "docs/planning-behavior-resolution.md", "output file path (absolute or relative to target-root)")
	flag.Parse()

	if strings.TrimSpace(*targetRoot) == "" {
		fatalf("--target-root is required")
	}

	absTargetRoot, err := filepath.Abs(*targetRoot)
	if err != nil {
		fatalf("failed to resolve --target-root: %v", err)
	}

	profilePath, err := resolveProfile(absTargetRoot, strings.TrimSpace(*profileFile))
	if err != nil {
		fatalf("%v", err)
	}

	resolvedOut := strings.TrimSpace(*outPath)
	if resolvedOut == "" {
		fatalf("--out cannot be empty")
	}
	if !filepath.IsAbs(resolvedOut) {
		resolvedOut = filepath.Join(absTargetRoot, resolvedOut)
	}

	if err := os.MkdirAll(filepath.Dir(resolvedOut), 0o755); err != nil {
		fatalf("failed to create output directory: %v", err)
	}

	profileValues, err := readTopLevelScalars(profilePath)
	if err != nil {
		fatalf("failed to read profile file: %v", err)
	}

	report := buildReport(absTargetRoot, profilePath, profileValues)
	if err := os.WriteFile(resolvedOut, []byte(report), 0o644); err != nil {
		fatalf("failed to write output: %v", err)
	}

	fmt.Println(resolvedOut)
}

func resolveProfile(targetRoot, explicit string) (string, error) {
	if explicit != "" {
		if exists(explicit) {
			return explicit, nil
		}
		joined := filepath.Join(targetRoot, explicit)
		if exists(joined) {
			return joined, nil
		}
	}

	candidates := []string{
		filepath.Join(targetRoot, "docs", "source", "02-architecture", "planning-behavior-profile.yaml"),
		filepath.Join(targetRoot, "docs", "source", "DemoArchitectureDocs", "planning-behavior-profile.yaml"),
		filepath.Join(targetRoot, ".github", "skills", "local-mcp-setup", "corporate-docs", "planning-behavior-profile.yaml"),
	}

	for _, candidate := range candidates {
		if exists(candidate) {
			return candidate, nil
		}
	}

	return "", errors.New("planning behavior profile not found")
}

func exists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func readTopLevelScalars(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := map[string]string{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		if strings.HasPrefix(strings.TrimLeft(line, " \t"), "-") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if indent != 0 {
			continue
		}

		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		value = strings.Trim(value, "\"'")
		if key != "" {
			result[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func scalar(m map[string]string, key string) string {
	if v, ok := m[key]; ok && strings.TrimSpace(v) != "" {
		return v
	}
	return "<unset>"
}

func relPath(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}

func buildReport(targetRoot, profilePath string, values map[string]string) string {
	now := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf(`# Planning Behavior Resolution

- Timestamp (UTC): %s
- Target root: %s
- Profile source: %s

## Resolved controls
- profile_id: %s
- topology.default: %s
- topology.service_per_repo_best_practice: %s
- contracts.contract_first_required: %s
- contracts.require_openapi_for_http: %s
- workstreams.dependency_graph_required: %s
- reviews.code_review_feedback_loop_required: %s
- integration.orchestration_requires_all_prerequisites_passed: %s
- evidence.ai_usage_metrics_required_when_available: %s

## Notes
- Values are resolved from the selected profile source for this run.
- Missing scalar values are marked as <unset> and should be treated as planning blockers where required by workflow gates.
`,
		now,
		targetRoot,
		relPath(targetRoot, profilePath),
		scalar(values, "profile_id"),
		scalar(values, "default"),
		scalar(values, "service_per_repo_best_practice"),
		scalar(values, "contract_first_required"),
		scalar(values, "require_openapi_for_http"),
		scalar(values, "dependency_graph_required"),
		scalar(values, "code_review_feedback_loop_required"),
		scalar(values, "orchestration_requires_all_prerequisites_passed"),
		scalar(values, "ai_usage_metrics_required_when_available"),
	)
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
