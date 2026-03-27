package platform

import (
	"runtime"
	"testing"
)

func TestOSMatchesRuntime(t *testing.T) {
	if OS() != runtime.GOOS {
		t.Fatalf("expected %s, got %s", runtime.GOOS, OS())
	}
}

func TestIsWindowsMatchesRuntime(t *testing.T) {
	if IsWindows() != (runtime.GOOS == "windows") {
		t.Fatalf("unexpected windows detection for %s", runtime.GOOS)
	}
}
