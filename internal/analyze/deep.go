package analyze

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	maxOrientationDocuments = 40
	maxDocumentsPerDir      = 10
	maxPromptSurfaces       = 200
	maxTotalPromptBytes     = 4194304
	maxEvidenceRunes        = 240
	maxAuditFindings        = 200
)

type DimensionState string

const (
	DimensionReady       DimensionState = "ready"
	DimensionPartial     DimensionState = "partial"
	DimensionMissing     DimensionState = "missing"
	DimensionConflicting DimensionState = "conflicting"
)

type ReadinessStatus string

const (
	ReadinessReady    ReadinessStatus = "ready"
	ReadinessPartial  ReadinessStatus = "partial"
	ReadinessNotReady ReadinessStatus = "not_ready"
)

type ClaimStatus string

const (
	ClaimConfirmed    ClaimStatus = "confirmed"
	ClaimContradicted ClaimStatus = "contradicted"
	ClaimNotChecked   ClaimStatus = "not_checked"
)

type Finding struct {
	FindingID       string     `json:"finding_id"`
	RuleID          string     `json:"rule_id"`
	Title           string     `json:"title"`
	Category        string     `json:"category"`
	Severity        string     `json:"severity"`
	Confidence      Confidence `json:"confidence"`
	Path            string     `json:"path,omitempty"`
	Line            int        `json:"line,omitempty"`
	Evidence        string     `json:"evidence"`
	Message         string     `json:"message"`
	SuggestedAction string     `json:"suggested_action"`
}

type Claim struct {
	ClaimID string      `json:"claim_id"`
	Status  ClaimStatus `json:"status"`
	Path    string      `json:"path,omitempty"`
	Reason  string      `json:"reason"`
}

type ReadinessDimensions struct {
	Context      DimensionState `json:"context"`
	Operability  DimensionState `json:"operability"`
	Safety       DimensionState `json:"safety"`
	Verification DimensionState `json:"verification"`
}

type AgentReadiness struct {
	Status     ReadinessStatus     `json:"status"`
	Mode       string              `json:"mode"`
	Dimensions ReadinessDimensions `json:"dimensions"`
	Claims     []Claim             `json:"claims"`
	Findings   []Finding           `json:"findings"`
}

type AuditNote struct {
	Path   string `json:"path,omitempty"`
	Reason string `json:"reason"`
}

type DeepAudit struct {
	PromptSurfaces int         `json:"prompt_surfaces"`
	DriftFindings  []Finding   `json:"drift_findings"`
	PromptFindings []Finding   `json:"prompt_findings"`
	Truncations    []AuditNote `json:"truncations"`
	Skipped        []AuditNote `json:"skipped"`
}

type inspectedDocument struct {
	Path string
	Body string
}

func analyzeDeep(root string, result AnalysisResult) (AgentReadiness, DeepAudit, error) {
	inventory, err := inspectRepository(context.Background(), root)
	if err != nil {
		return AgentReadiness{}, DeepAudit{}, err
	}
	return analyzeDeepWithInventory(context.Background(), root, result, inventory)
}

func analyzeDeepWithInventory(ctx context.Context, root string, result AnalysisResult, inventory repositoryInventory) (AgentReadiness, DeepAudit, error) {
	paths := inventory.Paths
	audit := DeepAudit{
		DriftFindings:  []Finding{},
		PromptFindings: []Finding{},
		Truncations:    []AuditNote{},
		Skipped:        []AuditNote{},
	}
	audit.Skipped = append(audit.Skipped, inventory.Skipped...)
	audit.Truncations = append(audit.Truncations, inventory.Truncations...)

	promptCandidates := filterPaths(paths, isPromptSurface)
	if len(promptCandidates) > maxPromptSurfaces {
		audit.Truncations = append(audit.Truncations, AuditNote{Reason: fmt.Sprintf("prompt surface limit reached: %d of %d", maxPromptSurfaces, len(promptCandidates))})
		promptCandidates = promptCandidates[:maxPromptSurfaces]
	}
	audit.PromptSurfaces = len(promptCandidates)
	promptDocs, usedBytes, err := inspectDocuments(ctx, root, promptCandidates, maxTotalPromptBytes, &audit)
	if err != nil {
		return AgentReadiness{}, DeepAudit{}, err
	}
	_ = usedBytes

	orientationCandidates, notes := selectOrientationPaths(paths)
	audit.Truncations = append(audit.Truncations, notes...)
	orientationDocs, _, err := inspectDocuments(ctx, root, orientationCandidates, maxTotalPromptBytes, &audit)
	if err != nil {
		return AgentReadiness{}, DeepAudit{}, err
	}

	audit.PromptFindings = append(audit.PromptFindings, detectDuplicateInstructions(promptDocs)...)
	audit.PromptFindings = append(audit.PromptFindings, detectStructuredConflicts(promptDocs)...)
	audit.PromptFindings = append(audit.PromptFindings, detectVendorCoupling(promptDocs)...)
	audit.DriftFindings = append(audit.DriftFindings, detectStaleReferences(root, paths, mergeInspectedDocuments(orientationDocs, promptDocs))...)
	sortFindings(audit.PromptFindings)
	sortFindings(audit.DriftFindings)
	capAuditFindings(&audit)
	sortAuditNotes(audit.Truncations)
	sortAuditNotes(audit.Skipped)

	readiness := deriveReadiness(root, result, paths, orientationDocs, audit)
	return readiness, audit, nil
}

