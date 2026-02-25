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

	out := map[string]string{}
	parents := []string{}
	indents := []int{}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "-") {
			continue
		}

		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		for len(indents) > 0 && indent <= indents[len(indents)-1] {
			indents = indents[:len(indents)-1]
			parents = parents[:len(parents)-1]
		}

		idx := strings.Index(trimmed, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(trimmed[:idx])
		value := strings.TrimSpace(trimmed[idx+1:])

		fullPath := key
		if len(parents) > 0 {
			fullPath = strings.Join(append(append([]string{}, parents...), key), ".")
		}

		if value == "" {
			parents = append(parents, key)
			indents = append(indents, indent)
			continue
		}
		out[fullPath] = strings.Trim(value, "\"'")
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
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
- plan_storage_mode: %s
- plan_directory: %s
- plan_index_file: %s
- plan_story_file_pattern: %s
- plan_traceability_required: %s
- topology.default: %s
- topology.service_per_repo_best_practice: %s
- contracts.contract_first_required: %s
- contracts.require_openapi_for_http: %s
- workstreams.dependency_graph_required: %s
- reviews.code_review_feedback_loop_required: %s
- integration.orchestration_requires_all_prerequisites_passed: %s
- evidence.ai_usage_metrics_required_when_available: %s
- storage.strategy_required: %s
- storage.primary_system_of_record: %s
- storage.cache_policy_required: %s
- storage.object_storage_for_blobs_required: %s
- storage.search_index_for_discovery_allowed: %s
- eventing.async_eventing_policy: %s
- eventing.delivery_semantics_default: %s
- eventing.ordering_scope_default: %s
- eventing.schema_versioning_required: %s
- eventing.idempotent_consumers_required: %s
- production.scalability_budget_required: %s
- production.performance_slo_required: %s
- production.maintainability_controls_required: %s
- production.upgrade_strategy_required: %s
- production.zero_downtime_upgrades_required: %s
- production.operability_slos_required: %s

## Notes
- Values are resolved from the selected profile source for this run.
- Missing scalar values are marked as <unset> and should be treated as planning blockers where required by workflow gates.
`,
		now,
		targetRoot,
		relPath(targetRoot, profilePath),
		scalar(values, "profile_id"),
		scalar(values, "plan_storage_mode"),
		scalar(values, "plan_directory"),
		scalar(values, "plan_index_file"),
		scalar(values, "plan_story_file_pattern"),
		scalar(values, "plan_traceability_required"),
		scalar(values, "topology.default"),
		scalar(values, "topology.service_per_repo_best_practice"),
		scalar(values, "contracts.contract_first_required"),
		scalar(values, "contracts.require_openapi_for_http"),
		scalar(values, "workstreams.dependency_graph_required"),
		scalar(values, "reviews.code_review_feedback_loop_required"),
		scalar(values, "integration.orchestration_requires_all_prerequisites_passed"),
		scalar(values, "evidence.ai_usage_metrics_required_when_available"),
		scalar(values, "storage.strategy_required"),
		scalar(values, "storage.primary_system_of_record"),
		scalar(values, "storage.cache_policy_required"),
		scalar(values, "storage.object_storage_for_blobs_required"),
		scalar(values, "storage.search_index_for_discovery_allowed"),
		scalar(values, "eventing.async_eventing_policy"),
		scalar(values, "eventing.delivery_semantics_default"),
		scalar(values, "eventing.ordering_scope_default"),
		scalar(values, "eventing.schema_versioning_required"),
		scalar(values, "eventing.idempotent_consumers_required"),
		scalar(values, "production.scalability_budget_required"),
		scalar(values, "production.performance_slo_required"),
		scalar(values, "production.maintainability_controls_required"),
		scalar(values, "production.upgrade_strategy_required"),
		scalar(values, "production.zero_downtime_upgrades_required"),
		scalar(values, "production.operability_slos_required"),
	)
}

func fatalf(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(2)
}
