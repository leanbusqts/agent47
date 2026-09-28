package initrepo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var legacyRootTargets = []string{"rules", "skills", "prompts"}

var legacyFileTargets = []struct {
	parent  []string
	name    string
	display string
}{
	{parent: []string{"specs"}, name: "spec.yml", display: "specs/spec.yml"},
	{parent: []string{".agents", "specs"}, name: "spec.yml", display: ".agents/specs/spec.yml"},
}

type legacyCleanup struct {
	rootPath   string
	repository confinedRoot
	rootInfo   fs.FileInfo
	backups    []legacyBackup
	opened     []confinedRoot
}

type legacyBackup struct {
	parent     confinedRoot
	parentPath string
	parentInfo fs.FileInfo
	original   string
	backup     string
	display    string
	digest     string
	info       fs.FileInfo
	linkTarget string
}

func legacyRemovalPlan(rootPath string, selectedRules []string) ([]string, error) {
	selected := make(map[string]bool, len(selectedRules))
	for _, rule := range selectedRules {
		selected[filepath.ToSlash(filepath.Join("rules", rule))] = true
	}

	var removals []string
	rulesPath := filepath.Join(rootPath, "rules")
	if info, err := os.Lstat(rulesPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, errors.New("managed rules path must be a real directory")
		}
		err = filepath.WalkDir(rulesPath, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if path == rulesPath {
				return nil
			}
			rel, relErr := filepath.Rel(rootPath, path)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)
			if selected[rel] && !entry.IsDir() {
				return nil
			}
			if entry.IsDir() {
				entries, readErr := os.ReadDir(path)
				if readErr != nil {
					return readErr
				}
				if len(entries) == 0 {
					removals = append(removals, rel+"/")
				}
				return nil
			}
			removals = append(removals, rel)
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("inspect legacy rules: %w", err)
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}

	for _, name := range legacyRootTargets[1:] {
		if _, err := os.Lstat(filepath.Join(rootPath, name)); err == nil {
			removals = append(removals, name+"/")
		} else if !errors.Is(err, fs.ErrNotExist) {
			return nil, err
		}
	}
	for _, item := range legacyFileTargets {
		exists, err := legacyFileExists(rootPath, item.parent, item.name)
		if err != nil {
			return nil, err
		}
		if exists {
			removals = append(removals, item.display)
		}
	}
	sort.Strings(removals)
	return removals, nil
}

func legacyFileExists(rootPath string, parents []string, name string) (bool, error) {
	current := rootPath
	for _, component := range parents {
		current = filepath.Join(current, component)
		info, err := os.Lstat(current)
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return false, fmt.Errorf("legacy cleanup parent must be a real directory: %s", filepath.ToSlash(strings.TrimPrefix(current, rootPath+string(filepath.Separator))))
		}
	}
	info, err := os.Lstat(filepath.Join(current, name))
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err == nil && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		parts := append(append([]string{}, parents...), name)
		return false, fmt.Errorf("legacy cleanup file target must not be a directory: %s", filepath.ToSlash(filepath.Join(parts...)))
	}
	return err == nil, err
}

func (s *Service) stageLegacyCleanup(ctx context.Context, rootPath string) (*legacyCleanup, error) {
	repository, err := s.rootFactory()(rootPath)
	if err != nil {
		return nil, err
	}
	rootInfo, err := repositoryInfo(repository)
	if err != nil {
		_ = repository.Close()
		return nil, err
	}
	cleanup := &legacyCleanup{rootPath: rootPath, repository: repository, rootInfo: rootInfo}

	fail := func(primary error) (*legacyCleanup, error) {
		rollbackErr := cleanup.rollback()
		if rollbackErr != nil {
			return nil, fmt.Errorf("%w; legacy cleanup rollback incomplete: %v; recovery files retained in %s", primary, rollbackErr, cleanup.recoveryPaths())
		}
		_ = cleanup.close()
		return nil, primary
	}

	for _, name := range legacyRootTargets {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		if err := s.stageLegacyEntry(ctx, cleanup, repository, rootPath, rootInfo, name, name+"/"); err != nil {
			return fail(err)
		}
	}
	for _, item := range legacyFileTargets {
		if err := ctx.Err(); err != nil {
			return fail(err)
		}
		parent, parentPath, parentInfo, exists, err := cleanup.openParent(item.parent)
		if err != nil {
			return fail(err)
		}
		if !exists {
			continue
		}
		if err := s.stageLegacyEntry(ctx, cleanup, parent, parentPath, parentInfo, item.name, item.display); err != nil {
			return fail(err)
		}
	}
	return cleanup, nil
}

