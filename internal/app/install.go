package app

import (
	"context"
	"errors"
	"os"

	"github.com/leanbusqts/agent47/internal/install"
	"github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/templates"
)

const internalInstallCommand = "__agent47_internal_install"

func (r *Root) runInstallInternal(ctx context.Context, cfg runtime.Config, args []string) int {
	var opts install.InstallOptions
	postOpts := install.PostInstallOptions{
		SkipPathCheck: os.Getenv("AGENT47_SKIP_WINDOWS_POSTINSTALL_PATH_HINT") == "true",
	}

	for _, arg := range args {
		switch arg {
		case "--force":
			opts.Force = true
		case "--non-interactive":
			postOpts.NonInteractive = true
		default:
			r.out.Err("Unknown internal install flag: %s", arg)
			return 1
		}
	}

	service, err := install.New(cfg, r.out)
	if err != nil {
		r.out.Err("Failed to initialize install service: %v", err)
		return 1
	}
	if err := service.Install(ctx, cfg, opts); err != nil {
		var missingTemplateErr templates.MissingTemplateError
		if errors.As(err, &missingTemplateErr) {
			r.out.Err("Required install asset missing: %s", missingTemplateErr.Path)
			return 1
		}
		r.out.Err("%v", err)
		return 1
	}
	if err := install.RunPostInstall(ctx, cfg, r.out, postOpts); err != nil {
		r.out.Err("%v", err)
		return 1
	}
	return 0
}
