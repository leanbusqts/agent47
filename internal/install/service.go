package install

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/fsx"
	"github.com/leanbusqts/agent47/internal/manifest"
	"github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/templates"
)

type Service struct {
	FS           fsx.Service
	Loader       *templates.Loader
	Out          cli.Output
	beforeRemove func(string)
}

var startDeferredUninstall = platformStartDeferredUninstall

type InstallOptions struct {
	Force bool
}

const (
	runtimeOwnershipMarker = ".agent47-owned"
	runtimeOwnershipValue  = "agent47-runtime:v1\n"
	backupOwnershipMarker  = ".agent47-backup-owned"
	backupOwnershipPrefix  = "agent47-template-backup:v1\nsha256:"
)

type installTransaction struct {
	rollbacks []func() error
}

func (tx *installTransaction) add(rollback func() error) {
	if rollback != nil {
		tx.rollbacks = append(tx.rollbacks, rollback)
	}
}

func (tx *installTransaction) fail(cause error) error {
	var rollbackErrors []error
	for index := len(tx.rollbacks) - 1; index >= 0; index-- {
		if err := tx.rollbacks[index](); err != nil {
			rollbackErrors = append(rollbackErrors, err)
		}
	}
	if len(rollbackErrors) == 0 {
		return cause
	}
	return errors.Join(cause, fmt.Errorf("install rollback failed: %w", errors.Join(rollbackErrors...)))
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
	ownershipRollback, err := s.ensureRuntimeOwnership(cfg)
	if err != nil {
		return err
	}
	tx := &installTransaction{}
	tx.add(ownershipRollback)

	manifestData, err := s.Loader.Source.ReadFile("manifest.txt")
	if err != nil {
		return tx.fail(err)
	}
	m, err := manifest.Parse(manifestData)
	if err != nil {
		return tx.fail(err)
	}
	bundleIDs, err := templates.DiscoverBundleIDs(s.Loader.RawSource)
	if err != nil {
		return tx.fail(err)
	}
	if err := templates.ValidateAssembly(s.Loader.RawSource, bundleIDs); err != nil {
		return tx.fail(err)
	}
	assembledManifest, err := templates.AssembleManifest(s.Loader.RawSource, bundleIDs)
	if err != nil {
		return tx.fail(err)
	}
	m = assembledManifest

	if err := s.preflight(m, cfg); err != nil {
		return tx.fail(err)
	}
	if err := ctx.Err(); err != nil {
		return tx.fail(err)
	}

	if err := s.FS.MkdirAll(cfg.UserBinDir); err != nil {
		return tx.fail(err)
	}
	if err := s.FS.MkdirAll(filepath.Join(cfg.Agent47Home, "bin")); err != nil {
		return tx.fail(err)
	}
	if err := ctx.Err(); err != nil {
		return tx.fail(err)
	}

	templateRollback, err := s.installManagedTemplates(m, cfg, opts)
	if err != nil {
		return tx.fail(err)
	}
	tx.add(templateRollback)
	if err := ctx.Err(); err != nil {
		return tx.fail(err)
	}

	versionPath := filepath.Join(cfg.Agent47Home, "VERSION")
	versionSnapshot, err := capturePathSnapshot(versionPath)
	if err != nil {
		return tx.fail(err)
	}
	if err := s.FS.WriteFileAtomic(versionPath, []byte(cfg.Version+"\n"), 0o644); err != nil {
		return tx.fail(err)
	}
	installedVersion, err := capturePathSnapshot(versionPath)
	if err != nil {
		return tx.fail(err)
	}
	tx.add(func() error { return versionSnapshot.restoreIfUnchanged(s.FS, installedVersion) })
	s.Out.OK("VERSION installed")

	binarySnapshot, err := capturePathSnapshot(managedBinaryPath(cfg))
	if err != nil {
		return tx.fail(err)
	}
	if err := s.installManagedBinary(cfg, opts); err != nil {
		return tx.fail(err)
	}
	installedBinary, err := capturePathSnapshot(managedBinaryPath(cfg))
	if err != nil {
		return tx.fail(err)
	}
	tx.add(func() error { return binarySnapshot.restoreIfUnchanged(s.FS, installedBinary) })
	if err := ctx.Err(); err != nil {
		return tx.fail(err)
	}

	userEntry := publishedAfsPath(cfg)
	var publishedSnapshot pathSnapshot
	trackPublished := !samePath(cfg.OS, userEntry, managedBinaryPath(cfg))
	if trackPublished {
		publishedSnapshot, err = capturePathSnapshot(userEntry)
		if err != nil {
			return tx.fail(err)
		}
	}
	if err := s.publishUserEntry(cfg, opts); err != nil {
		return tx.fail(err)
	}
	if trackPublished {
		installedPublished, err := capturePathSnapshot(userEntry)
		if err != nil {
			return tx.fail(err)
		}
		tx.add(func() error { return publishedSnapshot.restoreIfUnchanged(s.FS, installedPublished) })
	}
	if err := ctx.Err(); err != nil {
		return tx.fail(err)
	}

	if err := s.cleanupLegacyArtifacts(cfg); err != nil {
		s.Out.Warn("Installation succeeded, but legacy cleanup was incomplete: %v", err)
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
	owned, err := runtimeHomeOwned(cfg)
	if err != nil {
		return err
	}
	if !owned {
		s.Out.Warn("No owned agent47 runtime found at %s; preserving it", cfg.Agent47Home)
		return nil
	}
	runtimeRootInfo, err := os.Lstat(cfg.Agent47Home)
	if err != nil {
		return err
	}

	for _, script := range legacyHelperCommands {
		target := publishedHelperPath(cfg, script)
		if !s.FS.Exists(target) {
			continue
		}
		if isManagedPublishedHelper(cfg, target, script) {
			if err := s.FS.Remove(target); err != nil {
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
			if err := s.FS.Remove(userAfs); err != nil {
				return err
			}
			s.Out.OK("Removed afs launcher")
		} else if userAfs != managedBinaryPath(cfg) && s.FS.Exists(userAfs) {
			s.Out.Warn("Preserving unmanaged afs launcher in %s", cfg.UserBinDir)
		}
	} else if isManagedPublishedEntry(cfg, userAfs) {
		if err := s.FS.Remove(userAfs); err != nil {
			return err
		}
		s.Out.OK("Removed afs symlink")
	} else if s.FS.Exists(userAfs) {
		s.Out.Warn("Preserving unmanaged afs entry in %s", cfg.UserBinDir)
	}

	managedAfs := managedBinaryPath(cfg)
	deferSelfRemoval := cfg.OS == "windows" && samePath(cfg.OS, cfg.ExecutablePath, managedAfs)
	if s.FS.Exists(managedAfs) && !deferSelfRemoval {
		if err := s.guardedRemoveRuntimePath(cfg, runtimeRootInfo, managedAfs, false); err != nil {
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
		if deferSelfRemoval && samePath(cfg.OS, path, filepath.Join(cfg.Agent47Home, "bin")) {
			continue
		}
		if err := s.guardedRemoveRuntimePath(cfg, runtimeRootInfo, path, true); err != nil {
			return err
		}
	}

	if err := s.removeOwnedTemplateBackups(cfg, runtimeRootInfo); err != nil {
		return err
	}
	if err := s.guardedRemoveRuntimePath(cfg, runtimeRootInfo, filepath.Join(cfg.Agent47Home, "VERSION"), false); err != nil {
		return err
	}
	if deferSelfRemoval {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := startDeferredUninstall(cfg); err != nil {
			return fmt.Errorf("schedule Windows self-uninstall: %w", err)
		}
		s.Out.OK("Scheduled installed afs launcher removal after process exit")
		s.Out.OK("afs tools removed from system")
		return nil
	}
	if err := s.guardedRemoveRuntimePath(cfg, runtimeRootInfo, filepath.Join(cfg.Agent47Home, runtimeOwnershipMarker), false); err != nil {
		return err
	}
	if s.beforeRemove != nil {
		s.beforeRemove(cfg.Agent47Home)
	}
	if err := assertRuntimeRootUnchanged(cfg, runtimeRootInfo); err != nil {
		return err
	}
	if err := s.FS.Remove(cfg.Agent47Home); err != nil {
		if directoryHasEntries(cfg.Agent47Home) {
			s.Out.Warn("Preserving non-Agent47 content in %s", cfg.Agent47Home)
		} else {
			return err
		}
	}

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

func (s *Service) installManagedTemplates(m manifest.Manifest, cfg runtime.Config, opts InstallOptions) (func() error, error) {
	s.Out.Printf("[*] Installing agent47 templates...\n")
	if err := s.FS.MkdirAll(cfg.Agent47Home); err != nil {
		return nil, err
	}

	target := filepath.Join(cfg.Agent47Home, "templates")
	if s.FS.IsDir(target) && !opts.Force {
		s.Out.Warn("Templates already exist at %s (use --force to overwrite)", target)
		return nil, nil
	} else {
		if s.FS.IsDir(target) && opts.Force {
			s.Out.Warn("Overwriting existing templates at %s", target)
		}
		result, err := s.replaceTemplateDir(target, opts.Force)
		if err != nil {
			return nil, err
		}
		s.Out.OK("Templates installed")
		restorePrevious := func() error {
			if err := removeExactOwnedPath(s.FS, target, true); err != nil {
				return err
			}
			if result.BackupPath != "" {
				marker := filepath.Join(result.BackupPath, backupOwnershipMarker)
				if err := s.FS.Remove(marker); err != nil && !errors.Is(err, fs.ErrNotExist) {
					return err
				}
				if err := os.Rename(result.BackupPath, target); err != nil {
					return err
				}
			}
			return nil
		}
		if result.BackupPath != "" {
			marker := filepath.Join(result.BackupPath, backupOwnershipMarker)
			backupDigest, digestErr := directoryDigest(result.BackupPath)
			if digestErr != nil {
				return nil, errors.Join(digestErr, restorePrevious())
			}
			markerValue := backupOwnershipPrefix + backupDigest + "\n"
			if err := s.FS.WriteFileAtomic(marker, []byte(markerValue), 0o600); err != nil {
				return nil, errors.Join(err, restorePrevious())
			}
		}
		installedDigest, err := directoryDigest(target)
		if err != nil {
			return nil, errors.Join(err, restorePrevious())
		}
		rollback := func() error {
			currentDigest, err := directoryDigest(target)
			if err != nil {
				return err
			}
			if currentDigest != installedDigest {
				return fmt.Errorf("refusing to overwrite concurrently modified templates during rollback: %s", target)
			}
			return restorePrevious()
		}
		_ = m
		return rollback, nil
	}
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
	s.Out.OK("Installed afs launcher")
	return nil
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
		if err := s.FS.CopyFile(managed, target); err != nil {
			return err
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

func (s *Service) cleanupLegacy(cfg runtime.Config) error {
	var cleanupErrors []error
	rootInfo, err := os.Lstat(cfg.Agent47Home)
	if err != nil {
		return err
	}
	for _, script := range legacyHelperCommands {
		for _, target := range []string{
			filepath.Join(cfg.Agent47Home, "scripts", script),
			filepath.Join(cfg.Agent47Home, "scripts", script+".cmd"),
		} {
			if err := s.guardedRemoveRuntimePath(cfg, rootInfo, target, false); err != nil && !errors.Is(err, fs.ErrNotExist) {
				cleanupErrors = append(cleanupErrors, err)
			}
		}
	}
	scriptsDir := filepath.Join(cfg.Agent47Home, "scripts")
	if err := s.guardedRemoveRuntimePath(cfg, rootInfo, scriptsDir, false); err != nil && !errors.Is(err, fs.ErrNotExist) {
		if !directoryHasEntries(scriptsDir) {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	return errors.Join(cleanupErrors...)
}

func (s *Service) cleanupLegacyPublishedHelpers(cfg runtime.Config) error {
	var cleanupErrors []error
	for _, command := range legacyHelperCommands {
		target := publishedHelperPath(cfg, command)
		if !s.FS.Exists(target) {
			continue
		}
		if isManagedPublishedHelper(cfg, target, command) {
			if err := s.FS.Remove(target); err != nil {
				cleanupErrors = append(cleanupErrors, err)
			}
			continue
		}
		s.Out.Warn("Preserving unmanaged legacy helper %s in %s", command, cfg.UserBinDir)
	}
	return errors.Join(cleanupErrors...)
}

func (s *Service) removeLegacyManagedLib(cfg runtime.Config) error {
	rootInfo, err := os.Lstat(cfg.Agent47Home)
	if err != nil {
		return err
	}
	return s.guardedRemoveRuntimePath(cfg, rootInfo, filepath.Join(cfg.Agent47Home, "scripts", "lib"), true)
}

func (s *Service) cleanupLegacyArtifacts(cfg runtime.Config) error {
	return errors.Join(
		s.removeLegacyManagedLib(cfg),
		s.cleanupLegacy(cfg),
		s.cleanupLegacyPublishedHelpers(cfg),
	)
}

func (s *Service) replaceTemplateDir(target string, force bool) (fsx.ReplaceDirResult, error) {
	stageRoot := filepath.Dir(target)
	stageDir, err := os.MkdirTemp(stageRoot, ".templates.tmp.*")
	if err != nil {
		return fsx.ReplaceDirResult{}, err
	}
	defer func() { _ = os.RemoveAll(stageDir) }()

	if err := s.copyTemplateTree(s.Loader.RawSource, ".", stageDir); err != nil {
		return fsx.ReplaceDirResult{}, err
	}

	result, err := s.FS.ReplaceDirAtomic(stageDir, target, force)
	if err != nil {
		return fsx.ReplaceDirResult{}, err
	}
	if result.BackupPath != "" {
		s.Out.Info("Backup created: %s", result.BackupPath)
	}
	return result, nil
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
	cleanAgentHome, err := runtime.ValidateAgent47HomePath(cfg.HomeDir, cfg.Agent47Home, cfg.UserBinDir, cfg.OS == "windows")
	if err != nil {
		return err
	}
	if !samePath(cfg.OS, cleanAgentHome, cfg.Agent47Home) {
		return fmt.Errorf("unsafe runtime paths: AGENT47_HOME must be absolute and canonical")
	}
	if cfg.OS != "windows" && sameManagedAndPublishedEntry(cfg) {
		return fmt.Errorf("unsafe runtime paths: managed afs launcher would collide with the published afs entry")
	}
	return validateRuntimeEntryTypes(cfg)
}

func validateRuntimeEntryTypes(cfg runtime.Config) error {
	for _, path := range []string{
		cfg.Agent47Home,
		filepath.Join(cfg.Agent47Home, "bin"),
		filepath.Join(cfg.Agent47Home, "scripts"),
		filepath.Join(cfg.Agent47Home, "templates"),
		filepath.Join(cfg.Agent47Home, "cache"),
	} {
		info, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe runtime path is a symlink: %s", path)
		}
		if !info.IsDir() {
			return fmt.Errorf("runtime directory path is not a directory: %s", path)
		}
	}
	for _, path := range []string{
		filepath.Join(cfg.Agent47Home, runtimeOwnershipMarker),
		filepath.Join(cfg.Agent47Home, "VERSION"),
	} {
		info, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("runtime metadata path must be a regular file: %s", path)
		}
	}
	return nil
}

func (s *Service) ensureRuntimeOwnership(cfg runtime.Config) (func() error, error) {
	homeCreated := false
	info, err := os.Lstat(cfg.Agent47Home)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := s.FS.MkdirAll(cfg.Agent47Home); err != nil {
			return nil, err
		}
		homeCreated = true
	case err != nil:
		return nil, err
	case info.Mode()&os.ModeSymlink != 0 || !info.IsDir():
		return nil, fmt.Errorf("unsafe runtime home type: %s", cfg.Agent47Home)
	}

	owned, err := runtimeHomeOwned(cfg)
	if err != nil {
		return nil, err
	}
	if owned {
		return nil, nil
	}

	empty, err := directoryEmpty(cfg.Agent47Home)
	if err != nil {
		return nil, err
	}
	if !empty && !legacyRuntimeOwned(cfg) {
		return nil, fmt.Errorf("refusing to claim non-Agent47 directory: %s", cfg.Agent47Home)
	}

	markerPath := filepath.Join(cfg.Agent47Home, runtimeOwnershipMarker)
	if err := s.FS.WriteFileAtomic(markerPath, []byte(runtimeOwnershipValue), 0o600); err != nil {
		if homeCreated {
			_ = os.Remove(cfg.Agent47Home)
		}
		return nil, err
	}
	return func() error {
		var rollbackErrors []error
		owned, err := runtimeHomeOwned(cfg)
		if err != nil {
			return err
		}
		if !owned {
			return fmt.Errorf("refusing to remove changed ownership marker during rollback: %s", markerPath)
		}
		if err := s.FS.Remove(markerPath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			rollbackErrors = append(rollbackErrors, err)
		}
		if homeCreated {
			for _, child := range []string{"cache", "scripts", "templates", "bin"} {
				if err := s.FS.Remove(filepath.Join(cfg.Agent47Home, child)); err != nil && !errors.Is(err, fs.ErrNotExist) {
					rollbackErrors = append(rollbackErrors, err)
				}
			}
			if err := s.FS.Remove(cfg.Agent47Home); err != nil && !errors.Is(err, fs.ErrNotExist) {
				rollbackErrors = append(rollbackErrors, err)
			}
		}
		return errors.Join(rollbackErrors...)
	}, nil
}

func runtimeHomeOwned(cfg runtime.Config) (bool, error) {
	markerPath := filepath.Join(cfg.Agent47Home, runtimeOwnershipMarker)
	info, err := os.Lstat(markerPath)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return false, fmt.Errorf("runtime ownership marker must be a regular file: %s", markerPath)
	}
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return false, err
	}
	if string(data) != runtimeOwnershipValue {
		return false, fmt.Errorf("invalid runtime ownership marker: %s", markerPath)
	}
	return true, nil
}

func legacyRuntimeOwned(cfg runtime.Config) bool {
	for _, path := range []string{
		filepath.Join(cfg.Agent47Home, "VERSION"),
		filepath.Join(cfg.Agent47Home, "templates", "manifest.txt"),
	} {
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return false
		}
	}
	return true
}

func directoryEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}
	return len(entries) == 0, nil
}

