package app

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/bootstrap"
	"github.com/leanbusqts/agent47/internal/resolve"
	"github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/templates"
)

func (r *Root) runAddAgent(ctx context.Context, cfg runtime.Config, args []string) int {
	var opts bootstrap.Options
	var resolveOpts resolve.Options
	var previewOnly bool

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--force":
			opts.Force = true
		case "--only-skills":
			opts.OnlySkills = true
		case "--preview", "--dry-run":
			previewOnly = true
		case "--yes":
			opts.Yes = true
		case "--bundle":
			if i+1 >= len(args) {
				r.out.Printf("Usage: add-agent [--force] [--only-skills] [--preview|--dry-run] [--yes] [--bundle <name>] [--exclude-bundle <name>]\n")
				return 1
			}
			i++
			resolveOpts.ExplicitBundles = append(resolveOpts.ExplicitBundles, strings.ToLower(args[i]))
		case "--exclude-bundle":
			if i+1 >= len(args) {
				r.out.Printf("Usage: add-agent [--force] [--only-skills] [--preview|--dry-run] [--yes] [--bundle <name>] [--exclude-bundle <name>]\n")
				return 1
			}
			i++
			resolveOpts.ExcludeBundles = append(resolveOpts.ExcludeBundles, strings.ToLower(args[i]))
		default:
			r.out.Printf("Usage: add-agent [--force] [--only-skills] [--preview|--dry-run] [--yes] [--bundle <name>] [--exclude-bundle <name>]\n")
			return 1
		}
	}

	workDir, err := os.Getwd()
	if err != nil {
		r.out.Err("Failed to read working directory: %v", err)
		return 1
	}

	if !opts.OnlySkills {
		result, installSet, err := analyzeAndResolve(workDir, resolveOpts)
		if err != nil {
			r.out.Err("%s", normalizeBootstrapError(err))
			return 1
		}
		printAddAgentPlan(r, result, installSet, workDir, opts.Force)
		opts.InstallSet = installSet

		if previewOnly {
			return 0
		}
		if shouldConfirmWrite(opts.Yes) && !confirmWrite(r) {
			r.out.Info("Aborted before writing.")
			return 0
		}
	}

	service, err := bootstrap.New(cfg, r.out)
	if err != nil {
		r.out.Err("Failed to initialize bootstrap service: %v", err)
		return 1
	}

	if err := service.Run(ctx, opts); err != nil {
		r.out.Err("%s", normalizeBootstrapError(err))
		return 1
	}

	return 0
}

func normalizeBootstrapError(err error) string {
	if err == nil {
		return ""
	}
	var conflictErr templates.AssemblyConflictError
	if errors.As(err, &conflictErr) {
		return conflictErr.Error() + "\nUse a supported bundle composition or remove one of the conflicting bundles."
	}
	var missingBundleErr templates.MissingBundleManifestError
	if errors.As(err, &missingBundleErr) {
		return missingBundleErr.Error() + "\nThe selected bundle set is incomplete. Restore the bundle manifest or choose a different bundle selection."
	}
	var invalidBundleErr templates.InvalidBundleManifestError
	if errors.As(err, &invalidBundleErr) {
		return invalidBundleErr.Error() + "\nRestore the bundle manifest contents or exclude that bundle from the install set."
	}
	var missingTemplateErr templates.MissingTemplateError
	if errors.As(err, &missingTemplateErr) {
		if missingTemplateErr.Path == "skills" {
			return "Skills templates are missing.\nRestore the managed skills templates or use a valid agent47 runtime install."
		}
		return fmt.Sprintf("Template not found: %s\nRestore the missing template asset or choose a bundle set that does not require it.", missingTemplateErr.Path)
	}
	return fmt.Sprintf("%v", err)
}

func printAddAgentPlan(r *Root, result analyze.AnalysisResult, set resolve.InstallSet, workDir string, force bool) {
	r.out.Info("Analyzing repository...")
	if result.LowSignal {
		r.out.Info("No strong project signals found.")
	}
	if result.UnresolvedConflict {
		r.out.Warn("Multiple project types detected with no supported automatic composition:")
		for _, projectType := range result.ConflictProjectTypes {
			r.out.Printf("       - %s\n", projectType)
		}
		r.out.Info("Installing base bundle only.")
		r.out.Info("No project-specific bundles were installed because the project type is ambiguous.")
		r.out.Info("Run: afs analyze --verbose")
		r.out.Info("Or install explicit bundles with:")
		for _, projectType := range result.ConflictProjectTypes {
			r.out.Printf("       afs add-agent --bundle %s\n", projectType)
		}
	}
	if len(result.ManagedState.Notes) > 0 {
		for _, note := range result.ManagedState.Notes {
			r.out.Info("%s", note)
		}
	}

	r.out.Printf("Preview\n")
	r.out.Printf("  types: %s\n", summarizeProjectTypes(result.ProjectTypes))
	r.out.Printf("  bundles: %s\n", strings.Join(set.Bundles, ", "))

	plan := resolve.BuildActionPlan(workDir, set, force)
	if len(plan.Create) > 0 {
		r.out.Printf("  create:\n")
		for _, item := range plan.Create {
			r.out.Printf("    %s\n", item)
		}
	}
	if len(plan.Update) > 0 {
		r.out.Printf("  update:\n")
		for _, item := range plan.Update {
			r.out.Printf("    %s\n", item)
		}
	}
	if len(plan.Keep) > 0 {
		r.out.Printf("  keep:\n")
		for _, item := range plan.Keep {
			r.out.Printf("    %s\n", item)
		}
	}
	if len(plan.Remove) > 0 {
		r.out.Printf("  remove on --force:\n")
		for _, item := range plan.Remove {
			r.out.Printf("    %s\n", item)
		}
	}
}

func shouldConfirmWrite(yes bool) bool {
	if yes || os.Getenv("CI") != "" {
		return false
	}

	stdinInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	stdoutInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	return stdinInfo.Mode()&os.ModeCharDevice != 0 && stdoutInfo.Mode()&os.ModeCharDevice != 0
}

func confirmWrite(r *Root) bool {
	r.out.Printf("Proceed with scaffold write? [y/N]: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return false
	}

	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
