package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/leanbusqts/agent47/internal/app"
	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunRestoresCallerDirectory(t *testing.T) {
	restoreAFSHooks()
	defer restoreAFSHooks()
	originalArgs := os.Args
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalDir) }()
	t.Cleanup(func() { os.Args = originalArgs })

	callerDir := t.TempDir()
	otherDir := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(otherDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(otherDir); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AGENT47_CALLER_DIR", callerDir)
	os.Args = []string{originalArgs[0], "help"}

	if status := run(); status != 0 {
		t.Fatalf("expected status 0, got %d", status)
	}

	currentDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	resolvedCurrentDir, err := filepath.EvalSymlinks(currentDir)
	if err != nil {
		t.Fatal(err)
	}
	resolvedCallerDir, err := filepath.EvalSymlinks(callerDir)
	if err != nil {
		t.Fatal(err)
	}
	if resolvedCurrentDir != resolvedCallerDir {
		t.Fatalf("expected cwd %s, got %s", resolvedCallerDir, resolvedCurrentDir)
	}
}

func TestRunReturnsOneWhenRestoreCallerDirectoryFails(t *testing.T) {
	restoreAFSHooks()
	defer restoreAFSHooks()
	originalArgs := os.Args
	os.Args = []string{originalArgs[0], "help"}
	t.Cleanup(func() { os.Args = originalArgs })
	t.Setenv("AGENT47_CALLER_DIR", "/missing")

	var stderr bytes.Buffer
	afsStderr = &stderr
	afsChdir = func(string) error { return errors.New("boom") }

	if status := run(); status != 1 {
		t.Fatalf("expected status 1, got %d", status)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Failed to restore caller directory")) {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunReturnsOneWhenDetectConfigFails(t *testing.T) {
	restoreAFSHooks()
	defer restoreAFSHooks()
	originalArgs := os.Args
	os.Args = []string{originalArgs[0], "help"}
	t.Cleanup(func() { os.Args = originalArgs })

	var stderr bytes.Buffer
	afsStderr = &stderr
	afsDetectConfig = func(string) (runtime.Config, error) { return runtime.Config{}, errors.New("detect fail") }

	if status := run(); status != 1 {
		t.Fatalf("expected status 1, got %d", status)
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Failed to detect runtime")) {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}

func TestRunPrefersExecutablePathFromOSExecutable(t *testing.T) {
	restoreAFSHooks()
	defer restoreAFSHooks()
	originalArgs := os.Args
	os.Args = []string{"afs", "version"}
	t.Cleanup(func() { os.Args = originalArgs })

	wantExecutable := "/tmp/installed/afs"
	afsExecutable = func() (string, error) { return wantExecutable, nil }
	afsDetectConfig = func(got string) (runtime.Config, error) {
		if got != wantExecutable {
			t.Fatalf("expected detect config to use os.Executable path %q, got %q", wantExecutable, got)
		}
		return runtime.Config{Version: "vtest"}, nil
	}

	if status := run(); status != 0 {
		t.Fatalf("expected status 0, got %d", status)
	}
}

func TestRunDelegatesToRootWithDetectedConfigAndArgs(t *testing.T) {
	restoreAFSHooks()
	defer restoreAFSHooks()
	originalArgs := os.Args
	os.Args = []string{originalArgs[0], "doctor", "--fail-on-warn"}
	t.Cleanup(func() { os.Args = originalArgs })

	cfg := runtime.Config{Version: "vtest", ExecutablePath: "/tmp/afs"}
	afsDetectConfig = func(string) (runtime.Config, error) { return cfg, nil }
	fake := &fakeRoot{status: 7}
	afsNewRoot = func(out cli.Output) rootRunner {
		fake.out = out
		return fake
	}

	if status := run(); status != 7 {
		t.Fatalf("expected status 7, got %d", status)
	}
	if fake.cfg.Version != "vtest" {
		t.Fatalf("unexpected cfg: %+v", fake.cfg)
	}
	if len(fake.args) != 2 || fake.args[0] != "doctor" || fake.args[1] != "--fail-on-warn" {
		t.Fatalf("unexpected args: %v", fake.args)
	}
}

func TestRunPassesSignalContextToRoot(t *testing.T) {
	restoreAFSHooks()
	defer restoreAFSHooks()
	originalArgs := os.Args
	os.Args = []string{originalArgs[0], "version"}
	t.Cleanup(func() { os.Args = originalArgs })

	afsDetectConfig = func(string) (runtime.Config, error) { return runtime.Config{Version: "vtest"}, nil }
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	afsSignalContext = func() (context.Context, context.CancelFunc) { return ctx, func() {} }
	fake := &fakeRoot{status: 1}
	afsNewRoot = func(out cli.Output) rootRunner { return fake }

	if status := run(); status != 1 {
		t.Fatalf("expected canceled root status, got %d", status)
	}
	if !errors.Is(fake.ctx.Err(), context.Canceled) {
		t.Fatalf("expected canceled signal context, got %v", fake.ctx.Err())
	}
}

type fakeRoot struct {
	out    cli.Output
	ctx    context.Context
	cfg    runtime.Config
	args   []string
	status int
}

func (f *fakeRoot) Run(ctx context.Context, cfg runtime.Config, args []string) int {
	f.ctx = ctx
	f.cfg = cfg
	f.args = append([]string{}, args...)
	return f.status
}

func restoreAFSHooks() {
	afsStdout = os.Stdout
	afsStderr = os.Stderr
	afsChdir = os.Chdir
	afsExecutable = os.Executable
	afsDetectConfig = runtime.DetectConfig
	afsNewRoot = func(out cli.Output) rootRunner { return app.NewRoot(out) }
	afsSignalContext = func() (context.Context, context.CancelFunc) {
		return signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	}
}
