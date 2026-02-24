package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type publishReport struct {
	Status           string            `json:"status"`
	TargetRoot       string            `json:"target_root"`
	ProjectSlug      string            `json:"project_slug"`
	BundleRoot       string            `json:"bundle_root"`
	MissingRequired  []string          `json:"missing_required"`
	MissingOptional  []string          `json:"missing_optional"`
	PublishedTargets map[string]string `json:"published_targets"`
	CopiedFiles      []string          `json:"copied_files"`
	Issues           []string          `json:"issues"`
	TimestampUTC     string            `json:"timestamp_utc"`
}

func main() {
	targetRoot := flag.String("target-root", "", "target repository root (defaults to cwd)")
	archRepoRoot := flag.String("architecture-repo-root", "", "optional architecture standards repo root")
	catalogRepoRoot := flag.String("catalog-repo-root", "", "optional library catalog repo root")
	projectSlug := flag.String("project-slug", "", "optional stable slug for promotion paths")
	allowLocalBundle := flag.Bool("allow-local-bundle", false, "allow PASS without publishing to upstream repos")
	flag.Parse()

	root := strings.TrimSpace(*targetRoot)
	if root == "" {
		cwd, err := os.Getwd()
		if err != nil {
			printBlocked("", "", []string{"unable to determine working directory"}, nil, nil)
			return
		}
		root = cwd
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		printBlocked("", "", []string{"invalid target root"}, nil, nil)
		return
	}

	slug := strings.TrimSpace(*projectSlug)
	if slug == "" {
		slug = normalizeSlug(filepath.Base(absRoot))
	}
	if slug == "" {
		slug = "project"
	}

	archRoot := resolveArgOrEnv(strings.TrimSpace(*archRepoRoot), "ARCHITECTURE_REPO_ROOT")
	catalogRoot := resolveArgOrEnv(strings.TrimSpace(*catalogRepoRoot), "CATALOG_REPO_ROOT")

	bundleRoot := filepath.Join(absRoot, "docs", "tooling", "context-promotion-bundle")
	_ = os.MkdirAll(bundleRoot, 0o755)

	requiredSources := map[string][]string{
		"architecture": {
			"docs/data-architecture-decision.md",
			"docs/integration/compose-mode-decision.md",
		},
		"catalog": {
			"docs/plans/index.md",
			"docs/handoffs/routing-matrix.md",
		},
	}
	optionalSources := map[string][]string{
		"architecture": {
			"docs/adr/ADR-0001-repo-naming-conventions.md",
		},
		"catalog": {
			"docs/traceability-matrix.md",
		},
	}

	report := publishReport{
		Status:           "PASS",
		TargetRoot:       filepath.ToSlash(absRoot),
		ProjectSlug:      slug,
		BundleRoot:       filepath.ToSlash(bundleRoot),
		PublishedTargets: map[string]string{},
		CopiedFiles:      []string{},
		MissingRequired:  []string{},
		MissingOptional:  []string{},
		Issues:           []string{},
		TimestampUTC:     time.Now().UTC().Format(time.RFC3339),
	}

	bundleFiles := map[string][]string{"architecture": {}, "catalog": {}}
	for domain, rels := range requiredSources {
		for _, rel := range rels {
			src := filepath.Join(absRoot, filepath.FromSlash(rel))
			if !isFile(src) {
				report.MissingRequired = append(report.MissingRequired, rel)
				continue
			}
			dst := filepath.Join(bundleRoot, domain, filepath.Base(src))
			if copyErr := copyFile(src, dst); copyErr != nil {
				report.Issues = append(report.Issues, fmt.Sprintf("failed to copy %s: %v", rel, copyErr))
				continue
			}
			bundleFiles[domain] = append(bundleFiles[domain], dst)
			report.CopiedFiles = append(report.CopiedFiles, filepath.ToSlash(dst))
		}
	}
	for domain, rels := range optionalSources {
		for _, rel := range rels {
			src := filepath.Join(absRoot, filepath.FromSlash(rel))
			if !isFile(src) {
				report.MissingOptional = append(report.MissingOptional, rel)
				continue
			}
			dst := filepath.Join(bundleRoot, domain, filepath.Base(src))
			if copyErr := copyFile(src, dst); copyErr != nil {
				report.Issues = append(report.Issues, fmt.Sprintf("failed to copy optional %s: %v", rel, copyErr))
				continue
			}
			bundleFiles[domain] = append(bundleFiles[domain], dst)
			report.CopiedFiles = append(report.CopiedFiles, filepath.ToSlash(dst))
		}
	}

	if len(report.MissingRequired) > 0 {
		sort.Strings(report.MissingRequired)
		report.Status = "BLOCKED"
		report.Issues = append(report.Issues, "required promotion sources missing")
	}

	publishedAny := false
	if archRoot != "" {
		absArch, _ := filepath.Abs(archRoot)
		target := filepath.Join(absArch, "docs", "source", "02-architecture", "promotions", slug)
		if publishFiles(bundleFiles["architecture"], target, &report) {
			report.PublishedTargets["architecture"] = filepath.ToSlash(target)
			publishedAny = true
		}
	}
	if catalogRoot != "" {
		absCatalog, _ := filepath.Abs(catalogRoot)
		target := filepath.Join(absCatalog, "docs", "artifacts", "promotions", slug)
		if publishFiles(bundleFiles["catalog"], target, &report) {
			report.PublishedTargets["catalog"] = filepath.ToSlash(target)
			publishedAny = true
		}
	}

	if !publishedAny && !*allowLocalBundle {
		report.Status = "BLOCKED"
		report.Issues = append(report.Issues, "no upstream publish target configured; set ARCHITECTURE_REPO_ROOT and CATALOG_REPO_ROOT or use --allow-local-bundle")
	}

	reportPath := filepath.Join(absRoot, "docs", "tooling", "context-promotion-report.json")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err == nil {
		if data, err := json.MarshalIndent(report, "", "  "); err == nil {
			_ = os.WriteFile(reportPath, data, 0o644)
		}
	}

	printReport(report)
}

