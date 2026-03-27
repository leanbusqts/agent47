package update

import (
	"bytes"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestPrintUpdateAvailableUsesPlatformSpecificInstructions(t *testing.T) {
	stdout := &bytes.Buffer{}
	service := New(cli.NewOutput(stdout, stdout))

	service.print(CacheRecord{
		Status:        "update-available",
		LocalVersion:  "1.0.0",
		LatestVersion: "1.1.0",
	}, runtime.Config{OS: "windows", RepoRoot: `C:\src\agent47`})

	output := stdout.String()
	if !strings.Contains(output, "Update available: 1.0.0 -> 1.1.0") {
		t.Fatalf("expected update available marker, got %q", output)
	}
	if !strings.Contains(output, "install.ps1") {
		t.Fatalf("expected windows update instructions, got %q", output)
	}
}

func TestPrintVersionDiffersStillSupportedForCacheCompatibility(t *testing.T) {
	stdout := &bytes.Buffer{}
	service := New(cli.NewOutput(stdout, stdout))

	service.print(CacheRecord{
		Status:        "version-differs",
		LocalVersion:  "1.0.0",
		LatestVersion: "1.1.0",
	}, runtime.Config{OS: "linux"})

	output := stdout.String()
	if !strings.Contains(output, "Remote VERSION differs from local: 1.0.0 vs 1.1.0") {
		t.Fatalf("expected compatibility output, got %q", output)
	}
}
