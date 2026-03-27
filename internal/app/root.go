package app

import (
	"context"
	"path/filepath"
	"strings"

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
	if mapped, ok := helperCommandForExecutable(cfg.ExecutablePath, args); ok {
		args = mapped
	}

	if len(args) == 0 || args[0] == "help" {
		r.printHelp(cfg.Version)
		return 0
	}

	switch args[0] {
	case internalInstallCommand:
		return r.runInstallInternal(ctx, cfg, args[1:])
	case "version":
		r.out.Printf("%s\n", cfg.Version)
		return 0
	case "add-agent":
		return r.runAddAgent(ctx, cfg, args[1:])
	case "doctor":
		return r.runDoctor(ctx, cfg, args[1:])
	case "add-agent-prompt":
		return r.runAddAgentPrompt(ctx, cfg, args[1:])
	case "add-ss-prompt":
		return r.runAddSSPrompt(ctx, cfg, args[1:])
	case "uninstall":
		return r.runUninstall(ctx, cfg, args[1:])
	default:
		r.out.Printf("Unknown command: %s\n", args[0])
		r.printHelp(cfg.Version)
		return 1
	}
}

func (r *Root) printHelp(version string) {
	r.out.Printf("agent47 Agent CLI (command: afs, Agent Forty-Seven)\n")
	r.out.Printf("Version: %s\n", version)
	r.out.Printf("\n")
	r.out.Printf("Core commands:\n")
	r.out.Printf("  afs help\n")
	r.out.Printf("  afs uninstall\n")
	r.out.Printf("  afs doctor [--check-update|--check-update-force|--fail-on-warn]\n")
	r.out.Printf("\n")
	r.out.Printf("Project commands:\n")
	r.out.Printf("  afs add-agent                 bootstrap project scaffolding\n")
	r.out.Printf("  afs add-agent --force         refresh managed scaffolding\n")
	r.out.Printf("  afs add-agent --only-skills   install only skills\n")
	r.out.Printf("  afs add-agent --only-skills --force\n")
	r.out.Printf("                               refresh only skills\n")
	r.out.Printf("  afs add-agent-prompt [--force]\n")
	r.out.Printf("  afs add-ss-prompt\n")
}

func helperCommandForExecutable(executablePath string, args []string) ([]string, bool) {
	name := filepath.Base(executablePath)
	ext := filepath.Ext(name)
	if ext != "" {
		name = strings.TrimSuffix(name, ext)
	}

	switch name {
	case "add-agent", "add-agent-prompt", "add-ss-prompt":
		return append([]string{name}, args...), true
	default:
		return nil, false
	}
}
