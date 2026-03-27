package app

import (
	"context"

	"github.com/leanbusqts/agent47/internal/install"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func (r *Root) runUninstall(ctx context.Context, cfg runtime.Config, args []string) int {
	if len(args) > 0 {
		r.out.Printf("Unknown command: uninstall %s\n", args[0])
		return 1
	}

	service, err := install.New(cfg, r.out)
	if err != nil {
		r.out.Err("Failed to initialize install service: %v", err)
		return 1
	}
	if err := service.Uninstall(ctx, cfg); err != nil {
		r.out.Err("%v", err)
		return 1
	}
	return 0
}