func inspectDocuments(ctx context.Context, root string, paths []string, totalLimit int64, audit *DeepAudit) ([]inspectedDocument, int64, error) {
	docs := make([]inspectedDocument, 0, len(paths))
	var total int64
	for _, rel := range paths {
		if err := ctx.Err(); err != nil {
			return nil, total, err
		}
		if sensitiveAnalysisPath(rel) {
			audit.Skipped = append(audit.Skipped, AuditNote{Path: rel, Reason: "sensitive path contents not read"})
			continue
		}
		remaining := totalLimit - total
		if remaining <= 0 {
			audit.Truncations = append(audit.Truncations, AuditNote{Path: rel, Reason: "total inspected byte limit reached"})
			continue
		}
		limit := maxInspectedFileBytes
		if remaining < limit {
			limit = remaining
		}
		read, err := safeReadRepositoryFile(root, rel, limit)
		if err != nil {
			if rel == "AGENTS.md" {
				return nil, total, fmt.Errorf("required root contract AGENTS.md cannot be read: %w", err)
			}
			audit.Skipped = append(audit.Skipped, AuditNote{Path: rel, Reason: safeSkipReason(err)})
			continue
		}
		total += int64(len(read.Body))
		if read.Truncated {
			audit.Truncations = append(audit.Truncations, AuditNote{Path: rel, Reason: "file byte limit reached"})
		}
		docs = append(docs, inspectedDocument{Path: rel, Body: string(read.Body)})
	}
	return docs, total, nil
}

func filterPaths(paths []string, match func(string) bool) []string {
	var result []string
	for _, path := range paths {
		if match(path) {
			result = append(result, path)
		}
	}
	sort.Strings(result)
	return result
}

func mergeInspectedDocuments(groups ...[]inspectedDocument) []inspectedDocument {
	byPath := map[string]inspectedDocument{}
	for _, group := range groups {
		for _, document := range group {
			byPath[document.Path] = document
		}
	}
	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	merged := make([]inspectedDocument, 0, len(paths))
	for _, path := range paths {
		merged = append(merged, byPath[path])
	}
	return merged
}

func isPromptSurface(rel string) bool {
	base := filepath.Base(rel)
	ext := strings.ToLower(filepath.Ext(rel))
	switch {
	case base == "AGENTS.md":
		return true
	case rel == "CLAUDE.md" || rel == "CLAUDE.local.md":
		return true
	case strings.HasPrefix(rel, ".claude/rules/") || strings.HasPrefix(rel, ".claude/agents/"):
		return ext == ".md"
	case strings.HasPrefix(rel, ".codex/rules/"):
		return ext == ".md" || ext == ".yaml" || ext == ".yml"
	case strings.HasPrefix(rel, "rules/") && !strings.Contains(strings.TrimPrefix(rel, "rules/"), "/"):
		return ext == ".yaml" || ext == ".yml"
	case strings.HasPrefix(rel, ".opencode/agents/") || strings.HasPrefix(rel, ".opencode/commands/"):
		return ext == ".md"
	case strings.HasPrefix(rel, "prompts/"):
		return allowedPromptExtension(ext)
	case base == "SKILL.md":
		return true
	default:
		return false
	}
}

func allowedPromptExtension(ext string) bool {
	switch ext {
	case ".md", ".txt", ".yaml", ".yml", ".json", ".jsonc":
		return true
	default:
		return false
	}
}

func selectOrientationPaths(paths []string) ([]string, []AuditNote) {
	type candidate struct {
		path  string
		tier  int
		depth int
	}
	var candidates []candidate
	for _, rel := range paths {
		base := filepath.Base(rel)
		lowerBase := strings.ToLower(base)
		tier := 0
		switch {
		case rel == "AGENTS.md", lowerBase == "readme.md", rel == "RUNBOOK.md", rel == "SPEC.md", rel == "Makefile", rel == "package.json", rel == "pyproject.toml", rel == "go.mod":
			tier = 1
		case base == "AGENTS.md":
			tier = 2
		case (strings.HasPrefix(rel, "docs/") || strings.HasPrefix(lowerBase, "architecture") || strings.HasPrefix(lowerBase, "security") || strings.HasPrefix(lowerBase, "testing")) && strings.HasSuffix(lowerBase, ".md"):
			tier = 4
			if orientationRelevantPath(rel) {
				tier = 3
			}
		default:
			continue
		}
		candidates = append(candidates, candidate{path: rel, tier: tier, depth: strings.Count(rel, "/")})
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].tier != candidates[j].tier {
			return candidates[i].tier < candidates[j].tier
		}
		if candidates[i].depth != candidates[j].depth {
			return candidates[i].depth < candidates[j].depth
		}
		return candidates[i].path < candidates[j].path
	})
	counts := map[string]int{}
	var selected []string
	var notes []AuditNote
	for _, candidate := range candidates {
		if len(selected) >= maxOrientationDocuments {
			notes = append(notes, AuditNote{Path: candidate.path, Reason: "orientation document limit reached"})
			continue
		}
		top := strings.Split(candidate.path, "/")[0]
		if candidate.depth > 0 && counts[top] >= maxDocumentsPerDir {
			notes = append(notes, AuditNote{Path: candidate.path, Reason: "per-directory orientation limit reached"})
			continue
		}
		selected = append(selected, candidate.path)
		if candidate.depth > 0 {
			counts[top]++
		}
	}
	return selected, notes
}

func orientationRelevantPath(path string) bool {
	lower := strings.ToLower(filepath.ToSlash(path))
	return containsAny(lower, "agent", "architecture", "contributing", "operability", "operations", "runbook", "safety", "security", "testing", "verification")
}

