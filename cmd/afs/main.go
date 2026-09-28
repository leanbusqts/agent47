package main

import (
	"context"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/leanbusqts/agent47/internal/app"
	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

type rootRunner interface {
	Run(context.Context, runtime.Config, []string) int
}

var (
	afsStdout        io.Writer = os.Stdout
	afsStderr        io.Writer = os.Stderr
	afsChdir                   = os.Chdir
	afsExecutable              = os.Executable
	afsDetectConfig            = runtime.DetectConfig
	afsNewRoot                 = func(out cli.Output) rootRunner { return app.NewRoot(out) }
	afsSignalContext           = func() (context.Context, context.CancelFunc) {
		return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	}
)

func main() {
	os.Exit(run())
}

func run() int {
	out := cli.NewOutput(afsStdout, afsStderr)

	if callerDir := os.Getenv("AGENT47_CALLER_DIR"); callerDir != "" {
		if err := afsChdir(callerDir); err != nil {
			out.Err("Failed to restore caller directory: %v", err)
			return 1
		}
	}

	executablePath, err := afsExecutable()
	if err != nil {
		executablePath = os.Args[0]
	}

	cfg, err := afsDetectConfig(executablePath)
	if err != nil {
		out.Err("Failed to detect runtime: %v", err)
		return 1
	}

	root := afsNewRoot(out)
	ctx, stop := afsSignalContext()
	defer stop()
	return root.Run(ctx, cfg, os.Args[1:])
}