func directoryHasEntries(path string) bool {
	entries, err := os.ReadDir(path)
	return err == nil && len(entries) > 0
}

type pathSnapshot struct {
	path       string
	exists     bool
	mode       fs.FileMode
	data       []byte
	linkTarget string
}

func capturePathSnapshot(path string) (pathSnapshot, error) {
	snapshot := pathSnapshot{path: path}
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return snapshot, nil
	}
	if err != nil {
		return pathSnapshot{}, err
	}
	snapshot.exists = true
	snapshot.mode = info.Mode()
	if info.Mode()&os.ModeSymlink != 0 {
		snapshot.linkTarget, err = os.Readlink(path)
		return snapshot, err
	}
	if !info.Mode().IsRegular() {
		return pathSnapshot{}, fmt.Errorf("managed file path is not a regular file: %s", path)
	}
	snapshot.data, err = os.ReadFile(path)
	return snapshot, err
}

func (snapshot pathSnapshot) restore(service fsx.Service) error {
	info, err := os.Lstat(snapshot.path)
	if err == nil {
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("refusing to replace directory during rollback: %s", snapshot.path)
		}
		if err := service.Remove(snapshot.path); err != nil {
			return err
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if !snapshot.exists {
		return nil
	}
	if snapshot.mode&os.ModeSymlink != 0 {
		return os.Symlink(snapshot.linkTarget, snapshot.path)
	}
	return service.WriteFileAtomic(snapshot.path, snapshot.data, snapshot.mode.Perm())
}

func (snapshot pathSnapshot) restoreIfUnchanged(service fsx.Service, installed pathSnapshot) error {
	current, err := capturePathSnapshot(snapshot.path)
	if err != nil {
		return err
	}
	if !current.equal(installed) {
		return fmt.Errorf("refusing to overwrite concurrently modified path during rollback: %s", snapshot.path)
	}
	return snapshot.restore(service)
}

func (snapshot pathSnapshot) equal(other pathSnapshot) bool {
	return snapshot.exists == other.exists &&
		snapshot.mode == other.mode &&
		snapshot.linkTarget == other.linkTarget &&
		bytes.Equal(snapshot.data, other.data)
}

func directoryDigest(root string) (string, error) {
	return directoryDigestExcluding(root, "")
}

func directoryDigestExcluding(root string, excludedPath string) (string, error) {
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if excludedPath != "" && filepath.ToSlash(rel) == filepath.ToSlash(excludedPath) {
			return nil
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("managed template tree contains a symlink: %s", path)
		}
		if _, err := fmt.Fprintf(hash, "%s\x00%s\x00", filepath.ToSlash(rel), info.Mode().String()); err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("managed template tree contains an unsupported entry: %s", path)
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(hash, file)
		closeErr := file.Close()
		return errors.Join(copyErr, closeErr)
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func removeExactOwnedPath(service fsx.Service, path string, recursive bool) error {
	info, err := os.Lstat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if recursive && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return service.RemoveAll(path)
	}
	if !recursive && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("refusing to remove directory as a managed file: %s", path)
	}
	return service.Remove(path)
}

