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
	enforceSequence := flag.Bool("enforce-sequence", false, "require prior phase gate PASS before validating current phase")
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
		require("docs/plans/index.md")
	case "02":
		require("docs/requirements.md")
		require("docs/repo-topology-decision.md")
		require("docs/openapi-contract-plan.md")
		require("docs/data-architecture-decision.md")
	case "03":
		require("docs/implementation-plan.md")
		require("docs/technology-constraints.md")
		require("docs/plans/planning-signoff.md")
		require("docs/plans/control-applicability-matrix.md")
		require("docs/handoffs/architect/phase-gate.md")
		if requiresOpenAPISpec(absRoot) {
			if !hasOpenAPISpec(absRoot) {
				missing = append(missing, "docs/openapi/*.yaml")
			}
		}
		if !hasApprovedPlanningSignoff(absRoot) {
			missing = append(missing, "docs/plans/planning-signoff.md (must include Approval Status: APPROVED)")
		}
		if !hasApprovedControlApplicabilityMatrix(absRoot) {
			missing = append(missing, "docs/plans/control-applicability-matrix.md (must include APPLICABLE/NOT-APPLICABLE rows with Approval Status: APPROVED)")
		}
		if requiresModelBoundaryClassification(absRoot) && !hasApprovedModelBoundaryClassification(absRoot) {
			missing = append(missing, "docs/plans/model-boundary-classification.md (required and must include Approval Status: APPROVED when shared-module stream exists)")
		}
		if requiresSharedContractOwnership(absRoot) && !hasApprovedSharedContractOwnership(absRoot) {
			missing = append(missing, "docs/openapi-contract-ownership.md (required and must include Approval Status: APPROVED for shared client/server contract dependency)")
		}
		if requiresIntentControlAccountability(absRoot) && !hasApprovedIntentControlAccountability(absRoot) {
			missing = append(missing, "docs/plans/intent-control-accountability.md (required for PARTIAL/SKIPPED controls with owner/remediation/closure and Approval Status: APPROVED)")
		}
	case "04":
		require("docs/handoffs/dev/phase-gate.md")
		require("docs/tooling/mcp-usage-evidence.md")
		require("docs/integration/compose-mode-decision.md")
		missing = append(missing, repoDocumentationMaturityMissing(absRoot)...)
	case "05":
		require("docs/handoffs/release/phase-gate.md")
		require("docs/handoffs/routing-matrix.md")
		require("docs/data-architecture-decision.md")
		require("docs/handoffs/traceability-pack.md")
		missing = append(missing, repoDocumentationMaturityMissing(absRoot)...)
		missing = append(missing, repoTraceabilityBundleMissing(absRoot)...)
	default:
		printBlocked(*phase, []string{"unsupported phase value"})
		return
	}

	if *enforceSequence {
		if priorIssue := priorPhaseGateIssue(absRoot, strings.TrimSpace(*phase)); priorIssue != "" {
			missing = append(missing, priorIssue)
		}
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

func priorPhaseGateIssue(root, phase string) string {
	priorGateByPhase := map[string]string{
		"02": "docs/handoffs/po/phase-gate.md",
		"03": "docs/handoffs/architect/phase-gate.md",
		"04": "docs/handoffs/dev/phase-gate.md",
		"05": "docs/handoffs/release/phase-gate.md",
	}
	gateRel, ok := priorGateByPhase[phase]
	if !ok {
		return ""
	}
	gatePath := filepath.Join(root, filepath.FromSlash(gateRel))
	content, err := os.ReadFile(gatePath)
	if err != nil {
		return gateRel
	}
	if !strings.Contains(strings.ToUpper(string(content)), "PASS") {
		return gateRel + " (must indicate PASS when --enforce-sequence=true)"
	}
	return ""
}

func repoDocumentationMaturityMissing(root string) []string {
	missing := []string{}
	reposRoot := filepath.Join(root, "repos")
	entries, err := os.ReadDir(reposRoot)
	if err != nil {
		return missing
	}
	requiredReadmeHeadings := []string{"## Purpose", "## Scope", "## Runbook", "## Interfaces", "## Ownership", "## Traceability", "## Plan-to-Implementation Summary"}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoPath := filepath.Join(reposRoot, entry.Name())
		readmePath := filepath.Join(repoPath, "README.md")
		readmeContent, readmeErr := os.ReadFile(readmePath)
		if readmeErr != nil {
			missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "README.md")))
		} else {
			readmeText := string(readmeContent)
			if strings.Contains(readmeText, "Starter scaffold repository generated by project bootstrap") {
				missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "README.md (replace scaffold-only content)")))
			}
			if strings.Contains(readmeText, "<repo-name>") {
				missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "README.md (replace template placeholders)")))
			}
			for _, heading := range requiredReadmeHeadings {
				if !strings.Contains(readmeText, heading) {
					missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "README.md missing section "+heading)))
				}
			}
		}

		changelogPath := filepath.Join(repoPath, "CHANGELOG.md")
		changelogContent, changelogErr := os.ReadFile(changelogPath)
		if changelogErr != nil {
			missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "CHANGELOG.md")))
		} else {
			changelogText := string(changelogContent)
			if strings.Contains(changelogText, "Initial scaffold") {
				missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "CHANGELOG.md (replace scaffold-only content)")))
			}
			if strings.Contains(changelogText, "<version/tag>") || strings.Contains(changelogText, "<change summary>") {
				missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "CHANGELOG.md (replace template placeholders)")))
			}
			if !strings.Contains(changelogText, "### Plan Reference") {
				missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "CHANGELOG.md missing section ### Plan Reference")))
			}
		}

		summaryPath := filepath.Join(repoPath, "docs", "current-state", "implementation-summary.md")
		summaryContent, summaryErr := os.ReadFile(summaryPath)
		if summaryErr != nil {
			missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "docs", "current-state", "implementation-summary.md")))
		} else {
			summaryText := string(summaryContent)
			if !strings.Contains(summaryText, "Plan intent") || !strings.Contains(summaryText, "Key implementation details") {
				missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "docs", "current-state", "implementation-summary.md missing required sections")))
			}
			if strings.Contains(summaryText, "## Plan intent\n-") || strings.Contains(summaryText, "## Key implementation details\n-") {
				missing = append(missing, filepath.ToSlash(filepath.Join("repos", entry.Name(), "docs", "current-state", "implementation-summary.md (replace placeholder bullets)")))
			}
		}
	}
	return missing
}