func deriveReadiness(root string, result AnalysisResult, paths []string, docs []inspectedDocument, audit DeepAudit) AgentReadiness {
	pathSet := map[string]bool{}
	for _, path := range paths {
		pathSet[path] = true
	}
	docMap := make(map[string]string, len(docs))
	for _, doc := range docs {
		docMap[doc.Path] = doc.Body
	}
	hasIdentity, identityPath := discoverIdentity(docMap)
	hasScope := strings.TrimSpace(docMap["AGENTS.md"]) != ""
	identityClaim := makeEvidenceClaim("context.identity", hasIdentity, claimUncertain(audit, identityPath), identityPath, "repository identity is discoverable")
	scopeClaim := makeEvidenceClaim("context.scope", hasScope, claimUncertain(audit, "AGENTS.md"), "AGENTS.md", "root policy scope is discoverable")
	contextState := dimensionFromClaims(identityClaim, scopeClaim)

	hasTestCommand, testCommandPath, hasOtherCommand, otherCommandPath := validatedCommandSignals(root, pathSet, docs)
	testCommandClaim := makeEvidenceClaim("operability.test_command", hasTestCommand, claimUncertain(audit, testCommandPath), testCommandPath, "a statically valid test or verification command is declared")
	otherCommandClaim := makeEvidenceClaim("operability.work_command", hasOtherCommand, claimUncertain(audit, otherCommandPath), otherCommandPath, "a statically valid build, run, or lint command is declared")
	operabilityState := dimensionFromClaims(testCommandClaim, otherCommandClaim)

	hasSafetyPolicy := hasScope && hasSafetyBoundary(docMap["AGENTS.md"])
	hasSecurityRules := pathSet["rules/security-global.yaml"] || pathSet["rules/security-shell.yaml"]
	safetyPolicyClaim := makeEvidenceClaim("safety.boundaries", hasSafetyPolicy, claimUncertain(audit, "AGENTS.md"), "AGENTS.md", "approval and destructive-operation boundaries are explicit")
	securityRulesClaim := makeEvidenceClaim("safety.rules", hasSecurityRules, hasTraversalTruncation(audit), "rules/", "repository security rules are present")
	safetyState := dimensionFromClaims(safetyPolicyClaim, securityRulesClaim)

	hasTests := hasTestingTechnology(result.Technologies) || hasTestPath(paths)
	hasVerificationContract := hasTestCommand || hasPositiveContractPhrase(docs, "acceptance criteria", "deterministic check", "verification required")
	testsClaim := makeEvidenceClaim("verification.tests", hasTests, hasTraversalTruncation(audit), firstTestPath(paths), "test sources or a recognized testing stack are present")
	verificationClaim := makeEvidenceClaim("verification.contract", hasVerificationContract, claimUncertain(audit, testCommandPath), testCommandPath, "deterministic verification is declared")
	verificationState := dimensionFromClaims(testsClaim, verificationClaim)

	conflictFinding := firstFindingByRule(audit.PromptFindings, "PA-002")
	consistencyClaim := Claim{ClaimID: "policy.consistency", Status: ClaimConfirmed, Path: "AGENTS.md", Reason: "no deterministic same-authority conflict found"}
	if conflictFinding != nil {
		consistencyClaim.Status = ClaimContradicted
		consistencyClaim.Path = conflictFinding.Path
		consistencyClaim.Reason = "deterministic same-authority policy conflict found"
		contextState = DimensionConflicting
	} else if promptAuditUncertain(audit) {
		consistencyClaim.Status = ClaimNotChecked
		consistencyClaim.Path = "."
		consistencyClaim.Reason = "prompt audit was skipped or truncated"
		if contextState == DimensionReady {
			contextState = DimensionPartial
		}
	}

	dimensions := ReadinessDimensions{Context: contextState, Operability: operabilityState, Safety: safetyState, Verification: verificationState}
	readiness := AgentReadiness{Mode: "deep", Dimensions: dimensions, Claims: []Claim{identityClaim, scopeClaim, testCommandClaim, otherCommandClaim, safetyPolicyClaim, securityRulesClaim, testsClaim, verificationClaim, consistencyClaim}, Findings: []Finding{}}
	readiness.Findings = append(readiness.Findings, dimensionFinding("AR-CONTEXT", "missing_context_contract", contextState, "Repository identity and scope are not both discoverable.", "Add concise repository identity and scope guidance.")...)
	readiness.Findings = append(readiness.Findings, dimensionFinding("AR-OPERABILITY", "missing_command_contract", operabilityState, "Build, run, lint, or test commands are incompletely documented.", "Document the existing deterministic project commands.")...)
	readiness.Findings = append(readiness.Findings, dimensionFinding("AR-SAFETY", "missing_safety_contract", safetyState, "Safety boundaries or security rules are incomplete.", "Document approval and destructive-operation boundaries.")...)
	readiness.Findings = append(readiness.Findings, dimensionFinding("AR-VERIFICATION", "missing_verification_contract", verificationState, "Tests or verification instructions are incomplete.", "Document and retain deterministic verification checks.")...)

	readiness.Status = readinessStatus(dimensions)
	if (len(audit.Truncations) > 0 || len(audit.Skipped) > 0) && readiness.Status == ReadinessReady {
		readiness.Status = ReadinessPartial
	}
	if len(audit.DriftFindings) > 0 && readiness.Status == ReadinessReady {
		readiness.Status = ReadinessPartial
	}
	sortFindings(readiness.Findings)
	return readiness
}

func promptAuditUncertain(audit DeepAudit) bool {
	for _, note := range append(append([]AuditNote{}, audit.Truncations...), audit.Skipped...) {
		reason := strings.ToLower(note.Reason)
		if strings.Contains(reason, "orientation") {
			continue
		}
		if note.Path != "" && isPromptSurface(note.Path) {
			return true
		}
		if containsAny(reason, "repository entry limit", "repository traversal depth limit", "prompt surface limit", "finding limit") {
			return true
		}
	}
	return false
}

func evidenceState(left bool, right bool) DimensionState {
	if left && right {
		return DimensionReady
	}
	if left || right {
		return DimensionPartial
	}
	return DimensionMissing
}

func makeEvidenceClaim(id string, confirmed bool, uncertain bool, path string, description string) Claim {
	claim := Claim{ClaimID: id, Path: path}
	switch {
	case confirmed:
		claim.Status = ClaimConfirmed
		claim.Reason = description
	case uncertain:
		claim.Status = ClaimNotChecked
		claim.Reason = "evidence was skipped or truncated"
	default:
		claim.Status = ClaimContradicted
		claim.Reason = "required evidence was not discovered"
	}
	if claim.Path == "" {
		claim.Path = "."
	}
	return claim
}

