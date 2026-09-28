package initrepo

import (
	"io/fs"
	"os"
)

type confinedRoot interface {
	Close() error
	Link(oldName, newName string) error
	Lstat(name string) (fs.FileInfo, error)
	Mkdir(name string, perm fs.FileMode) error
	Open(name string) (*os.File, error)
	OpenFile(name string, flag int, perm fs.FileMode) (*os.File, error)
	OpenRoot(name string) (confinedRoot, error)
	ReadDir() ([]fs.DirEntry, error)
	Remove(name string) error
	Rename(oldName, newName string) error
	Stat(name string) (fs.FileInfo, error)
	Sync() error
}
