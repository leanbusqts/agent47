//go:build !darwin && !linux

package initrepo

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type pathConfinedRoot struct {
	path string
	info fs.FileInfo
}

func openConfinedRoot(path string) (confinedRoot, error) {
	canonical, err := filepath.EvalSymlinks(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(canonical)
	if err != nil {
		return nil, err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return nil, errors.New("confined root must be a real directory")
	}
	return &pathConfinedRoot{path: canonical, info: info}, nil
}

func (r *pathConfinedRoot) resolve(name string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash(name))
	if cleaned == ".." || filepath.IsAbs(cleaned) || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path escapes confined root: %s", name)
	}
	current, err := os.Lstat(r.path)
	if err != nil || current.Mode()&os.ModeSymlink != 0 || !os.SameFile(r.info, current) {
		return "", errors.New("confined root changed during init")
	}
	return filepath.Join(r.path, cleaned), nil
}

func (r *pathConfinedRoot) Close() error { return nil }

func (r *pathConfinedRoot) Link(oldName, newName string) error {
	oldPath, err := r.resolve(oldName)
	if err != nil {
		return err
	}
	newPath, err := r.resolve(newName)
	if err != nil {
		return err
	}
	return os.Link(oldPath, newPath)
}

func (r *pathConfinedRoot) Lstat(name string) (fs.FileInfo, error) {
	path, err := r.resolve(name)
	if err != nil {
		return nil, err
	}
	return os.Lstat(path)
}

func (r *pathConfinedRoot) Mkdir(name string, perm fs.FileMode) error {
	path, err := r.resolve(name)
	if err != nil {
		return err
	}
	return os.Mkdir(path, perm)
}

func (r *pathConfinedRoot) Open(name string) (*os.File, error) {
	path, err := r.resolve(name)
	if err != nil {
		return nil, err
	}
	return os.Open(path)
}

func (r *pathConfinedRoot) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	path, err := r.resolve(name)
	if err != nil {
		return nil, err
	}
	return os.OpenFile(path, flag, perm)
}

func (r *pathConfinedRoot) OpenRoot(name string) (confinedRoot, error) {
	path, err := r.resolve(name)
	if err != nil {
		return nil, err
	}
	return openConfinedRoot(path)
}

func (r *pathConfinedRoot) ReadDir() ([]fs.DirEntry, error) {
	if _, err := r.resolve("."); err != nil {
		return nil, err
	}
	return os.ReadDir(r.path)
}

func (r *pathConfinedRoot) Remove(name string) error {
	path, err := r.resolve(name)
	if err != nil {
		return err
	}
	return os.Remove(path)
}

func (r *pathConfinedRoot) Rename(oldName, newName string) error {
	oldPath, err := r.resolve(oldName)
	if err != nil {
		return err
	}
	newPath, err := r.resolve(newName)
	if err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}

func (r *pathConfinedRoot) Stat(name string) (fs.FileInfo, error) {
	path, err := r.resolve(name)
	if err != nil {
		return nil, err
	}
	return os.Stat(path)
}

func (r *pathConfinedRoot) Sync() error {
	if runtime.GOOS == "windows" {
		return nil
	}
	dir, err := os.Open(r.path)
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}
