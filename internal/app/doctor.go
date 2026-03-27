package app

import (
	"context"

	"github.com/leanbusqts/agent47/internal/doctor"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func (r *Root) runDoctor(ctx context.Context, cfg runtime.Config, args []string) int {
	var opts doctor.Options

	for _, arg := range args {
		switch arg {
		case "--check-update":
			opts.CheckUpdate = true
		case "--check-update-force":
			opts.CheckUpdate = true
			opts.ForceUpdate = true
		case "--fail-on-warn":
			opts.FailOnWarn = true
		default:
			r.out.Printf("Usage: afs doctor [--check-update|--check-update-force|--fail-on-warn]\n")
			return 1
		}
	}

	service, err := doctor.New(cfg, r.out)
	if err != nil {
		r.out.Err("Failed to initialize doctor service: %v", err)
		return 1
	}
	if err := service.Run(ctx, cfg, opts); err != nil {
		r.out.Err("%v", err)
		return 1
	}
	return 0
}
