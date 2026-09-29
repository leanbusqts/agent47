//go:build !windows

package install

import (
	"context"
	"errors"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func platformStartDeferredUninstall(runtime.Config) error {
	return errors.New("deferred uninstall is only supported on Windows")
}

func RunDeferredUninstallIfRequested(context.Context, runtime.Config, cli.Output) (bool, error) {
	return false, nil
}
