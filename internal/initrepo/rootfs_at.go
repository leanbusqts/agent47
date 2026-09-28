//go:build darwin || linux

package initrepo

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

type atRoot struct {
	dir *os.File
}

func openConfinedRoot(path string) (confinedRoot, error) {
	dir, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	info, err := dir.Stat()
	if err != nil {
		dir.Close()
		return nil, err
	}
	if !info.IsDir() {
		dir.Close()
		return nil, errors.New("confined root must be a directory")
	}
	return &atRoot{dir: dir}, nil
}

func (r *atRoot) fd() int { return int(r.dir.Fd()) }

func (r *atRoot) Close() error { return r.dir.Close() }

func (r *atRoot) Link(oldName, newName string) error {
	if err := validateRelativeName(oldName); err != nil {
		return err
	}
	if err := validateRelativeName(newName); err != nil {
		return err
	}
	return linkat(r.fd(), oldName, r.fd(), newName)
}

func (r *atRoot) Lstat(name string) (fs.FileInfo, error) {
	if name == "." {
		return r.dir.Stat()
	}
	if err := validateRelativeName(name); err != nil {
		return nil, err
	}
	fd, err := openat(r.fd(), name, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_NONBLOCK, 0)
	if errors.Is(err, syscall.ELOOP) {
		return symlinkFileInfo{name: filepath.Base(name)}, nil
	}
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("open relative file returned invalid descriptor")
	}
	info, statErr := file.Stat()
	closeErr := file.Close()
	if statErr != nil {
		return nil, statErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return info, nil
}

func (r *atRoot) Mkdir(name string, perm fs.FileMode) error {
	if err := validateRelativeName(name); err != nil {
		return err
	}
	return mkdirat(r.fd(), name, uint32(perm.Perm()))
}

func (r *atRoot) Open(name string) (*os.File, error) {
	return r.OpenFile(name, os.O_RDONLY, 0)
}

func (r *atRoot) OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error) {
	if err := validateRelativeName(name); err != nil {
		return nil, err
	}
	fd, err := openat(r.fd(), name, flag|syscall.O_CLOEXEC|syscall.O_NOFOLLOW, uint32(perm.Perm()))
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("open relative file returned invalid descriptor")
	}
	return file, nil
}

func (r *atRoot) OpenRoot(name string) (confinedRoot, error) {
	if err := validateRelativeName(name); err != nil {
		return nil, err
	}
	fd, err := openat(r.fd(), name, syscall.O_RDONLY|syscall.O_CLOEXEC|syscall.O_NOFOLLOW|syscall.O_DIRECTORY, 0)
	if err != nil {
		return nil, err
	}
	file := os.NewFile(uintptr(fd), name)
	if file == nil {
		_ = syscall.Close(fd)
		return nil, errors.New("open relative directory returned invalid descriptor")
	}
	return &atRoot{dir: file}, nil
}

func (r *atRoot) ReadDir() ([]fs.DirEntry, error) { return r.dir.ReadDir(-1) }

func (r *atRoot) Remove(name string) error {
	info, err := r.Lstat(name)
	if err != nil {
		return err
	}
	flags := 0
	if info.IsDir() {
		flags = atRemoveDir
	}
	return unlinkat(r.fd(), name, flags)
}

func (r *atRoot) Rename(oldName, newName string) error {
	if err := validateRelativeName(oldName); err != nil {
		return err
	}
	if err := validateRelativeName(newName); err != nil {
		return err
	}
	return renameat(r.fd(), oldName, r.fd(), newName)
}

func (r *atRoot) Stat(name string) (fs.FileInfo, error) { return r.Lstat(name) }

func (r *atRoot) Sync() error { return r.dir.Sync() }

type symlinkFileInfo struct {
	name string
}

func (i symlinkFileInfo) Name() string     { return i.name }
func (symlinkFileInfo) Size() int64        { return 0 }
func (symlinkFileInfo) Mode() fs.FileMode  { return os.ModeSymlink }
func (symlinkFileInfo) ModTime() time.Time { return time.Time{} }
func (symlinkFileInfo) IsDir() bool        { return false }
func (symlinkFileInfo) Sys() any           { return nil }

func validateRelativeName(name string) error {
	if name == "" || name == "." || filepath.Base(name) != name || filepath.IsAbs(name) {
		return errors.New("confined filesystem operation requires one relative path component")
	}
	return nil
}