func dimensionFromClaims(claims ...Claim) DimensionState {
	confirmed := 0
	for _, claim := range claims {
		switch claim.Status {
		case ClaimNotChecked:
			return DimensionPartial
		case ClaimConfirmed:
			confirmed++
		}
	}
	if confirmed == len(claims) {
		return DimensionReady
	}
	if confirmed > 0 {
		return DimensionPartial
	}
	return DimensionMissing
}

func discoverIdentity(docs map[string]string) (bool, string) {
	for path, body := range docs {
		if strings.EqualFold(path, "README.md") && strings.TrimSpace(body) != "" {
			return true, path
		}
	}
	if body := docs["go.mod"]; regexp.MustCompile(`(?m)^\s*module\s+\S+`).MatchString(body) {
		return true, "go.mod"
	}
	if body := docs["package.json"]; body != "" {
		var manifest struct {
			Name string `json:"name"`
		}
		if json.Unmarshal([]byte(body), &manifest) == nil && strings.TrimSpace(manifest.Name) != "" {
			return true, "package.json"
		}
	}
	if body := docs["pyproject.toml"]; regexp.MustCompile(`(?m)^\s*name\s*=\s*["'][^"']+["']`).MatchString(body) {
		return true, "pyproject.toml"
	}
	return false, "README.md"
}

func auditUncertain(audit DeepAudit, path string) bool {
	for _, note := range append(append([]AuditNote{}, audit.Truncations...), audit.Skipped...) {
		if note.Path == "" || note.Path == path {
			return true
		}
	}
	return false
}

func claimUncertain(audit DeepAudit, path string) bool {
	return auditUncertain(audit, path) || hasTraversalTruncation(audit)
}

func hasTraversalTruncation(audit DeepAudit) bool {
	for _, note := range audit.Truncations {
		if strings.Contains(note.Reason, "repository") {
			return true
		}
	}
	return false
}

func validatedCommandSignals(root string, pathSet map[string]bool, docs []inspectedDocument) (bool, string, bool, string) {
	makeTargets := discoverMakeTargets(root)
	packageScripts := discoverPackageScripts(root)
	taskTargets := discoverTaskTargets(root)
	testFound, otherFound := false, false
	testPath, otherPath := ".", "."
	makePattern := regexp.MustCompile(`\bmake[ \t]+([A-Za-z0-9_.-]+)\b`)
	packagePattern := regexp.MustCompile(`\b(?:npm|pnpm|yarn)[ \t]+(?:run[ \t]+)?([A-Za-z0-9_.:-]+)\b`)
	taskPattern := regexp.MustCompile(`\b(?:task|just)[ \t]+([A-Za-z0-9_.:-]+)\b`)
	for _, doc := range docs {
		for _, raw := range strings.Split(doc.Body, "\n") {
			line := strings.ToLower(strings.TrimSpace(raw))
			if line == "" || commandLineIsNegated(line) {
				continue
			}
			for _, match := range makePattern.FindAllStringSubmatch(line, -1) {
				if !makeTargets[match[1]] {
					continue
				}
				if commandNameIsVerification(match[1]) {
					testFound, testPath = true, doc.Path
				} else {
					otherFound, otherPath = true, doc.Path
				}
			}
			for _, match := range packagePattern.FindAllStringSubmatch(line, -1) {
				if !packageScripts[match[1]] {
					continue
				}
				if commandNameIsVerification(match[1]) {
					testFound, testPath = true, doc.Path
				} else {
					otherFound, otherPath = true, doc.Path
				}
			}
			for _, match := range taskPattern.FindAllStringSubmatch(line, -1) {
				if !taskTargets[match[1]] {
					continue
				}
				if commandNameIsVerification(match[1]) {
					testFound, testPath = true, doc.Path
				} else {
					otherFound, otherPath = true, doc.Path
				}
			}
			if pathSet["go.mod"] && regexp.MustCompile(`\bgo[ \t]+test\b`).MatchString(line) {
				testFound, testPath = true, doc.Path
			}
			if pathSet["go.mod"] && regexp.MustCompile(`\bgo[ \t]+(?:build|run|vet)\b`).MatchString(line) {
				otherFound, otherPath = true, doc.Path
			}
		}
	}
	return testFound, testPath, otherFound, otherPath
}

func discoverPackageScripts(root string) map[string]bool {
	result := map[string]bool{}
	read, err := safeReadRepositoryFile(root, "package.json", maxInspectedFileBytes)
	if err != nil || read.Truncated {
		return result
	}
	var manifest struct {
		Scripts map[string]json.RawMessage `json:"scripts"`
	}
	if json.Unmarshal(read.Body, &manifest) != nil {
		return result
	}
	for name, raw := range manifest.Scripts {
		var command string
		if json.Unmarshal(raw, &command) == nil && strings.TrimSpace(command) != "" {
			result[strings.ToLower(name)] = true
		}
	}
	return result
}

func discoverTaskTargets(root string) map[string]bool {
	result := map[string]bool{}
	for _, path := range []string{"Taskfile.yml", "Taskfile.yaml", "Justfile", "justfile"} {
		read, err := safeReadRepositoryFile(root, path, maxInspectedFileBytes)
		if err != nil || read.Truncated {
			continue
		}
		inTasks := strings.HasPrefix(strings.ToLower(path), "just")
		for _, raw := range strings.Split(string(read.Body), "\n") {
			if strings.TrimSpace(raw) == "tasks:" {
				inTasks = true
				continue
			}
			if !inTasks || strings.TrimSpace(raw) == "" || strings.HasPrefix(strings.TrimSpace(raw), "#") {
				continue
			}
			if strings.HasPrefix(strings.ToLower(path), "taskfile") {
				indent := len(raw) - len(strings.TrimLeft(raw, " "))
				if indent != 2 {
					continue
				}
			}
			trimmed := strings.TrimSpace(raw)
			if index := strings.Index(trimmed, ":"); index > 0 {
				name := strings.ToLower(strings.TrimSpace(trimmed[:index]))
				if regexp.MustCompile(`^[a-z0-9_.-]+$`).MatchString(name) {
					result[name] = true
				}
			}
		}
	}
	return result
}

