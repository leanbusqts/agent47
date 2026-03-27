package fsx

import (
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
	return os.Remove(path)
}

func (Service) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (Service) Rename(oldPath, newPath string) error {
	return os.Rename(oldPath, newPath)
}

func (Service) CopyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return fmt.Errorf("copy file source is directory: %s", src)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
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

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if err := out.Chmod(info.Mode()); err != nil {
		return err
	}

	return out.Close()
}

func (s Service) CopyDir(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("copy dir source is not directory: %s", src)
	}

	if err := os.MkdirAll(dst, info.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := s.CopyDir(srcPath, dstPath); err != nil {
				return err
			}
			continue
		}

		if err := s.CopyFile(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

func (Service) SymlinkAtomic(targetPath, linkPath string) error {
	linkDir := filepath.Dir(linkPath)
	linkName := filepath.Base(linkPath)

	if err := os.MkdirAll(linkDir, 0o755); err != nil {
		return err
	}

	tmpLink := filepath.Join(linkDir, "."+linkName+".tmp")
	_ = os.Remove(tmpLink)
	if err := os.Symlink(targetPath, tmpLink); err != nil {
		return err
	}
	defer os.Remove(tmpLink)

	if testHookFail("AGENT47_FAIL_SYMLINK_TARGET", linkPath) {
		return fmt.Errorf("injected symlink swap failure for %s", linkPath)
	}

	return replaceAtomicPath(tmpLink, linkPath)
}

func (Service) ReplaceDirAtomic(stageDir, targetDir string, force bool) (ReplaceDirResult, error) {
	var result ReplaceDirResult

	if info, err := os.Stat(targetDir); err == nil && info.IsDir() {
		if !force {
			return result, fmt.Errorf("target directory already exists: %s", targetDir)
		}

		parentDir := filepath.Dir(targetDir)
		targetName := filepath.Base(targetDir)
		matches, _ := filepath.Glob(filepath.Join(parentDir, targetName+".bak.*"))
		for _, match := range matches {
			_ = os.RemoveAll(match)
		}

		result.BackupPath = filepath.Join(parentDir, fmt.Sprintf("%s.bak.%s", targetName, time.Now().Format("20060102150405")))
		if err := os.Rename(targetDir, result.BackupPath); err != nil {
			return ReplaceDirResult{}, err
		}
	}

	if testHookFailDirSwap(targetDir) {
		if result.BackupPath != "" && !dirExists(targetDir) && dirExists(result.BackupPath) {
			_ = os.Rename(result.BackupPath, targetDir)
		}
		return ReplaceDirResult{}, fmt.Errorf("injected dir swap failure for %s", targetDir)
	}

	if err := os.Rename(stageDir, targetDir); err != nil {
		if result.BackupPath != "" && !dirExists(targetDir) && dirExists(result.BackupPath) {
			_ = os.Rename(result.BackupPath, targetDir)
		}
		return ReplaceDirResult{}, err
	}

	return result, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func replaceAtomicPath(srcPath, dstPath string) error {
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
			_ = os.Rename(backupPath, dstPath)
		}
		return err
	}

	if backupPath != "" {
		_ = os.Remove(backupPath)
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
