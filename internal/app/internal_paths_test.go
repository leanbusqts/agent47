package app

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestRunUninstallRejectsUnexpectedArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	root := NewRoot(cli.NewOutput(&stdout, &stderr))

	status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{"uninstall", "unexpected"})
	if status != 2 {
		t.Fatalf("expected usage status 2, got %d", status)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "Unknown command: uninstall unexpected") {
		t.Fatalf("unexpected streams: stdout=%s stderr=%s", stdout.String(), stderr.String())
	}
}

func TestRemovedCommandsAreUnknown(t *testing.T) {
	for _, command := range []string{"add-agent", "add-agent-prompt", "add-ss-prompt"} {
		t.Run(command, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			root := NewRoot(cli.NewOutput(&stdout, &stderr))
			if status := root.Run(context.Background(), runtime.Config{Version: "vtest"}, []string{command}); status != 2 {
				t.Fatalf("expected status 2, got %d", status)
			}
			if stdout.Len() != 0 || !strings.Contains(stderr.String(), "Unknown command: "+command) {
				t.Fatalf("unexpected streams: stdout=%s stderr=%s", stdout.String(), stderr.String())
			}
		})
	}
}