func discoverCLICommands(root string, paths []string) (map[string]bool, map[string]bool) {
	binaries := map[string]bool{}
	commands := map[string]bool{}
	for _, path := range paths {
		parts := strings.Split(path, "/")
		if len(parts) == 2 && parts[0] == "cmd" {
			binaries[strings.ToLower(parts[1])] = true
		}
	}
	read, err := safeReadRepositoryFile(root, "package.json", maxInspectedFileBytes)
	if err == nil && !read.Truncated {
		var manifest struct {
			Name string          `json:"name"`
			Bin  json.RawMessage `json:"bin"`
		}
		if json.Unmarshal(read.Body, &manifest) == nil {
			var single string
			if json.Unmarshal(manifest.Bin, &single) == nil && strings.TrimSpace(single) != "" && strings.TrimSpace(manifest.Name) != "" {
				binaries[strings.ToLower(manifest.Name)] = true
			} else {
				var named map[string]json.RawMessage
				if json.Unmarshal(manifest.Bin, &named) == nil {
					for name := range named {
						binaries[strings.ToLower(name)] = true
					}
				}
			}
		}
	}
	casePattern := regexp.MustCompile(`(?m)\bcase\s+"([A-Za-z0-9][A-Za-z0-9_.:-]*)"\s*:`)
	for _, path := range paths {
		if filepath.Ext(path) != ".go" || (filepath.Base(path) != "root.go" && filepath.Base(path) != "main.go") {
			continue
		}
		read, err := safeReadRepositoryFile(root, path, maxInspectedFileBytes)
		if err != nil || read.Truncated {
			continue
		}
		for _, match := range casePattern.FindAllStringSubmatch(string(read.Body), -1) {
			commands[strings.ToLower(match[1])] = true
		}
	}
	if len(binaries) == 0 || len(commands) == 0 {
		return map[string]bool{}, map[string]bool{}
	}
	return binaries, commands
}

func commandLineIsNegated(line string) bool {
	return containsAny(line, "do not run", "don't run", "never run", "never use", "must not run", "should not run", "cannot run", "can't run", "not available", "unavailable", "not supported", "unsupported", "does not exist", "no test command", "no build command")
}

func commandNameIsVerification(name string) bool {
	lower := strings.ToLower(name)
	return containsAny(lower, "test", "check", "verify", "lint", "vet")
}

func hasSafetyBoundary(body string) bool {
	lower := strings.ToLower(body)
	hasApproval := strings.Contains(lower, "approval") && !containsAny(lower, "no approval required", "approval unnecessary")
	hasDestructiveBoundary := containsAny(lower, "never perform destructive", "destructive operation", "rollback", "do not delete", "do not overwrite") && !strings.Contains(lower, "no rollback available")
	return hasApproval && hasDestructiveBoundary
}

func hasPositiveContractPhrase(docs []inspectedDocument, phrases ...string) bool {
	for _, doc := range docs {
		for _, raw := range strings.Split(doc.Body, "\n") {
			line := strings.ToLower(strings.TrimSpace(raw))
			if commandLineIsNegated(line) {
				continue
			}
			if containsAny(line, phrases...) {
				return true
			}
		}
	}
	return false
}

func firstTestPath(paths []string) string {
	for _, path := range paths {
		if hasTestPath([]string{path}) {
			return path
		}
	}
	return "."
}

func firstFindingByRule(findings []Finding, ruleID string) *Finding {
	for index := range findings {
		if findings[index].RuleID == ruleID {
			return &findings[index]
		}
	}
	return nil
}

func readinessStatus(dimensions ReadinessDimensions) ReadinessStatus {
	states := []DimensionState{dimensions.Context, dimensions.Operability, dimensions.Safety, dimensions.Verification}
	status := ReadinessReady
	for _, state := range states {
		if state == DimensionMissing || state == DimensionConflicting {
			return ReadinessNotReady
		}
		if state == DimensionPartial {
			status = ReadinessPartial
		}
	}
	return status
}

func dimensionFinding(ruleID string, category string, state DimensionState, message string, action string) []Finding {
	if state == DimensionReady {
		return nil
	}
	severity := "medium"
	if state == DimensionMissing || state == DimensionConflicting {
		severity = "high"
	}
	evidence := fmt.Sprintf("repository-level dimension state: %s", state)
	return []Finding{newFinding(ruleID, category, severity, ConfidenceHigh, ".", 0, evidence, message, action)}
}

type instructionOccurrence struct {
	path string
	line int
	raw  string
}

func detectDuplicateInstructions(docs []inspectedDocument) []Finding {
	seen := map[string]instructionOccurrence{}
	var findings []Finding
	for _, doc := range docs {
		for index, raw := range strings.Split(doc.Body, "\n") {
			normalized := normalizeInstruction(raw)
			if !normativeInstruction(normalized) {
				continue
			}
			if previous, ok := seen[normalized]; ok && previous.path != doc.Path {
				digest := sha256.Sum256([]byte(normalized))
				evidence := fmt.Sprintf("same normalized instruction at %s:%d and %s:%d (content hash %s)", previous.path, previous.line, doc.Path, index+1, hex.EncodeToString(digest[:4]))
				findings = append(findings, newFinding("PA-001", "duplicate_instruction", "low", ConfidenceHigh, doc.Path, index+1, evidence, "The same instruction appears on two prompt surfaces.", "Keep the instruction in the authoritative contract and reference it elsewhere."))
				continue
			}
			if _, ok := seen[normalized]; !ok {
				seen[normalized] = instructionOccurrence{path: doc.Path, line: index + 1, raw: raw}
			}
		}
	}
	return findings
}

