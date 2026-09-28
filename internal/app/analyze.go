package app

import (
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/resolve"
	"github.com/leanbusqts/agent47/internal/runtime"
)

type analyzeOptions struct {
	JSON     bool
	Verbose  bool
	Evidence bool
	Deep     bool
}

func (r *Root) runAnalyze(ctx context.Context, _ runtime.Config, args []string) int {
	opts, ok := parseAnalyzeOptions(args, r)
	if !ok {
		return 2
	}

	workDir, err := os.Getwd()
	if err != nil {
		r.out.Err("Failed to read working directory: %v", err)
		return 1
	}
	if err := ctx.Err(); err != nil {
		r.out.Err("%v", err)
		return 1
	}

	result, set, err := analyzeAndResolveWithAnalyzeOptions(ctx, workDir, resolve.Options{}, analyze.AnalyzeOptions{Deep: opts.Deep})
	if err != nil {
		r.out.Err("%v", err)
		return 1
	}

	if opts.JSON {
		payload := struct {
			analyze.AnalysisResult
			InstallPlan resolve.InstallSet `json:"install_plan"`
		}{
			AnalysisResult: result,
			InstallPlan:    set,
		}
		data, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			r.out.Err("Failed to encode JSON: %v", err)
			return 1
		}
		r.out.Printf("%s\n", string(data))
		return 0
	}

	printAnalysisText(r, result, set, opts)
	return 0
}

func parseAnalyzeOptions(args []string, r *Root) (analyzeOptions, bool) {
	var opts analyzeOptions
	for _, arg := range args {
		switch arg {
		case "--json":
			opts.JSON = true
		case "--verbose":
			opts.Verbose = true
		case "--evidence":
			opts.Evidence = true
		case "--deep":
			opts.Deep = true
		default:
			r.out.Diagnosticf("Usage: afs analyze [--json] [--verbose] [--evidence] [--deep]\n")
			return analyzeOptions{}, false
		}
	}
	return opts, true
}

func analyzeAndResolve(workDir string, opts resolve.Options) (analyze.AnalysisResult, resolve.InstallSet, error) {
	return analyzeAndResolveWithAnalyzeOptions(context.Background(), workDir, opts, analyze.AnalyzeOptions{})
}

func analyzeAndResolveWithAnalyzeOptions(ctx context.Context, workDir string, opts resolve.Options, analyzeOpts analyze.AnalyzeOptions) (analyze.AnalysisResult, resolve.InstallSet, error) {
	result, err := (analyze.Service{}).AnalyzeContext(ctx, workDir, analyzeOpts)
	if err != nil {
		return analyze.AnalysisResult{}, resolve.InstallSet{}, err
	}
	set, err := resolve.Resolve(result, opts)
	if err != nil {
		return analyze.AnalysisResult{}, resolve.InstallSet{}, err
	}
	return result, set, nil
}

func printAnalysisText(r *Root, result analyze.AnalysisResult, set resolve.InstallSet, opts analyzeOptions) {
	r.out.Printf("Project summary\n")
	r.out.Printf("  type: %s\n", summarizeProjectTypes(result.ProjectTypes))
	r.out.Printf("  confidence: %s\n", result.Confidence)
	r.out.Printf("  repo shape: %s\n", result.RepoShape)
	r.out.Printf("\n")
	r.out.Printf("Install set\n")
	r.out.Printf("  bundles: %s\n", strings.Join(set.Bundles, ", "))
	if len(set.DecisionNotes) > 0 {
		r.out.Printf("  note: %s\n", set.DecisionNotes[0])
	}
	if result.UnresolvedConflict && opts.Verbose {
		r.out.Printf("  unresolved conflict: %s\n", strings.Join(result.ConflictProjectTypes, ", "))
	}
	warnings := visibleAnalysisWarnings(result.Warnings, opts)
	if len(warnings) > 0 {
		r.out.Printf("\nWarnings\n")
		for _, warning := range warnings {
			r.out.Printf("  %s\n", warning)
		}
	}

	if !opts.Verbose && !opts.Evidence && !opts.Deep {
		return
	}

	if opts.Verbose || opts.Evidence {
		r.out.Printf("\nDetected technologies\n")
		for _, technology := range result.Technologies {
			r.out.Printf("  %s (%s)\n", technology.ID, technology.Confidence)
		}

		if testing := testingTechnologies(result.Technologies); len(testing) > 0 {
			r.out.Printf("\nTesting stacks\n")
			for _, technology := range testing {
				r.out.Printf("  %s (%s)\n", technology.ID, technology.Confidence)
			}
		}

		r.out.Printf("\nRules\n")
		for _, rule := range set.Rules {
			r.out.Printf("  %s\n", rule)
		}

		r.out.Printf("\nEvidence\n")
		for _, item := range result.Evidence {
			r.out.Printf("  %s: %s", item.Kind, item.Detail)
			if len(item.SourcePaths) > 0 {
				r.out.Printf(" (%s)", strings.Join(item.SourcePaths, ", "))
			}
			r.out.Printf("\n")
		}

		printAgentPolicyText(r, result.AgentPolicy, opts.Evidence)
	}

	if len(result.ManagedState.Notes) > 0 {
		r.out.Printf("\nManaged state\n")
		for _, note := range result.ManagedState.Notes {
			r.out.Printf("  %s\n", note)
		}
	}

	if result.UnresolvedConflict {
		r.out.Printf("\nConflict\n")
		r.out.Printf("  unsupported automatic composition: %s\n", strings.Join(result.ConflictProjectTypes, ", "))
		r.out.Printf("  fallback: base bundle only\n")
	}

	if opts.Deep && result.AgentReadiness != nil && result.DeepAudit != nil {
		printDeepAnalysisText(r, *result.AgentReadiness, *result.DeepAudit, opts)
	}
}

