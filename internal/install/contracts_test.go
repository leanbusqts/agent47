package install

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestHelperCommandsReturnsExpectedCopy(t *testing.T) {
	got := HelperCommands()
	want := []string{"add-agent", "add-agent-prompt", "add-ss-prompt"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected helper commands: got %v want %v", got, want)
	}

	got[0] = "mutated"
	again := HelperCommands()
	if !reflect.DeepEqual(again, want) {
		t.Fatalf("helper commands should be immutable copy, got %v", again)
	}
}

func TestReinstallHintVariesByPlatform(t *testing.T) {
	if !strings.Contains(ReinstallHint(runtime.Config{OS: "windows"}), "install.ps1") {
		t.Fatal("expected windows reinstall hint to mention install.ps1")
	}
	if !strings.Contains(ReinstallHint(runtime.Config{OS: "darwin"}), "install.sh") {
		t.Fatal("expected unix reinstall hint to mention install.sh")
	}
}

func TestUpdateInstructionsVariesByPlatformAndRepo(t *testing.T) {
	win := UpdateInstructions(runtime.Config{OS: "windows", RepoRoot: `C:\src\agent47`})
	if !strings.Contains(win, "install.ps1") {
		t.Fatalf("expected windows update instruction to mention install.ps1, got %q", win)
	}

	repoRoot := filepath.Join("/tmp", "agent47")
	unix := UpdateInstructions(runtime.Config{OS: "linux", RepoRoot: repoRoot})
	if !strings.Contains(unix, repoRoot) || !strings.Contains(unix, "./install.sh") {
		t.Fatalf("unexpected unix update instruction: %q", unix)
	}

	noRepo := UpdateInstructions(runtime.Config{OS: "linux"})
	if !strings.Contains(noRepo, "install.sh") {
		t.Fatalf("expected generic unix install instruction, got %q", noRepo)
	}
}
