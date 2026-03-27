package install

import (
	"path/filepath"

	"github.com/leanbusqts/agent47/internal/runtime"
)

func managedBinaryName(cfg runtime.Config) string {
	if cfg.OS == "windows" {
		return "afs.exe"
	}
	return "afs"
}

func managedBinaryPath(cfg runtime.Config) string {
	return filepath.Join(cfg.Agent47Home, "bin", managedBinaryName(cfg))
}

func publishedHelperName(cfg runtime.Config, command string) string {
	if cfg.OS == "windows" {
		return command + ".cmd"
	}
	return command
}

func publishedHelperPath(cfg runtime.Config, command string) string {
	return filepath.Join(cfg.UserBinDir, publishedHelperName(cfg, command))
}

func publishedAfsPath(cfg runtime.Config) string {
	if cfg.OS == "windows" {
		return filepath.Join(cfg.UserBinDir, managedBinaryName(cfg))
	}
	return filepath.Join(cfg.UserBinDir, "afs")
}

func ManagedBinaryPathForDoctor(cfg runtime.Config) string {
	return managedBinaryPath(cfg)
}

func PublishedHelperPathForDoctor(cfg runtime.Config, command string) string {
	return publishedHelperPath(cfg, command)
}

func PublishedAfsPathForDoctor(cfg runtime.Config) string {
	return publishedAfsPath(cfg)
}