func visibleAnalysisWarnings(warnings []string, opts analyzeOptions) []string {
	if opts.Verbose || opts.Evidence || opts.Deep {
		return warnings
	}
	visible := []string{}
	for _, warning := range warnings {
		if strings.Contains(warning, "limit reached") {
			visible = append(visible, warning)
		}
	}
	return visible
}

func printAgentPolicyText(r *Root, policy analyze.AgentPolicy, showEvidence bool) {
	r.out.Printf("\nAgent policy\n")
	r.out.Printf("  AGENTS.md files: %d\n", len(policy.AgentsFiles))
	r.out.Printf("  claude: %s\n", policy.Tools.Claude.State)
	r.out.Printf("  codex: %s\n", policy.Tools.Codex.State)
	r.out.Printf("  opencode: %s\n", policy.Tools.OpenCode.State)
	for _, warning := range policy.Warnings {
		r.out.Printf("  policy warning: %s\n", warning)
	}
	printToolPolicyWarnings(r, "claude", policy.Tools.Claude)
	printToolPolicyWarnings(r, "codex", policy.Tools.Codex)
	printToolPolicyWarnings(r, "opencode", policy.Tools.OpenCode)
	if !showEvidence {
		return
	}
	for _, file := range policy.AgentsFiles {
		r.out.Printf("  policy: %s (scope %s)\n", file.Path, file.Scope)
	}
	printToolPolicyEvidence(r, "claude", policy.Tools.Claude)
	printToolPolicyEvidence(r, "codex", policy.Tools.Codex)
	printToolPolicyEvidence(r, "opencode", policy.Tools.OpenCode)
}

func printToolPolicyEvidence(r *Root, name string, policy analyze.ToolPolicy) {
	for _, path := range policy.Evidence {
		r.out.Printf("  %s evidence: %s\n", name, path)
	}
}

func printToolPolicyWarnings(r *Root, name string, policy analyze.ToolPolicy) {
	for _, warning := range policy.Warnings {
		r.out.Printf("  %s warning: %s\n", name, warning)
	}
}

func printDeepAnalysisText(r *Root, readiness analyze.AgentReadiness, audit analyze.DeepAudit, opts analyzeOptions) {
	r.out.Printf("\nAgent readiness\n")
	r.out.Printf("  status: %s\n", readiness.Status)
	r.out.Printf("  context: %s\n", readiness.Dimensions.Context)
	r.out.Printf("  operability: %s\n", readiness.Dimensions.Operability)
	r.out.Printf("  safety: %s\n", readiness.Dimensions.Safety)
	r.out.Printf("  verification: %s\n", readiness.Dimensions.Verification)
	r.out.Printf("\nDeep audit\n")
	r.out.Printf("  prompt surfaces: %d\n", audit.PromptSurfaces)
	r.out.Printf("  drift findings: %d\n", len(audit.DriftFindings))
	r.out.Printf("  prompt findings: %d\n", len(audit.PromptFindings))
	if len(audit.Truncations) > 0 {
		r.out.Printf("  truncations: %d\n", len(audit.Truncations))
	}
	if len(audit.Skipped) > 0 {
		r.out.Printf("  skipped: %d\n", len(audit.Skipped))
	}
	if !opts.Verbose && !opts.Evidence {
		r.out.Printf("  note: use --verbose, --evidence, or --json for details\n")
		return
	}
	for _, finding := range append(append([]analyze.Finding{}, readiness.Findings...), append(audit.DriftFindings, audit.PromptFindings...)...) {
		r.out.Printf("  %s [%s] %s\n", finding.FindingID, finding.Severity, finding.Message)
		if opts.Verbose {
			r.out.Printf("    action: %s\n", finding.SuggestedAction)
		}
		if opts.Evidence {
			if finding.Path != "" {
				r.out.Printf("    location: %s", finding.Path)
				if finding.Line > 0 {
					r.out.Printf(":%d", finding.Line)
				}
				r.out.Printf("\n")
			}
			r.out.Printf("    evidence: %s\n", finding.Evidence)
		}
	}
	for _, note := range audit.Truncations {
		r.out.Printf("  truncated: %s", note.Reason)
		if note.Path != "" {
			r.out.Printf(" (%s)", note.Path)
		}
		r.out.Printf("\n")
	}
	for _, note := range audit.Skipped {
		r.out.Printf("  skipped: %s", note.Reason)
		if note.Path != "" {
			r.out.Printf(" (%s)", note.Path)
		}
		r.out.Printf("\n")
	}
}

func summarizeProjectTypes(projectTypes []analyze.DetectedProjectType) string {
	if len(projectTypes) == 0 {
		return "unknown"
	}
	ids := make([]string, 0, len(projectTypes))
	for _, projectType := range projectTypes {
		ids = append(ids, projectType.ID)
	}
	return strings.Join(ids, ", ")
}

func testingTechnologies(technologies []analyze.DetectedTechnology) []analyze.DetectedTechnology {
	var testing []analyze.DetectedTechnology
	for _, technology := range technologies {
		switch technology.ID {
		case "vitest", "jest", "playwright", "cypress", "go-test", "bats":
			testing = append(testing, technology)
		}
	}
	return testing
}