func (s *Service) removeOwnedTemplateBackups(cfg runtime.Config, rootInfo os.FileInfo) error {
	entries, err := os.ReadDir(cfg.Agent47Home)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), "templates.bak.") {
			continue
		}
		path := filepath.Join(cfg.Agent47Home, entry.Name())
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			s.Out.Warn("Preserving unverified template backup at %s", path)
			continue
		}
		marker := filepath.Join(path, backupOwnershipMarker)
		markerInfo, markerErr := os.Lstat(marker)
		if markerErr != nil || markerInfo.Mode()&os.ModeSymlink != 0 || !markerInfo.Mode().IsRegular() {
			s.Out.Warn("Preserving unverified template backup at %s", path)
			continue
		}
		markerData, markerErr := os.ReadFile(marker)
		markerValue := strings.TrimSuffix(string(markerData), "\n")
		if markerErr != nil || !strings.HasPrefix(markerValue, strings.TrimSuffix(backupOwnershipPrefix, "\n")) {
			s.Out.Warn("Preserving unverified template backup at %s", path)
			continue
		}
		expectedDigest := strings.TrimPrefix(markerValue, strings.TrimSuffix(backupOwnershipPrefix, "\n"))
		actualDigest, digestErr := directoryDigestExcluding(path, backupOwnershipMarker)
		if digestErr != nil || expectedDigest == "" || actualDigest != expectedDigest {
			s.Out.Warn("Preserving modified template backup at %s", path)
			continue
		}
		if err := s.guardedRemoveRuntimePath(cfg, rootInfo, path, true); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) guardedRemoveRuntimePath(cfg runtime.Config, rootInfo os.FileInfo, path string, recursive bool) error {
	if s.beforeRemove != nil {
		s.beforeRemove(path)
	}
	if err := assertRuntimeRootUnchanged(cfg, rootInfo); err != nil {
		return err
	}
	return removeExactOwnedPath(s.FS, path, recursive)
}