func (s *Service) stageLegacyEntry(ctx context.Context, cleanup *legacyCleanup, parent confinedRoot, parentPath string, parentInfo fs.FileInfo, name, display string) error {
	info, err := parent.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect legacy path %s: %w", display, err)
	}
	if name == "spec.yml" && info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
		return fmt.Errorf("legacy cleanup file target must not be a directory: %s", display)
	}
	linkTarget := ""
	if info.Mode()&os.ModeSymlink != 0 {
		linkTarget, err = os.Readlink(filepath.Join(parentPath, name))
		if err != nil {
			return fmt.Errorf("inspect legacy symlink %s: %w", display, err)
		}
	}
	token, err := randomToken()
	if err != nil {
		return err
	}
	backupName := ".agent47-force-" + token + "-" + filepath.Base(name)
	if err := s.mutate(mutationBackup, display); err != nil {
		return err
	}
	if err := cleanup.verifyParent(parentPath, parentInfo); err != nil {
		return err
	}
	current, err := parent.Lstat(name)
	if err != nil || current.Mode() != info.Mode() || info.Mode()&os.ModeSymlink == 0 && !os.SameFile(info, current) {
		return concurrentChangeError(display)
	}
	if err := parent.Rename(name, backupName); err != nil {
		return fmt.Errorf("stage legacy cleanup for %s: %w", display, err)
	}
	cleanup.backups = append(cleanup.backups, legacyBackup{
		parent: parent, parentPath: parentPath, parentInfo: parentInfo,
		original: name, backup: backupName, display: display, info: info, linkTarget: linkTarget,
	})
	backupIndex := len(cleanup.backups) - 1
	if err := parent.Sync(); err != nil {
		return err
	}
	backupInfo, err := parent.Lstat(backupName)
	if err != nil {
		return fmt.Errorf("inspect staged legacy path %s: %w", display, err)
	}
	cleanup.backups[backupIndex].info = backupInfo
	if info.Mode()&os.ModeSymlink != 0 {
		currentTarget, readErr := os.Readlink(filepath.Join(parentPath, backupName))
		if readErr != nil || currentTarget != linkTarget {
			return concurrentChangeError(display)
		}
	}
	digest, err := digestLegacyPath(ctx, filepath.Join(parentPath, backupName))
	if err != nil {
		return fmt.Errorf("fingerprint staged legacy path %s: %w", display, err)
	}
	cleanup.backups[backupIndex].digest = digest
	return nil
}

func (c *legacyCleanup) openParent(components []string) (confinedRoot, string, fs.FileInfo, bool, error) {
	current := c.repository
	currentPath := c.rootPath
	for _, component := range components {
		info, err := current.Lstat(component)
		if errors.Is(err, fs.ErrNotExist) {
			if current != c.repository {
				_ = current.Close()
			}
			return nil, "", nil, false, nil
		}
		if err != nil {
			if current != c.repository {
				_ = current.Close()
			}
			return nil, "", nil, false, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			if current != c.repository {
				_ = current.Close()
			}
			return nil, "", nil, false, fmt.Errorf("legacy cleanup parent must be a real directory: %s", filepath.ToSlash(filepath.Join(components...)))
		}
		next, err := current.OpenRoot(component)
		if err != nil {
			if current != c.repository {
				_ = current.Close()
			}
			return nil, "", nil, false, err
		}
		if current != c.repository {
			_ = current.Close()
		}
		current = next
		currentPath = filepath.Join(currentPath, component)
	}
	info, err := current.Stat(".")
	if err != nil {
		_ = current.Close()
		return nil, "", nil, false, err
	}
	c.opened = append(c.opened, current)
	return current, currentPath, info, true, nil
}

func (c *legacyCleanup) finalize() error {
	for _, backup := range c.backups {
		if err := c.verifyBackup(context.Background(), backup); err != nil {
			return fmt.Errorf("legacy cleanup not finalized: %w; recovery retained at %s", err, filepath.Join(backup.parentPath, backup.backup))
		}
	}
	for _, backup := range c.backups {
		if err := c.verifyBackup(context.Background(), backup); err != nil {
			return fmt.Errorf("legacy cleanup not finalized: %w; recovery retained at %s", err, filepath.Join(backup.parentPath, backup.backup))
		}
		if err := removeLegacyTree(backup.parent, backup.backup, backup.display); err != nil {
			return fmt.Errorf("remove legacy backup for %s: %w", backup.display, err)
		}
		if err := backup.parent.Sync(); err != nil {
			return err
		}
	}
	c.pruneEmptyLegacyParents()
	return nil
}

func removeLegacyTree(parent confinedRoot, name, display string) error {
	info, err := parent.Lstat(name)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return parent.Remove(name)
	}
	child, err := parent.OpenRoot(name)
	if err != nil {
		return fmt.Errorf("open staged directory %s: %w", display, err)
	}
	childInfo, err := child.Stat(".")
	if err != nil || !os.SameFile(info, childInfo) {
		_ = child.Close()
		return concurrentChangeError(display)
	}
	entries, err := child.ReadDir()
	if err != nil {
		_ = child.Close()
		return err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if err := removeLegacyTree(child, entry.Name(), filepath.ToSlash(filepath.Join(display, entry.Name()))); err != nil {
			_ = child.Close()
			return err
		}
	}
	if err := child.Sync(); err != nil {
		_ = child.Close()
		return err
	}
	if err := child.Close(); err != nil {
		return err
	}
	current, err := parent.Lstat(name)
	if err != nil || !os.SameFile(info, current) {
		return concurrentChangeError(display)
	}
	return parent.Remove(name)
}