func repoTraceabilityBundleMissing(root string) []string {
	missing := []string{}
	reposRoot := filepath.Join(root, "repos")
	entries, err := os.ReadDir(reposRoot)
	if err != nil {
		return missing
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoName := entry.Name()
		repoPath := filepath.Join(reposRoot, repoName)
		requireRel := []string{
			filepath.ToSlash(filepath.Join("repos", repoName, "docs", "handoffs", "traceability-pack.md")),
			filepath.ToSlash(filepath.Join("repos", repoName, "docs", "diagrams", "high-level.mmd")),
		}
		for _, rel := range requireRel {
			if info, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); statErr != nil || info.IsDir() {
				missing = append(missing, rel)
			}
		}
		packPath := filepath.Join(repoPath, "docs", "handoffs", "traceability-pack.md")
		if content, readErr := os.ReadFile(packPath); readErr == nil {
			text := string(content)
			requiredSnippets := []string{"## ID Inventory", "## Mapping", "## ADR Ledger", "## System Description", "## Diagram Index"}
			for _, snippet := range requiredSnippets {
				if !strings.Contains(text, snippet) {
					missing = append(missing, filepath.ToSlash(filepath.Join("repos", repoName, "docs", "handoffs", "traceability-pack.md missing section "+snippet)))
				}
			}
		}
	}
	return missing
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

func hasApprovedPlanningSignoff(root string) bool {
	path := filepath.Join(root, "docs", "plans", "planning-signoff.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(strings.ToUpper(string(content)), "APPROVAL STATUS: APPROVED")
}

func hasApprovedControlApplicabilityMatrix(root string) bool {
	path := filepath.Join(root, "docs", "plans", "control-applicability-matrix.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := strings.ToUpper(string(content))
	hasApplicability := strings.Contains(text, "APPLICABLE") && strings.Contains(text, "NOT-APPLICABLE")
	hasApproved := strings.Contains(text, "APPROVAL STATUS") && strings.Contains(text, "APPROVED")
	hasOwner := strings.Contains(text, "OWNER")
	hasRationale := strings.Contains(text, "RATIONALE")
	return hasApplicability && hasApproved && hasOwner && hasRationale
}

func requiresModelBoundaryClassification(root string) bool {
	paths := []string{
		filepath.Join(root, "docs", "plans", "repo-change-plan.md"),
		filepath.Join(root, "docs", "implementation-plan.md"),
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := strings.ToLower(string(content))
		if strings.Contains(text, "shared module") || strings.Contains(text, "shared-module") || strings.Contains(text, "shared contract dto") {
			return true
		}
	}
	return false
}

func hasApprovedModelBoundaryClassification(root string) bool {
	path := filepath.Join(root, "docs", "plans", "model-boundary-classification.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := strings.ToUpper(string(content))
	hasClassification := strings.Contains(text, "DOMAIN_MODEL_INTERNAL") || strings.Contains(text, "SHARED_CONTRACT_DTO_DAO")
	hasApproved := strings.Contains(text, "APPROVAL STATUS") && strings.Contains(text, "APPROVED")
	return hasClassification && hasApproved
}

func requiresSharedContractOwnership(root string) bool {
	path := filepath.Join(root, "docs", "openapi-contract-plan.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := strings.ToLower(string(content))
	hasClient := strings.Contains(text, "client")
	hasServer := strings.Contains(text, "server")
	hasShared := strings.Contains(text, "shared") || strings.Contains(text, "same contract")
	return hasClient && hasServer && hasShared
}

func hasApprovedSharedContractOwnership(root string) bool {
	path := filepath.Join(root, "docs", "openapi-contract-ownership.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := strings.ToUpper(string(content))
	hasBoundary := strings.Contains(text, "CONTRACT BOUNDARY TYPE")
	hasSource := strings.Contains(text, "SOURCE OF TRUTH")
	hasApproved := strings.Contains(text, "APPROVAL STATUS") && strings.Contains(text, "APPROVED")
	return hasBoundary && hasSource && hasApproved
}

func requiresIntentControlAccountability(root string) bool {
	path := filepath.Join(root, "docs", "plans", "control-applicability-matrix.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := strings.ToUpper(string(content))
	return strings.Contains(text, "PARTIAL") || strings.Contains(text, "SKIPPED")
}

func hasApprovedIntentControlAccountability(root string) bool {
	path := filepath.Join(root, "docs", "plans", "intent-control-accountability.md")
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	text := strings.ToUpper(string(content))
	hasRequiredFields := strings.Contains(text, "OWNER") && strings.Contains(text, "REMEDIATION") && strings.Contains(text, "TARGET CLOSURE PHASE")
	hasApproved := strings.Contains(text, "APPROVAL STATUS") && strings.Contains(text, "APPROVED")
	return hasRequiredFields && hasApproved
}

func printBlocked(phase string, missing []string) {
	payload, _ := json.Marshal(map[string]any{
		"status":  "BLOCKED",
		"phase":   strings.TrimSpace(phase),
		"missing": missing,
	})
	fmt.Println(string(payload))
}
