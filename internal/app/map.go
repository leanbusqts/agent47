package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/leanbusqts/agent47/internal/analyze"
	"github.com/leanbusqts/agent47/internal/contextmap"
	"github.com/leanbusqts/agent47/internal/initrepo"
	"github.com/leanbusqts/agent47/internal/runtime"
)

const mapUsage = "Usage: afs map [--force]"

func (r *Root) runMap(ctx context.Context, cfg runtime.Config, args []string) int {
	force, ok := parseMapOptions(args, r)
	if !ok {
		return 2
	}
	workDir, err := os.Getwd()
	if err != nil {
		r.out.Err("Failed to read working directory: %v", err)
		return 1
	}
	if err := validateLocalContextPath(workDir, cfg); err != nil {
		r.out.Err("%v", err)
		return 1
	}
	r.out.Info("Mapping repository...")
	analysis, err := (analyze.Service{}).AnalyzeContext(ctx, workDir, analyze.AnalyzeOptions{})
	if err != nil {
		r.out.Err("%v", err)
		return 1
	}
	document, err := contextmap.Build(ctx, workDir, analysis, contextmap.BuildOptions{})
	if err != nil {
		r.out.Err("%v", err)
		return 1
	}
	action, err := initrepo.NewContextService().Run(ctx, initrepo.ContextOptions{
		WorkDir: workDir,
		Content: document.Content,
		Force:   force,
	})
	if err != nil {
		r.out.Err("%v", err)
		return 1
	}
	switch action {
	case contextmap.ActionCreate:
		r.out.OK("Created %s.", contextmap.TargetPath)
	case contextmap.ActionUpdate:
		r.out.OK("Updated %s.", contextmap.TargetPath)
	case contextmap.ActionCurrent:
		r.out.OK("%s is current.", contextmap.TargetPath)
	}
	return 0
}

func parseMapOptions(args []string, r *Root) (bool, bool) {
	force := false
	for _, arg := range args {
		if arg != "--force" || force {
			r.out.Diagnosticf("%s\n", mapUsage)
			return false, false
		}
		force = true
	}
	return force, true
}

func validateLocalContextPath(workDir string, cfg runtime.Config) error {
	if cfg.Agent47Home == "" {
		return nil
	}
	root, err := filepath.Abs(workDir)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return err
	}
	local := filepath.Clean(filepath.Join(root, ".agent47"))
	runtimeHome, err := canonicalPotentialPath(cfg.Agent47Home)
	if err != nil {
		return err
	}
	equal := local == runtimeHome
	if cfg.OS == "windows" {
		equal = strings.EqualFold(local, runtimeHome)
	}
	if equal {
		return errors.New("project .agent47 directory conflicts with the Agent47 runtime home")
	}
	return nil
}

func canonicalPotentialPath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	parent, err := filepath.EvalSymlinks(filepath.Dir(abs))
	if err != nil {
		return "", err
	}
	return filepath.Clean(filepath.Join(parent, filepath.Base(abs))), nil
}