func assertRuntimeRootUnchanged(cfg runtime.Config, expected os.FileInfo) error {
	current, err := os.Lstat(cfg.Agent47Home)
	if err != nil {
		return err
	}
	if current.Mode()&os.ModeSymlink != 0 || !current.IsDir() || !os.SameFile(expected, current) {
		return fmt.Errorf("runtime home changed during lifecycle operation: %s", cfg.Agent47Home)
	}
	return nil
}

func sameManagedAndPublishedEntry(cfg runtime.Config) bool {
	return samePath(cfg.OS, managedBinaryPath(cfg), publishedAfsPath(cfg))
}

func isManagedPublishedHelper(cfg runtime.Config, target, command string) bool {
	if cfg.OS != "windows" {
		return symlinkPointsExactlyTo(target, managedBinaryPath(cfg), cfg.OS)
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
		return symlinkPointsExactlyTo(target, managedBinaryPath(cfg), cfg.OS)
	}

	managedData, managedErr := os.ReadFile(managedBinaryPath(cfg))
	targetData, targetErr := os.ReadFile(target)
	if managedErr != nil || targetErr != nil {
		return false
	}
	return bytes.Equal(managedData, targetData)
}

func symlinkPointsExactlyTo(linkPath, expectedTarget, osName string) bool {
	info, err := os.Lstat(linkPath)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	actualTarget, err := os.Readlink(linkPath)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(actualTarget) {
		actualTarget = filepath.Join(filepath.Dir(linkPath), actualTarget)
	}
	return samePath(osName, actualTarget, expectedTarget)
}