func detectStructuredConflicts(docs []inspectedDocument) []Finding {
	type ruleOccurrence struct {
		path string
		line int
		rule string
	}
	seen := map[string]ruleOccurrence{}
	var findings []Finding
	for _, doc := range docs {
		lines := strings.Split(doc.Body, "\n")
		currentID := ""
		idLine := 0
		for index, raw := range lines {
			trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "-"))
			if strings.HasPrefix(trimmed, "id:") {
				currentID = trimYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "id:")))
				idLine = index + 1
				continue
			}
			if currentID == "" || !strings.HasPrefix(trimmed, "rule:") {
				continue
			}
			rule := normalizeInstruction(trimYAMLScalar(strings.TrimSpace(strings.TrimPrefix(trimmed, "rule:"))))
			if rule == "" {
				continue
			}
			authority := filepath.ToSlash(filepath.Dir(doc.Path)) + "|" + currentID
			if previous, ok := seen[authority]; ok && previous.rule != rule {
				evidence := fmt.Sprintf("rule %s differs at %s:%d and %s:%d", currentID, previous.path, previous.line, doc.Path, idLine)
				findings = append(findings, newFinding("PA-002", "prompt_conflict", "high", ConfidenceHigh, doc.Path, idLine, evidence, "Two same-authority structured rules use the same ID with incompatible values.", "Choose one authoritative requirement for the rule ID."))
			} else if !ok {
				seen[authority] = ruleOccurrence{path: doc.Path, line: idLine, rule: rule}
			}
		}
	}
	return findings
}

func detectVendorCoupling(docs []inspectedDocument) []Finding {
	var findings []Finding
	operational := regexp.MustCompile(`(?i)\b(must|shall|required to|only)\s+(use|run with|select)\s+(claude|codex|opencode|gpt[- ]?[0-9a-z.]+|model\s+[a-z0-9._-]+)\b`)
	for _, doc := range docs {
		lower := strings.ToLower(doc.Body)
		if !strings.Contains(lower, "vendor-neutral") && !strings.Contains(lower, "provider-agnostic") {
			continue
		}
		for index, line := range strings.Split(doc.Body, "\n") {
			match := operational.FindString(line)
			if match == "" {
				continue
			}
			findings = append(findings, newFinding("PA-007", "vendor_coupling", "medium", ConfidenceHigh, doc.Path, index+1, excerpt(match), "A vendor-neutral contract requires a specific provider or model.", "Express the capability requirement without a provider or model dependency."))
		}
	}
	return findings
}

var markdownLinkPattern = regexp.MustCompile(`\[[^\]]+\]\(([^)]+)\)`)
var makeCommandPattern = regexp.MustCompile("`make[ \\t]+([A-Za-z0-9_.-]+)`")
var packageScriptPattern = regexp.MustCompile("`(?:npm|pnpm|yarn)[ \\t]+run[ \\t]+([A-Za-z0-9_.:-]+)`")
var taskCommandPattern = regexp.MustCompile("`(?:task|just)[ \\t]+([A-Za-z0-9_.:-]+)`")
var cliCommandPattern = regexp.MustCompile("`([A-Za-z0-9_.-]+)[ \\t]+([A-Za-z0-9][A-Za-z0-9_.:-]*)[^`]*`")
var delimitedPathPattern = regexp.MustCompile("`((?:rules|skills|prompts|docs|\\.agents)/[^`[:space:]]+)`")

func detectStaleReferences(root string, paths []string, docs []inspectedDocument) []Finding {
	makeTargets := discoverMakeTargets(root)
	packageScripts := discoverPackageScripts(root)
	taskTargets := discoverTaskTargets(root)
	cliBinaries, cliCommands := discoverCLICommands(root, paths)
	var findings []Finding
	for _, doc := range docs {
		if historicalReferenceDocument(doc.Path) {
			continue
		}
		lines := strings.Split(doc.Body, "\n")
		headings := map[int]string{}
		for index, line := range lines {
			if level, heading, ok := markdownHeading(line); ok {
				for current := range headings {
					if current >= level {
						delete(headings, current)
					}
				}
				headings[level] = heading
			}
			if historicalReferenceContext(headings, line) {
				continue
			}
			var markdownLinks [][]string
			if strings.EqualFold(filepath.Ext(doc.Path), ".md") {
				markdownLinks = markdownLinkPattern.FindAllStringSubmatch(line, -1)
			}
			for _, match := range markdownLinks {
				target := strings.TrimSpace(strings.Split(match[1], "#")[0])
				if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") || strings.HasPrefix(target, "#") {
					continue
				}
				candidate := filepath.Clean(filepath.Join(filepath.Dir(doc.Path), filepath.FromSlash(target)))
				if strings.HasPrefix(candidate, ".."+string(filepath.Separator)) || filepath.IsAbs(candidate) {
					continue
				}
				if _, err := os.Lstat(filepath.Join(root, candidate)); err != nil && os.IsNotExist(err) {
					evidence := fmt.Sprintf("referenced path is absent: %s", filepath.ToSlash(candidate))
					findings = append(findings, newFinding("PA-008", "stale_reference", "medium", ConfidenceHigh, doc.Path, index+1, evidence, "A repository-relative Markdown link points to an absent path.", "Update or remove the stale path reference."))
				}
			}
			for _, match := range makeCommandPattern.FindAllStringSubmatch(line, -1) {
				if !makeTargets[match[1]] {
					evidence := "documented Make target is absent: " + match[1]
					findings = append(findings, newFinding("PA-008", "stale_command", "medium", ConfidenceHigh, doc.Path, index+1, evidence, "A documented Make command has no matching target.", "Correct the command or add the intended Make target."))
				}
			}
			for _, match := range packageScriptPattern.FindAllStringSubmatch(line, -1) {
				if !packageScripts[strings.ToLower(match[1])] {
					evidence := "documented package script is absent: " + match[1]
					findings = append(findings, newFinding("PA-008", "stale_command", "medium", ConfidenceHigh, doc.Path, index+1, evidence, "A documented package-manager command has no matching script.", "Correct the command or add the intended package script."))
				}
			}
			for _, match := range taskCommandPattern.FindAllStringSubmatch(line, -1) {
				if !taskTargets[strings.ToLower(match[1])] {
					evidence := "documented task-runner target is absent: " + match[1]
					findings = append(findings, newFinding("PA-008", "stale_command", "medium", ConfidenceHigh, doc.Path, index+1, evidence, "A documented task-runner command has no matching target.", "Correct the command or add the intended task target."))
				}
			}
			for _, match := range cliCommandPattern.FindAllStringSubmatch(line, -1) {
				binary := strings.ToLower(match[1])
				command := strings.ToLower(match[2])
				if !cliBinaries[binary] || cliCommands[command] {
					continue
				}
				evidence := "documented CLI command is absent: " + binary + " " + command
				findings = append(findings, newFinding("PA-008", "stale_command", "medium", ConfidenceHigh, doc.Path, index+1, evidence, "A documented CLI subcommand has no matching static command declaration.", "Correct the command or add the intended CLI subcommand."))
			}
			for _, match := range delimitedPathPattern.FindAllStringSubmatch(line, -1) {
				if nonLiteralContractPath(match[1]) || intentionalAbsentPathContext(lines, index) {
					continue
				}
				candidate := filepath.Clean(filepath.FromSlash(match[1]))
				if filepath.IsAbs(candidate) || candidate == ".." || strings.HasPrefix(candidate, ".."+string(filepath.Separator)) {
					continue
				}
				if _, err := os.Lstat(filepath.Join(root, candidate)); err != nil && os.IsNotExist(err) {
					evidence := "referenced managed or repository path is absent: " + filepath.ToSlash(candidate)
					findings = append(findings, newFinding("PA-008", "stale_managed_reference", "medium", ConfidenceHigh, doc.Path, index+1, evidence, "A delimited repository contract path is absent.", "Update or remove the stale managed-path reference."))
				}
			}
		}
	}
	return findings
}