func resolveArgOrEnv(value, envKey string) string {
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return strings.TrimSpace(os.Getenv(envKey))
}

func normalizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	b := strings.Builder{}
	lastDash := false
	for _, r := range value {
		valid := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
		if valid {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteRune('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func publishFiles(files []string, target string, report *publishReport) bool {
	if len(files) == 0 {
		report.Issues = append(report.Issues, fmt.Sprintf("no files available for publish target %s", filepath.ToSlash(target)))
		return false
	}
	if err := os.MkdirAll(target, 0o755); err != nil {
		report.Issues = append(report.Issues, fmt.Sprintf("cannot create publish target %s: %v", filepath.ToSlash(target), err))
		return false
	}
	for _, file := range files {
		dst := filepath.Join(target, filepath.Base(file))
		if err := copyFile(file, dst); err != nil {
			report.Issues = append(report.Issues, fmt.Sprintf("failed to publish %s: %v", filepath.ToSlash(file), err))
			continue
		}
		report.CopiedFiles = append(report.CopiedFiles, filepath.ToSlash(dst))
	}
	return true
}

func printBlocked(root, slug string, issues []string, missingRequired []string, missingOptional []string) {
	report := publishReport{
		Status:           "BLOCKED",
		TargetRoot:       filepath.ToSlash(root),
		ProjectSlug:      slug,
		MissingRequired:  missingRequired,
		MissingOptional:  missingOptional,
		Issues:           issues,
		PublishedTargets: map[string]string{},
		CopiedFiles:      []string{},
		TimestampUTC:     time.Now().UTC().Format(time.RFC3339),
	}
	printReport(report)
}

func printReport(report publishReport) {
	data, _ := json.Marshal(report)
	fmt.Println(string(data))
}