func (c *legacyCleanup) rollback() error {
	var rollbackErrors []error
	for i := len(c.backups) - 1; i >= 0; i-- {
		backup := c.backups[i]
		if err := c.verifyBackup(context.Background(), backup); err != nil {
			rollbackErrors = append(rollbackErrors, err)
			continue
		}
		if _, err := backup.parent.Lstat(backup.original); err == nil || !errors.Is(err, fs.ErrNotExist) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback conflict for %s: destination exists; backup left at %s", backup.display, filepath.Join(backup.parentPath, backup.backup)))
			continue
		}
		if err := backup.parent.Rename(backup.backup, backup.original); err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("restore %s: %w", backup.display, err))
			continue
		}
		if err := backup.parent.Sync(); err != nil {
			rollbackErrors = append(rollbackErrors, err)
		}
	}
	return errors.Join(rollbackErrors...)
}

func (c *legacyCleanup) verifyBackup(ctx context.Context, backup legacyBackup) error {
	if err := c.verifyParent(backup.parentPath, backup.parentInfo); err != nil {
		return err
	}
	current, err := backup.parent.Lstat(backup.backup)
	if err != nil {
		return fmt.Errorf("legacy backup missing for %s", backup.display)
	}
	if backup.info.Mode()&os.ModeSymlink == 0 && !os.SameFile(backup.info, current) {
		return fmt.Errorf("legacy backup changed identity for %s", backup.display)
	}
	if backup.info.Mode()&os.ModeSymlink != 0 {
		target, readErr := os.Readlink(filepath.Join(backup.parentPath, backup.backup))
		if readErr != nil || target != backup.linkTarget {
			return fmt.Errorf("legacy backup changed for %s", backup.display)
		}
	}
	if backup.digest == "" {
		return nil
	}
	digest, err := digestLegacyPath(ctx, filepath.Join(backup.parentPath, backup.backup))
	if err != nil || digest != backup.digest {
		return fmt.Errorf("legacy backup changed for %s", backup.display)
	}
	return nil
}

func (c *legacyCleanup) verifyParent(parentPath string, parentInfo fs.FileInfo) error {
	rootNow, err := os.Lstat(c.rootPath)
	if err != nil || rootNow.Mode()&os.ModeSymlink != 0 || !rootNow.IsDir() || !os.SameFile(c.rootInfo, rootNow) {
		return errors.New("repository root changed during legacy cleanup")
	}
	parentNow, err := os.Lstat(parentPath)
	if err != nil || parentNow.Mode()&os.ModeSymlink != 0 || !parentNow.IsDir() || !os.SameFile(parentInfo, parentNow) {
		return fmt.Errorf("legacy cleanup parent changed: %s", parentPath)
	}
	return nil
}

func (c *legacyCleanup) pruneEmptyLegacyParents() {
	for _, rel := range []string{".agents/specs", ".agents", "specs"} {
		path := filepath.Join(c.rootPath, filepath.FromSlash(rel))
		info, err := os.Lstat(path)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			continue
		}
		_ = os.Remove(path)
	}
}

func (c *legacyCleanup) recoveryPaths() string {
	paths := make([]string, 0, len(c.backups))
	for _, backup := range c.backups {
		paths = append(paths, filepath.Join(backup.parentPath, backup.backup))
	}
	sort.Strings(paths)
	return strings.Join(paths, ", ")
}

func (c *legacyCleanup) close() error {
	var closeErrors []error
	for _, root := range c.opened {
		if err := root.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	if c.repository != nil {
		if err := c.repository.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}

func digestLegacyPath(ctx context.Context, root string) (string, error) {
	hash := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := os.Lstat(path)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(hash, "%s\x00%s\x00", filepath.ToSlash(rel), info.Mode().String()); err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			target, err := os.Readlink(path)
			if err != nil {
				return err
			}
			_, err = io.WriteString(hash, target)
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(hash, file)
		after, statErr := file.Stat()
		closeErr := file.Close()
		pathAfter, pathErr := os.Lstat(path)
		if copyErr != nil || statErr != nil || closeErr != nil || pathErr != nil {
			return errors.Join(copyErr, statErr, closeErr, pathErr)
		}
		if !os.SameFile(info, after) || !os.SameFile(after, pathAfter) || info.Size() != pathAfter.Size() || !info.ModTime().Equal(pathAfter.ModTime()) {
			return concurrentChangeError(filepath.ToSlash(path))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
