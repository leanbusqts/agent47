package app

import (
	"bytes"
	"context"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunPrintsHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, nil)
	if status != 0 {
		t.Fatalf("expected status 0, got %d", status)
	}
	if !bytes.Contains(stdout.Bytes(), []byte("afs help")) {
		t.Fatalf("expected help output, got %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("--force replaces rules/")) {
		t.Fatalf("expected destructive force behavior in help, got %s", stdout.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte("afs map [--force]")) {
		t.Fatalf("expected map command in help, got %s", stdout.String())
	}
}

func TestRunPrintsVersion(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{"version"})
	if status != 0 {
		t.Fatalf("expected status 0, got %d", status)
	}
	if stdout.String() != "vtest\n" {
		t.Fatalf("expected version output, got %q", stdout.String())
	}
}

func TestRunRejectsUnknownCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{"does-not-exist"})
	if status != 2 {
		t.Fatalf("expected usage status 2, got %d", status)
	}
	if stdout.Len() != 0 || !bytes.Contains(stderr.Bytes(), []byte("Unknown command: does-not-exist")) {
		t.Fatalf("expected diagnostic-only unknown command output, stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRunRejectsArgumentsForArgumentlessCommands(t *testing.T) {
	t.Parallel()

	for _, command := range []string{"help", "version"} {
		command := command
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			root := NewRoot(cli.NewOutput(&stdout, &stderr))

			status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{command, "extra"})
			if status != 2 {
				t.Fatalf("expected usage status 2, got %d", status)
			}
			if stdout.Len() != 0 || !bytes.Contains(stderr.Bytes(), []byte("Usage: afs "+command)) {
				t.Fatalf("expected diagnostic usage, stdout=%q stderr=%q", stdout.String(), stderr.String())
			}
		})
	}
}
