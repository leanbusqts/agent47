package app

import (
	"context"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

type Root struct {
	out cli.Output
}

func NewRoot(out cli.Output) *Root {
	return &Root{out: out}
}

func (r *Root) Run(ctx context.Context, cfg runtime.Config, args []string) int {
	if len(args) == 0 {
		r.printHelp(cfg.Version)
		return 0
	}
	if args[0] == "help" {
		if len(args) != 1 {
			r.out.Diagnosticf("Usage: afs help\n")
			return 2
		}
		r.printHelp(cfg.Version)
		return 0
	}

	switch args[0] {
	case internalInstallCommand:
		return r.runInstallInternal(ctx, cfg, args[1:])
	case "version":
		if len(args) != 1 {
			r.out.Diagnosticf("Usage: afs version\n")
			return 2
		}
		r.out.Printf("%s\n", cfg.Version)
		return 0
	case "analyze":
		return r.runAnalyze(ctx, cfg, args[1:])
	case "map":
		return r.runMap(ctx, cfg, args[1:])
	case "init":
		return r.runInit(ctx, cfg, args[1:])
	case "doctor":
		return r.runDoctor(ctx, cfg, args[1:])
	case "uninstall":
		return r.runUninstall(ctx, cfg, args[1:])
	default:
		r.out.Diagnosticf("Unknown command: %s\n", args[0])
		r.printHelpWith(r.out.Diagnosticf, cfg.Version)
		return 2
	}
}

func (r *Root) printHelp(version string) {
	r.printHelpWith(r.out.Printf, version)
}

func (r *Root) printHelpWith(write func(string, ...any), version string) {
	write("agent47 Agent CLI (command: afs, Agent Forty-Seven)\n")
	write("Version: %s\n", version)
	write("\n")
	write("Core commands:\n")
	write("  afs help\n")
	write("  afs version\n")
	write("  afs uninstall\n")
	write("  afs doctor [--json] [--check-update|--check-update-force|--fail-on-warn]\n")
	write("\n")
	write("Project commands:\n")
	write("  afs analyze [--json] [--verbose] [--evidence] [--deep]\n")
	write("  afs map [--force]\n")
	write("  afs init [--bundle <name> ...] [--exclude-bundle <name> ...]\n")
	write("           [--force] [--preview]\n")
	write("           --force replaces rules/ and removes legacy skills, prompts, and task specs\n")
}
