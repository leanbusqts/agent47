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
	if status == 0 {
		t.Fatal("expected non-zero status")
	}
	if !bytes.Contains(stdout.Bytes(), []byte("Unknown command: does-not-exist")) {
		t.Fatalf("expected unknown command output, got %s", stdout.String())
	}
}

func TestRunMapsAddAgentExecutableName(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{
		Version:        "vtest",
		ExecutablePath: "/tmp/add-agent-prompt",
	}, []string{"--force"})
	if status == 0 {
		t.Fatal("expected non-zero status because templates are not configured")
	}
	if !bytes.Contains(stderr.Bytes(), []byte("Failed to initialize prompts service")) {
		t.Fatalf("expected prompt command dispatch, got stderr %s", stderr.String())
	}
}
