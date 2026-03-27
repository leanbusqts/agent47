package install

import (
	"path/filepath"
	"testing"

	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestDoctorPathHelpersMatchInternalPathsUnix(t *testing.T) {
	cfg := runtime.Config{
		HomeDir:     "/home/test",
		UserBinDir:  "/home/test/bin",
		Agent47Home: "/home/test/.agent47",
	}

	if ManagedBinaryPathForDoctor(cfg) != filepath.Join(cfg.Agent47Home, "bin", "afs") {
		t.Fatalf("unexpected managed binary path: %s", ManagedBinaryPathForDoctor(cfg))
	}
	if PublishedHelperPathForDoctor(cfg, "add-agent") != filepath.Join(cfg.UserBinDir, "add-agent") {
		t.Fatalf("unexpected helper path: %s", PublishedHelperPathForDoctor(cfg, "add-agent"))
	}
	if PublishedAfsPathForDoctor(cfg) != filepath.Join(cfg.UserBinDir, "afs") {
		t.Fatalf("unexpected afs path: %s", PublishedAfsPathForDoctor(cfg))
	}
}

func TestDoctorPathHelpersMatchInternalPathsWindows(t *testing.T) {
	cfg := runtime.Config{
		OS:          "windows",
		UserBinDir:  `C:\Users\Test\AppData\Local\agent47\bin`,
		Agent47Home: `C:\Users\Test\AppData\Local\agent47`,
	}

	if ManagedBinaryPathForDoctor(cfg) != filepath.Join(cfg.Agent47Home, "bin", "afs.exe") {
		t.Fatalf("unexpected managed binary path: %s", ManagedBinaryPathForDoctor(cfg))
	}
	if PublishedHelperPathForDoctor(cfg, "add-agent") != filepath.Join(cfg.UserBinDir, "add-agent.cmd") {
		t.Fatalf("unexpected helper path: %s", PublishedHelperPathForDoctor(cfg, "add-agent"))
	}
	if PublishedAfsPathForDoctor(cfg) != filepath.Join(cfg.UserBinDir, "afs.exe") {
		t.Fatalf("unexpected afs path: %s", PublishedAfsPathForDoctor(cfg))
	}
}
