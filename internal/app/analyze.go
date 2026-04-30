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
}

func (r *Root) runAnalyze(ctx context.Context, _ runtime.Config, args []string) int {
	opts, ok := parseAnalyzeOptions(args, r)
	if !ok {
		return 1
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

	result, set, err := analyzeAndResolve(workDir, resolve.Options{})
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
		default:
			r.out.Printf("Usage: analyze [--json] [--verbose] [--evidence]\n")
			return analyzeOptions{}, false
		}
	}
	return opts, true
}

func analyzeAndResolve(workDir string, opts resolve.Options) (analyze.AnalysisResult, resolve.InstallSet, error) {
	result, err := (analyze.Service{}).Analyze(workDir)
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

	if !opts.Verbose && !opts.Evidence {
		return
	}

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

	r.out.Printf("\nSkills\n")
	for _, skill := range set.Skills {
		r.out.Printf("  %s\n", skill)
	}

	if opts.Evidence || opts.Verbose {
		r.out.Printf("\nEvidence\n")
		for _, item := range result.Evidence {
			r.out.Printf("  %s: %s", item.Kind, item.Detail)
			if len(item.SourcePaths) > 0 {
				r.out.Printf(" (%s)", strings.Join(item.SourcePaths, ", "))
			}
			r.out.Printf("\n")
		}
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
