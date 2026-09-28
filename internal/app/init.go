package app

import (
	"context"
	"os"
	"sort"
	"strings"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/contextmap"
	"github.com/leanbusqts/agent47/internal/initrepo"
	"github.com/leanbusqts/agent47/internal/resolve"
	"github.com/leanbusqts/agent47/internal/runtime"
)

const initUsage = "Usage: afs init [--bundle <name>] [--exclude-bundle <name>] [--force] [--preview]"

type initOptions struct {
	Force   bool
	Preview bool
}

func (r *Root) runInit(ctx context.Context, cfg runtime.Config, args []string) int {
	opts, resolveOpts, ok := parseInitOptions(args, r)
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
	if err := validateLocalContextPath(workDir, cfg); err != nil {
		r.out.Err("%v", err)
		return 1
	}

	r.out.Info("Analyzing repository...")
	result, err := (analyze.Service{}).AnalyzeContext(ctx, workDir, analyze.AnalyzeOptions{})
	if err != nil {
		r.out.Err("%v", err)
		return 1
	}
	installSet, err := resolve.Resolve(result, resolveOpts)
	if err != nil {
		r.out.Err("%v", err)
		r.out.Diagnosticf("%s\n", initUsage)
		return 2
	}
	plannedPolicies := []string{"AGENTS.md"}
	for _, rule := range installSet.Rules {
		plannedPolicies = append(plannedPolicies, "rules/"+rule)
	}
	mapOptions := contextmap.BuildOptions{PlannedPolicies: plannedPolicies}
	if opts.Force {
		mapOptions.ExcludedPaths = []string{"rules", "skills", "prompts", "specs/spec.yml", ".agents/specs/spec.yml"}
	}
	document, err := contextmap.Build(ctx, workDir, result, mapOptions)
	if err != nil {
		r.out.Err("Failed to build repository context: %v", err)
		return 1
	}

	service, err := initrepo.New(cfg)
	if err != nil {
		r.out.Err("Failed to initialize repository: %v", err)
		return 1
	}
	plan, err := service.PlanWithContext(workDir, installSet, opts.Force, document.Content)
	if err != nil {
		r.out.Err("%v", err)
		return 1
	}
	printInitPreview(r, result, installSet, plan)
	if opts.Preview {
		return 0
	}

	err = service.Run(ctx, initrepo.Options{
		Force:      opts.Force,
		WorkDir:    workDir,
		InstallSet: installSet,
		Context:    document.Content,
	})
	if err != nil {
		r.out.Err("%v", err)
		return 1
	}

	r.out.OK("Repository initialized.")
	return 0
}

func parseInitOptions(args []string, r *Root) (initOptions, resolve.Options, bool) {
	var opts initOptions
	var resolveOpts resolve.Options
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--force":
			opts.Force = true
		case "--preview", "--dry-run":
			opts.Preview = true
		case "--bundle":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				r.out.Diagnosticf("%s\n", initUsage)
				return initOptions{}, resolve.Options{}, false
			}
			i++
			resolveOpts.ExplicitBundles = append(resolveOpts.ExplicitBundles, strings.ToLower(args[i]))
		case "--exclude-bundle":
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
				r.out.Diagnosticf("%s\n", initUsage)
				return initOptions{}, resolve.Options{}, false
			}
			i++
			resolveOpts.ExcludeBundles = append(resolveOpts.ExcludeBundles, strings.ToLower(args[i]))
		default:
			r.out.Diagnosticf("%s\n", initUsage)
			return initOptions{}, resolve.Options{}, false
		}
	}
	for _, included := range resolveOpts.ExplicitBundles {
		for _, excluded := range resolveOpts.ExcludeBundles {
			if included == excluded {
				r.out.Diagnosticf("%s\n", initUsage)
				return initOptions{}, resolve.Options{}, false
			}
		}
	}
	return opts, resolveOpts, true
}

func printInitPreview(r *Root, result analyze.AnalysisResult, set resolve.InstallSet, plan initrepo.Plan) {
	r.out.Printf("Preview\n")
	r.out.Printf("  types: %s\n", summarizeProjectTypes(result.ProjectTypes))
	r.out.Printf("  bundles: %s\n", strings.Join(set.Bundles, ", "))
	printInitPlanGroup(r, "create", plan.Create)
	printInitPlanGroup(r, "update", plan.Update)
	printInitPlanGroup(r, "keep", plan.Keep)
	printInitPlanGroup(r, "remove", plan.Remove)
	printInitPlanGroup(r, "warnings", collectInitWarnings(result))
}

func collectInitWarnings(result analyze.AnalysisResult) []string {
	warnings := append([]string{}, result.Warnings...)
	for _, warning := range result.AgentPolicy.Warnings {
		warnings = append(warnings, "agent policy warning: "+warning)
	}
	for _, file := range result.AgentPolicy.AgentsFiles {
		if file.Path != "AGENTS.md" {
			warnings = append(warnings, "nested policy detected: "+file.Path)
		}
	}
	tools := []struct {
		name   string
		policy analyze.ToolPolicy
	}{
		{name: "claude", policy: result.AgentPolicy.Tools.Claude},
		{name: "codex", policy: result.AgentPolicy.Tools.Codex},
		{name: "opencode", policy: result.AgentPolicy.Tools.OpenCode},
	}
	for _, tool := range tools {
		if tool.policy.State != analyze.PolicyAbsent {
			detail := ""
			if len(tool.policy.Evidence) > 0 {
				detail = ": " + strings.Join(tool.policy.Evidence, ", ")
			}
			warnings = append(warnings, "vendor policy detected: "+tool.name+" ("+string(tool.policy.State)+")"+detail)
		}
		for _, warning := range tool.policy.Warnings {
			warnings = append(warnings, tool.name+" policy warning: "+warning)
		}
	}
	if result.UnresolvedConflict {
		warnings = append(warnings, "unresolved project-type conflict; using the base bundle: "+strings.Join(result.ConflictProjectTypes, ", "))
	}
	sort.Strings(warnings)
	return uniqueStrings(warnings)
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	return result
}

func printInitPlanGroup(r *Root, name string, paths []string) {
	r.out.Printf("  %s:\n", name)
	if len(paths) == 0 {
		r.out.Printf("    (none)\n")
		return
	}
	for _, path := range paths {
		r.out.Printf("    %s\n", path)
	}
}
