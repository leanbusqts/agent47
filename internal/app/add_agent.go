package app

import (
	"context"
	"fmt"

	"github.com/leanbusqts/agent47/internal/bootstrap"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func (r *Root) runAddAgent(ctx context.Context, cfg runtime.Config, args []string) int {
	var opts bootstrap.Options

	for _, arg := range args {
		switch arg {
		case "--force":
			opts.Force = true
		case "--only-skills":
			opts.OnlySkills = true
		default:
			r.out.Printf("Usage: add-agent [--force] [--only-skills]\n")
			return 1
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
	return fmt.Sprintf("%v", err)
}
