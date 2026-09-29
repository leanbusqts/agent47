package fsx

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	goRuntime "runtime"
	"time"
)

type Service struct{}

type ReplaceDirResult struct {
	BackupPath string
}

func (Service) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (Service) WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if testHookFail("AGENT47_FAIL_WRITE_TARGET", path) {
		return fmt.Errorf("injected write failure for %s", path)
	}

	tmpFile, err := os.CreateTemp(dir, ".agent47-tmp-*")
	if err != nil {
		return err
	}

	tmpPath := tmpFile.Name()
	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Chmod(perm); err != nil {
		_ = tmpFile.Close()
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}

	return replaceAtomicPath(tmpPath, path)
}

func (Service) MkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}

func (Service) Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (Service) IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (Service) Remove(path string) error {
	if testHookFail("AGENT47_FAIL_REMOVE_TARGET", path) {
		return fmt.Errorf("injected remove failure for %s", path)
	}
	return os.Remove(path)
}

func (Service) RemoveAll(path string) error {
	if testHookFail("AGENT47_FAIL_REMOVE_TARGET", path) {
		return fmt.Errorf("injected remove failure for %s", path)
	}
	return os.RemoveAll(path)
}

func (Service) CopyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("copy file source is directory: %s", src)
	}

	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if testHookFail("AGENT47_FAIL_COPY_TARGET", dst) {
		return fmt.Errorf("injected copy failure for %s", dst)
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.CreateTemp(dir, ".agent47-copy-*")
	if err != nil {
		return err
	}
	tmpPath := out.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Chmod(info.Mode()); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Close(); err != nil {
		return err
	}

	return replaceAtomicPath(tmpPath, dst)
}

func (Service) SymlinkAtomic(targetPath, linkPath string) error {
	linkDir := filepath.Dir(linkPath)
	linkName := filepath.Base(linkPath)

	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		return err
	}

	tmpFile, err := os.CreateTemp(linkDir, "."+linkName+".tmp-*")
	if err != nil {
		return err
	}
	tmpLink := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		_ = os.Remove(tmpLink)
		return err
	}
	if err := os.Remove(tmpLink); err != nil {
		return err
	}
	if err := os.Symlink(targetPath, tmpLink); err != nil {
		return err
	}
	defer func() { _ = os.Remove(tmpLink) }()

	if testHookFail("AGENT47_FAIL_SYMLINK_TARGET", linkPath) {
		return fmt.Errorf("injected symlink swap failure for %s", linkPath)
	}

	return replaceAtomicPath(tmpLink, linkPath)
}

func (Service) ReplaceDirAtomic(stageDir, targetDir string, force bool) (ReplaceDirResult, error) {
	var result ReplaceDirResult

	if info, err := os.Lstat(targetDir); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return result, fmt.Errorf("target directory must not be a symlink: %s", targetDir)
		}
		if !info.IsDir() {
			return result, fmt.Errorf("target directory path is not a directory: %s", targetDir)
		}
		if !force {
			return result, fmt.Errorf("target directory already exists: %s", targetDir)
		}

		parentDir := filepath.Dir(targetDir)
		targetName := filepath.Base(targetDir)
		result.BackupPath = filepath.Join(parentDir, fmt.Sprintf("%s.bak.%d", targetName, time.Now().UnixNano()))
		if err := os.Rename(targetDir, result.BackupPath); err != nil {
			return ReplaceDirResult{}, err
		}
	} else if !os.IsNotExist(err) {
		return result, err
	}

	if testHookFailDirSwap(targetDir) {
		injectedErr := fmt.Errorf("injected dir swap failure for %s", targetDir)
		if rollbackErr := restoreDirectoryBackup(result.BackupPath, targetDir); rollbackErr != nil {
			return ReplaceDirResult{}, errors.Join(injectedErr, fmt.Errorf("restore template backup: %w", rollbackErr))
		}
		return ReplaceDirResult{}, injectedErr
	}

	if err := os.Rename(stageDir, targetDir); err != nil {
		if rollbackErr := restoreDirectoryBackup(result.BackupPath, targetDir); rollbackErr != nil {
			return ReplaceDirResult{}, errors.Join(err, fmt.Errorf("restore template backup: %w", rollbackErr))
		}
		return ReplaceDirResult{}, err
	}

	return result, nil
}

func restoreDirectoryBackup(backupPath, targetPath string) error {
	if backupPath == "" {
		return nil
	}
	if _, err := os.Lstat(targetPath); err == nil {
		return fmt.Errorf("refusing to overwrite target while restoring backup: %s", targetPath)
	} else if !os.IsNotExist(err) {
		return err
	}
	if _, err := os.Lstat(backupPath); err != nil {
		return err
	}
	return os.Rename(backupPath, targetPath)
}

func replaceAtomicPath(srcPath, dstPath string) error {
	if info, err := os.Lstat(dstPath); err == nil {
		if info.IsDir() && info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("destination path is a directory: %s", dstPath)
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if goRuntime.GOOS != "windows" {
		return os.Rename(srcPath, dstPath)
	}

	backupPath := ""
	if _, err := os.Lstat(dstPath); err == nil {
		backupPath = filepath.Join(filepath.Dir(dstPath), fmt.Sprintf(".%s.swap-%d", filepath.Base(dstPath), time.Now().UnixNano()))
		_ = os.Remove(backupPath)
		if err := os.Rename(dstPath, backupPath); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.Rename(srcPath, dstPath); err != nil {
		if backupPath != "" {
			if rollbackErr := os.Rename(backupPath, dstPath); rollbackErr != nil {
				return errors.Join(err, fmt.Errorf("restore replaced path: %w", rollbackErr))
			}
		}
		return err
	}

	if backupPath != "" {
		if err := os.Remove(backupPath); err != nil {
			return err
		}
	}
	return nil
}

func testHookFail(name, target string) bool {
	if os.Getenv("AGENT47_ENABLE_TEST_HOOKS") != "true" {
		return false
	}
	return os.Getenv(name) == target
}

func testHookFailDirSwap(target string) bool {
	if !testHookFail("AGENT47_FAIL_DIR_SWAP_TARGET", target) {
		return false
	}

	marker := os.Getenv("AGENT47_FAIL_DIR_SWAP_MARKER")
	if marker == "" {
		return true
	}
	if _, err := os.Stat(marker); err == nil {
		return false
	}
	if err := os.WriteFile(marker, []byte{}, 0o644); err != nil {
		return true
	}
	return true
}