func historicalReferenceDocument(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return base == "changelog.md" || strings.Contains(base, "release-notes") || strings.Contains(filepath.ToSlash(strings.ToLower(path)), "/archive/")
}

func markdownHeading(line string) (int, string, bool) {
	trimmed := strings.TrimSpace(line)
	level := 0
	for level < len(trimmed) && trimmed[level] == '#' {
		level++
	}
	if level == 0 || level > 6 || level >= len(trimmed) || trimmed[level] != ' ' {
		return 0, "", false
	}
	return level, strings.ToLower(strings.TrimSpace(trimmed[level+1:])), true
}

func historicalReferenceContext(headings map[int]string, line string) bool {
	for _, heading := range headings {
		if containsAny(heading,
			"non-goal", "commands and options to remove", "commands explicitly not planned",
			"previous proposals", "compatibility and migration", "breaking changes",
			"applied transition", "implementation plan", "rollback", "release decisions",
			"change log", "source-derived design principles", "future vocabulary",
			"source disposition", "rejected feature") {
			return true
		}
	}
	lower := strings.ToLower(line)
	return containsAny(lower,
		"does not add", "does not generate", "do not add", "do not generate",
		"not planned", "removal of ", "replacement of ", "is removed", "are removed",
		"rejected", "outside core", "outside the core")
}

func intentionalAbsentPathContext(lines []string, index int) bool {
	start := index - 1
	if start < 0 {
		start = 0
	}
	end := index + 2
	if end > len(lines) {
		end = len(lines)
	}
	lower := strings.ToLower(strings.Join(lines[start:end], " "))
	return containsAny(lower,
		"legacy", "migration", "force cleanup", "force-cleanup", "cleanup target",
		"remove", "delete", "prune", "except", "must be absent", "is absent", "are absent")
}

func nonLiteralContractPath(path string) bool {
	return strings.ContainsAny(path, "*?[") || strings.Contains(path, "<") || strings.Contains(path, ">") || strings.Contains(path, "...")
}

func discoverMakeTargets(root string) map[string]bool {
	targets := map[string]bool{}
	read, err := safeReadRepositoryFile(root, "Makefile", maxInspectedFileBytes)
	if err != nil || read.Truncated {
		return targets
	}
	pattern := regexp.MustCompile(`^([A-Za-z0-9_.-]+)\s*:`)
	for _, line := range strings.Split(string(read.Body), "\n") {
		if match := pattern.FindStringSubmatch(line); len(match) == 2 {
			targets[match[1]] = true
		}
	}
	return targets
}

func newFinding(ruleID string, category string, severity string, confidence Confidence, path string, line int, evidence string, message string, action string) Finding {
	evidence = excerpt(evidence)
	location := fmt.Sprintf("%s:%d", path, line)
	sum := sha256.Sum256([]byte(ruleID + "\x00" + path + "\x00" + location + "\x00" + evidence))
	return Finding{
		FindingID:       ruleID + "-" + hex.EncodeToString(sum[:4]),
		RuleID:          ruleID,
		Title:           findingTitle(ruleID, category),
		Category:        category,
		Severity:        severity,
		Confidence:      confidence,
		Path:            path,
		Line:            line,
		Evidence:        evidence,
		Message:         message,
		SuggestedAction: action,
	}
}

func findingTitle(ruleID string, category string) string {
	titles := map[string]string{
		"duplicate_instruction":         "Duplicate instruction",
		"prompt_conflict":               "Conflicting prompt rule",
		"vendor_coupling":               "Vendor-coupled instruction",
		"stale_reference":               "Stale repository reference",
		"stale_command":                 "Stale documented command",
		"stale_managed_reference":       "Stale managed-path reference",
		"missing_context_contract":      "Incomplete context contract",
		"missing_command_contract":      "Incomplete command contract",
		"missing_safety_contract":       "Incomplete safety contract",
		"missing_verification_contract": "Incomplete verification contract",
	}
	if title := titles[category]; title != "" {
		return title
	}
	return ruleID + " finding"
}

