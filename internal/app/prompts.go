package app

import (
	"context"
	"os"

	"github.com/leanbusqts/agent47/internal/prompts"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func (r *Root) runAddAgentPrompt(ctx context.Context, cfg runtime.Config, args []string) int {
	_ = ctx
	force := false
	for _, arg := range args {
		switch arg {
		case "--force":
			force = true
		default:
			r.out.Printf("Usage: add-agent-prompt [--force]\n")
			return 1
		}
	}

	workDir, err := os.Getwd()
	if err != nil {
		r.out.Err("Failed to detect working directory: %v", err)
		return 1
	}

	service, err := prompts.New(cfg, r.out)
	if err != nil {
		r.out.Err("Failed to initialize prompts service: %v", err)
		return 1
	}
	if err := service.AddAgentPrompt(workDir, force); err != nil {
		r.out.Err("%v", err)
		return 1
	}
	return 0
}

func (r *Root) runAddSSPrompt(ctx context.Context, cfg runtime.Config, args []string) int {
	_ = ctx
	if len(args) > 0 {
		r.out.Printf("Usage: add-ss-prompt\n")
		return 1
	}

	service, err := prompts.New(cfg, r.out)
	if err != nil {
		r.out.Err("Failed to initialize prompts service: %v", err)
		return 1
	}
	if err := service.AddSSPrompt(); err != nil {
		r.out.Err("%v", err)
		return 1
	}
	return 0
}
