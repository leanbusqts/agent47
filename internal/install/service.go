package install

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/fsx"
	"github.com/leanbusqts/agent47/internal/manifest"
	"github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/templates"
)

type Service struct {
	FS     fsx.Service
	Loader *templates.Loader
	Out    cli.Output
}

type InstallOptions struct {
	Force bool
}

func New(cfg runtime.Config, out cli.Output) (*Service, error) {
	loader, err := templates.NewLoader(cfg.TemplateMode, cfg.RepoRoot)
	if err != nil {
		return nil, err
	}
	return &Service{
		FS:     fsx.Service{},
		Loader: loader,
		Out:    out,
	}, nil
}

func (s *Service) Install(ctx context.Context, cfg runtime.Config, opts InstallOptions) error {
	s.Out.Printf("[*] Installing agent47...\n")
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := validateRuntimePaths(cfg); err != nil {
		return err
	}

	manifestData, err := s.Loader.Source.ReadFile("manifest.txt")
	if err != nil {
		return err
	}
	m, err := manifest.Parse(manifestData)
	if err != nil {
		return err
	}
	bundleIDs, err := templates.DiscoverBundleIDs(s.Loader.RawSource)
	if err != nil {
		return err
	}
	if err := templates.ValidateAssembly(s.Loader.RawSource, bundleIDs); err != nil {
		return err
	}
	assembledManifest, err := templates.AssembleManifest(s.Loader.RawSource, bundleIDs)
	if err != nil {
		return err
	}
	m = assembledManifest

	if err := s.preflight(m, cfg); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := s.FS.MkdirAll(cfg.UserBinDir); err != nil {
		return err
	}
	if err := s.FS.MkdirAll(filepath.Join(cfg.Agent47Home, "bin")); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := s.installManagedTemplates(m, cfg, opts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := s.installManagedBinary(cfg, opts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.removeLegacyManagedLib(cfg)
	s.cleanupLegacy(cfg)
	if err := s.publishUserScripts(cfg, opts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.publishUserEntry(cfg, opts); err != nil {
		return err
	}

	s.Out.OK("afs installation complete")
	return nil
}

func (s *Service) Uninstall(ctx context.Context, cfg runtime.Config) error {
	s.Out.Printf("[*] Uninstalling afs scripts...\n")
	if err := ctx.Err(); err != nil {
		return err
	}

	if err := validateRuntimePaths(cfg); err != nil {
		return err
	}

	for _, script := range helperCommands {
		target := publishedHelperPath(cfg, script)
		if !s.FS.Exists(target) {
			s.Out.Warn("%s not found in %s", script, cfg.UserBinDir)
			continue
		}
		if isManagedPublishedHelper(cfg, target, script) {
			if err := os.Remove(target); err != nil {
				return err
			}
			s.Out.OK("Removed %s", script)
		} else {
			s.Out.Warn("Preserving unmanaged %s in %s", script, cfg.UserBinDir)
		}
	}

	userAfs := publishedAfsPath(cfg)
	if cfg.OS == "windows" {
		if userAfs != managedBinaryPath(cfg) && s.FS.Exists(userAfs) && isManagedPublishedEntry(cfg, userAfs) {
			if err := os.Remove(userAfs); err != nil {
				return err
			}
			s.Out.OK("Removed afs launcher")
		} else if userAfs != managedBinaryPath(cfg) && s.FS.Exists(userAfs) {
			s.Out.Warn("Preserving unmanaged afs launcher in %s", cfg.UserBinDir)
		}
	} else if isManagedPublishedEntry(cfg, userAfs) {
		if err := os.Remove(userAfs); err != nil {
			return err
		}
		s.Out.OK("Removed afs symlink")
	} else if s.FS.Exists(userAfs) {
		s.Out.Warn("Preserving unmanaged afs entry in %s", cfg.UserBinDir)
	}

	managedAfs := managedBinaryPath(cfg)
	if s.FS.Exists(managedAfs) {
		if err := os.Remove(managedAfs); err != nil {
			return err
		}
		s.Out.OK("Removed installed afs launcher")
	}

	for _, path := range []string{
		filepath.Join(cfg.Agent47Home, "bin"),
		filepath.Join(cfg.Agent47Home, "scripts"),
		filepath.Join(cfg.Agent47Home, "templates"),
		filepath.Join(cfg.Agent47Home, "cache"),
	} {
		_ = os.RemoveAll(path)
	}

	matches, _ := filepath.Glob(filepath.Join(cfg.Agent47Home, "templates.bak.*"))
	for _, match := range matches {
		_ = os.RemoveAll(match)
	}
	scriptBackups, _ := filepath.Glob(filepath.Join(cfg.Agent47Home, "scripts", "*.bak.*"))
	for _, match := range scriptBackups {
		_ = os.RemoveAll(match)
	}
	_ = os.Remove(filepath.Join(cfg.Agent47Home, "VERSION"))
	_ = os.Remove(cfg.Agent47Home)

	s.Out.OK("afs tools removed from system")
	return nil
}

func (s *Service) preflight(m manifest.Manifest, cfg runtime.Config) error {
	if cfg.ExecutablePath == "" {
		return fmt.Errorf("missing executable path")
	}
	if _, err := os.Stat(cfg.ExecutablePath); err != nil {
		return fmt.Errorf("Required install asset missing: %s", cfg.ExecutablePath)
	}
	for _, file := range m.RequiredTemplateFiles {
		if _, err := s.Loader.Source.Stat(file); err != nil {
			return templates.MissingTemplateError{Path: file}
		}
	}
	for _, dir := range m.RequiredTemplateDirs {
		if info, err := s.Loader.Source.Stat(dir); err != nil || !info.IsDir() {
			return templates.MissingTemplateError{Path: dir}
		}
	}
	for _, file := range m.RuleTemplates {
		if _, err := s.Loader.Source.Stat(filepath.ToSlash(filepath.Join("rules", file))); err != nil {
			return templates.MissingTemplateError{Path: filepath.ToSlash(filepath.Join("rules", file))}
		}
	}
	return nil
}

func (s *Service) installManagedTemplates(m manifest.Manifest, cfg runtime.Config, opts InstallOptions) error {
	s.Out.Printf("[*] Installing agent47 templates...\n")
	if err := s.FS.MkdirAll(cfg.Agent47Home); err != nil {
		return err
	}

	target := filepath.Join(cfg.Agent47Home, "templates")
	if s.FS.IsDir(target) && !opts.Force {
		s.Out.Warn("Templates already exist at %s (use --force to overwrite)", target)
	} else {
		if s.FS.IsDir(target) && opts.Force {
			s.Out.Warn("Overwriting existing templates at %s", target)
		}
		if err := s.replaceTemplateDir(target, opts.Force); err != nil {
			return err
		}
		s.Out.OK("Templates installed")
	}

	versionFile := filepath.Join(cfg.Agent47Home, "VERSION")
	if err := s.FS.WriteFileAtomic(versionFile, []byte(cfg.Version+"\n"), 0o644); err != nil {
		return err
	}
	s.Out.OK("VERSION installed")
	_ = m
	return nil
}

func (s *Service) installManagedBinary(cfg runtime.Config, opts InstallOptions) error {
	target := managedBinaryPath(cfg)
	if s.FS.Exists(target) && !opts.Force {
		s.Out.Warn("afs launcher already exists in %s (use --force to overwrite)", filepath.Join(cfg.Agent47Home, "bin"))
		return nil
	}

	if err := s.FS.CopyFile(cfg.ExecutablePath, target); err != nil {
		return err
	}
	if err := os.Chmod(target, 0o755); err != nil {
		return err
	}
	s.Out.OK("Installed afs launcher")
	return nil
}

func (s *Service) publishUserScripts(cfg runtime.Config, opts InstallOptions) error {
	var done []published

	managed := managedBinaryPath(cfg)
	for _, script := range helperCommands {
		target := publishedHelperPath(cfg, script)
		if s.FS.Exists(target) && !opts.Force {
			s.Out.Warn("%s already exists in %s (use --force to overwrite)", script, cfg.UserBinDir)
			continue
		}

		var state published
		state.name = script
		state.target = target
		if s.FS.Exists(target) {
			state.backup = target + ".bak.install"
			_ = os.Remove(state.backup)
			if err := s.FS.CopyFile(target, state.backup); err != nil {
				return err
			}
			state.hadBackup = true
		}

		if err := s.publishHelper(cfg, managed, script, target); err != nil {
			s.restorePublished(cfg, done)
			return err
		}
		done = append(done, state)
		s.Out.OK("Installed %s", script)
	}

	sort.Slice(done, func(i, j int) bool { return done[i].name < done[j].name })
	for _, item := range done {
		if item.hadBackup {
			_ = os.Remove(item.backup)
		}
	}
	return nil
}

func (s *Service) restorePublished(cfg runtime.Config, done []published) {
	for i := len(done) - 1; i >= 0; i-- {
		target := done[i].target
		_ = os.Remove(target)
		if done[i].hadBackup {
			_ = s.FS.CopyFile(done[i].backup, target)
			_ = os.Remove(done[i].backup)
		}
	}
}

type published struct {
	name      string
	target    string
	backup    string
	hadBackup bool
}

func (s *Service) publishUserEntry(cfg runtime.Config, opts InstallOptions) error {
	target := publishedAfsPath(cfg)
	managed := managedBinaryPath(cfg)
	if cfg.OS == "windows" {
		if target == managed {
			s.Out.OK("Managed afs launcher available in %s", cfg.UserBinDir)
			return nil
		}
		if s.FS.Exists(target) && !opts.Force {
			s.Out.Warn("afs entry already exists in %s (use --force to refresh)", cfg.UserBinDir)
			return nil
		}
		backup := target + ".bak.install"
		hadBackup := false
		if s.FS.Exists(target) {
			_ = os.Remove(backup)
			if err := s.FS.CopyFile(target, backup); err != nil {
				return err
			}
			hadBackup = true
		}
		if err := s.FS.CopyFile(managed, target); err != nil {
			if hadBackup {
				_ = os.Remove(target)
				_ = s.FS.CopyFile(backup, target)
				_ = os.Remove(backup)
			}
			return err
		}
		if err := os.Chmod(target, 0o755); err != nil {
			if hadBackup {
				_ = os.Remove(target)
				_ = s.FS.CopyFile(backup, target)
				_ = os.Remove(backup)
			}
			return err
		}
		if hadBackup {
			_ = os.Remove(backup)
		}
		s.Out.OK("Installed afs launcher into %s", cfg.UserBinDir)
		return nil
	}
	if s.FS.Exists(target) && !opts.Force {
		s.Out.Warn("afs entry already exists in ~/bin (use --force to refresh)")
		return nil
	}
	if sameManagedAndPublishedEntry(cfg) {
		return fmt.Errorf("unsafe runtime paths: published afs entry would point to itself")
	}
	if err := s.FS.SymlinkAtomic(managed, target); err != nil {
		return err
	}
	s.Out.OK("Linked afs into ~/bin -> %s", managed)
	return nil
}

func (s *Service) cleanupLegacy(cfg runtime.Config) {
	for _, script := range helperCommands {
		_ = os.Remove(filepath.Join(cfg.Agent47Home, "scripts", script))
		_ = os.Remove(filepath.Join(cfg.Agent47Home, "scripts", script+".cmd"))
	}
	_ = os.Remove(filepath.Join(cfg.Agent47Home, "scripts"))
}

func (s *Service) publishHelper(cfg runtime.Config, managedBinary, command, target string) error {
	if cfg.OS != "windows" {
		return s.FS.SymlinkAtomic(managedBinary, target)
	}

	body := strings.Join([]string{
		"@echo off",
		fmt.Sprintf("\"%%~dp0%s\" %s %%*", managedBinaryName(cfg), command),
		"exit /b %ERRORLEVEL%",
		"",
	}, "\r\n")
	return s.FS.WriteFileAtomic(target, []byte(body), 0o755)
}

func (s *Service) removeLegacyManagedLib(cfg runtime.Config) {
	_ = os.RemoveAll(filepath.Join(cfg.Agent47Home, "scripts", "lib"))
}

func (s *Service) replaceTemplateDir(target string, force bool) error {
	stageRoot := filepath.Dir(target)
	stageDir, err := os.MkdirTemp(stageRoot, ".templates.tmp.*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(stageDir)

	if err := s.copyTemplateTree(s.Loader.RawSource, ".", stageDir); err != nil {
		return err
	}

	result, err := s.FS.ReplaceDirAtomic(stageDir, target, force)
	if err != nil {
		return err
	}
	if result.BackupPath != "" {
		s.Out.Info("Backup created: %s", result.BackupPath)
	}
	return nil
}

func (s *Service) copyTemplateTree(src templates.Source, srcPath, dstPath string) error {
	entries, err := src.ReadDir(srcPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		return err
	}
	for _, entry := range entries {
		childSrc := filepath.ToSlash(filepath.Join(srcPath, entry.Name()))
		childDst := filepath.Join(dstPath, entry.Name())
		if entry.IsDir() {
			if err := s.copyTemplateTree(src, childSrc, childDst); err != nil {
				return err
			}
			continue
		}
		data, err := src.ReadFile(childSrc)
		if err != nil {
			return err
		}
		if err := s.FS.WriteFileAtomic(childDst, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func validateRuntimePaths(cfg runtime.Config) error {
	if cfg.Agent47Home == "" {
		return fmt.Errorf("missing agent47 home")
	}
	if cfg.HomeDir == "" {
		return fmt.Errorf("missing home dir")
	}
	if samePath(cfg.OS, cfg.Agent47Home, cfg.HomeDir) {
		return fmt.Errorf("unsafe runtime paths: AGENT47_HOME cannot be the same as HOME")
	}
	if samePath(cfg.OS, cfg.Agent47Home, cfg.UserBinDir) {
		return fmt.Errorf("unsafe runtime paths: AGENT47_HOME cannot be the same as the published bin directory")
	}
	if cfg.OS != "windows" && sameManagedAndPublishedEntry(cfg) {
		return fmt.Errorf("unsafe runtime paths: managed afs launcher would collide with the published afs entry")
	}
	return nil
}

func sameManagedAndPublishedEntry(cfg runtime.Config) bool {
	return samePath(cfg.OS, managedBinaryPath(cfg), publishedAfsPath(cfg))
}

func isManagedPublishedHelper(cfg runtime.Config, target, command string) bool {
	if cfg.OS != "windows" {
		info, err := os.Lstat(target)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return false
		}
		resolvedTarget, err := filepath.EvalSymlinks(target)
		if err != nil {
			return false
		}
		resolvedManaged, err := filepath.EvalSymlinks(managedBinaryPath(cfg))
		if err != nil {
			resolvedManaged = managedBinaryPath(cfg)
		}
		return resolvedTarget == resolvedManaged
	}

	data, err := os.ReadFile(target)
	if err != nil {
		return false
	}
	expected := strings.Join([]string{
		"@echo off",
		fmt.Sprintf("\"%%~dp0%s\" %s %%*", managedBinaryName(cfg), command),
		"exit /b %ERRORLEVEL%",
		"",
	}, "\r\n")
	return string(data) == expected
}

func isManagedPublishedEntry(cfg runtime.Config, target string) bool {
	if cfg.OS != "windows" {
		info, err := os.Lstat(target)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return false
		}
		resolvedTarget, err := filepath.EvalSymlinks(target)
		if err != nil {
			return false
		}
		resolvedManaged, err := filepath.EvalSymlinks(managedBinaryPath(cfg))
		if err != nil {
			resolvedManaged = managedBinaryPath(cfg)
		}
		return resolvedTarget == resolvedManaged
	}

	managedData, managedErr := os.ReadFile(managedBinaryPath(cfg))
	targetData, targetErr := os.ReadFile(target)
	if managedErr != nil || targetErr != nil {
		return false
	}
	return bytes.Equal(managedData, targetData)
}