func excerpt(value string) string {
	value = strings.Join(strings.Fields(value), " ")
	secretPattern := regexp.MustCompile(`(?i)(token|password|passwd|secret|api[_-]?key|client[_-]?secret|authorization)\s*[:=]\s*(?:"[^"]*"|'[^']*'|\S+)`)
	value = secretPattern.ReplaceAllString(value, "$1=[REDACTED]")
	bearerPattern := regexp.MustCompile(`(?i)\b(?:bearer|basic)\s+[A-Za-z0-9._~+/=-]+`)
	value = bearerPattern.ReplaceAllString(value, "authorization [REDACTED]")
	credentialPattern := regexp.MustCompile(`\b(?:sk-[A-Za-z0-9_-]+|gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|xox[baprs]-[A-Za-z0-9-]+|AKIA[0-9A-Z]{16})\b`)
	value = credentialPattern.ReplaceAllString(value, "[REDACTED]")
	jwtPattern := regexp.MustCompile(`\beyJ[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\.[A-Za-z0-9_-]+\b`)
	value = jwtPattern.ReplaceAllString(value, "[REDACTED]")
	urlCredentials := regexp.MustCompile(`(?i)(https?://)[^/@[:space:]]+:[^/@[:space:]]+@`)
	value = urlCredentials.ReplaceAllString(value, "${1}[REDACTED]@")
	unixHome := regexp.MustCompile(`(?:/Users|/home)/[^/[:space:]]+`)
	value = unixHome.ReplaceAllString(value, "/[HOME]")
	windowsHome := regexp.MustCompile(`(?i)[A-Z]:\\Users\\[^\\[:space:]]+`)
	value = windowsHome.ReplaceAllString(value, `[HOME]`)
	emailPattern := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}\b`)
	value = emailPattern.ReplaceAllString(value, "[REDACTED_EMAIL]")
	runes := []rune(value)
	if len(runes) > maxEvidenceRunes {
		return string(runes[:maxEvidenceRunes-1]) + "…"
	}
	return value
}

func normalizeInstruction(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimLeftFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || r == '-' || r == '*' || r == '+' || r == '>' || unicode.IsDigit(r) || r == '.' || r == ')'
	})
	return strings.ToLower(strings.Join(strings.Fields(value), " "))
}

func normativeInstruction(value string) bool {
	if len([]rune(value)) < 24 {
		return false
	}
	return containsAny(value, "must ", "must not", "never ", "do not ", "required", "should ", "prefer ", "only ")
}

func trimYAMLScalar(value string) string {
	return strings.Trim(strings.TrimSpace(value), `"'`)
}

func sensitiveAnalysisPath(rel string) bool {
	lower := strings.ToLower(rel)
	base := strings.ToLower(filepath.Base(rel))
	if base == ".env" || strings.HasPrefix(base, ".env.") || strings.Contains(base, "credential") || strings.Contains(base, "token") || strings.Contains(base, "private") {
		return true
	}
	if strings.Contains(base, ".local.") && base != "claude.local.md" {
		return true
	}
	switch strings.ToLower(filepath.Ext(lower)) {
	case ".pem", ".key", ".p12", ".pfx", ".crt", ".cer":
		return true
	default:
		return false
	}
}

func safeSkipReason(err error) string {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "utf-8") || strings.Contains(message, "binary") {
		return "binary or invalid UTF-8"
	}
	if strings.Contains(message, "escape") || strings.Contains(message, "symlink") {
		return "unsafe path not followed"
	}
	return "unreadable file"
}

func joinDocumentBodies(docs []inspectedDocument) string {
	var builder strings.Builder
	for _, doc := range docs {
		builder.WriteString(doc.Body)
		builder.WriteByte('\n')
	}
	return builder.String()
}

func hasTestingTechnology(technologies []DetectedTechnology) bool {
	for _, technology := range technologies {
		switch technology.ID {
		case "vitest", "jest", "playwright", "cypress", "go-test", "bats":
			return true
		}
	}
	return false
}

func hasTestPath(paths []string) bool {
	for _, path := range paths {
		lower := strings.ToLower(path)
		if strings.Contains(lower, "/tests/") || strings.HasPrefix(lower, "tests/") || strings.HasSuffix(lower, "_test.go") || strings.HasSuffix(lower, ".bats") || strings.HasSuffix(lower, ".test.ts") || strings.HasSuffix(lower, ".spec.ts") {
			return true
		}
	}
	return false
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].RuleID != findings[j].RuleID {
			return findings[i].RuleID < findings[j].RuleID
		}
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		if findings[i].Line != findings[j].Line {
			return findings[i].Line < findings[j].Line
		}
		return findings[i].FindingID < findings[j].FindingID
	})
}

func sortAuditNotes(notes []AuditNote) {
	sort.Slice(notes, func(i, j int) bool {
		if notes[i].Path != notes[j].Path {
			return notes[i].Path < notes[j].Path
		}
		return notes[i].Reason < notes[j].Reason
	})
}

func capAuditFindings(audit *DeepAudit) {
	total := len(audit.DriftFindings) + len(audit.PromptFindings)
	if total <= maxAuditFindings {
		return
	}
	highPrompt := make([]Finding, 0)
	otherPrompt := make([]Finding, 0, len(audit.PromptFindings))
	for _, finding := range audit.PromptFindings {
		if finding.Severity == "high" {
			highPrompt = append(highPrompt, finding)
		} else {
			otherPrompt = append(otherPrompt, finding)
		}
	}
	selectedPrompt := append([]Finding{}, highPrompt...)
	if len(selectedPrompt) > maxAuditFindings {
		selectedPrompt = selectedPrompt[:maxAuditFindings]
	}
	remaining := maxAuditFindings - len(selectedPrompt)
	selectedDrift := audit.DriftFindings
	if len(selectedDrift) > remaining {
		selectedDrift = selectedDrift[:remaining]
	}
	remaining -= len(selectedDrift)
	if len(otherPrompt) > remaining {
		otherPrompt = otherPrompt[:remaining]
	}
	selectedPrompt = append(selectedPrompt, otherPrompt...)
	audit.DriftFindings = selectedDrift
	audit.PromptFindings = selectedPrompt
	sortFindings(audit.DriftFindings)
	sortFindings(audit.PromptFindings)
	audit.Truncations = append(audit.Truncations, AuditNote{Reason: fmt.Sprintf("finding limit reached: %d", maxAuditFindings)})
}
