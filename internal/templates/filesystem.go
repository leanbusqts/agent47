package templates

import (
	"io/fs"
	"os"
	"path/filepath"
)

type FilesystemSource struct {
	root string
}

func NewFilesystemSource(root string) *FilesystemSource {
	return &FilesystemSource{root: root}
}

func (s *FilesystemSource) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(s.resolve(path))
}

func (s *FilesystemSource) ReadDir(path string) ([]fs.DirEntry, error) {
	return os.ReadDir(s.resolve(path))
}

func (s *FilesystemSource) Stat(path string) (fs.FileInfo, error) {
	return os.Stat(s.resolve(path))
}

func (s *FilesystemSource) resolve(path string) string {
	if path == "" || path == "." {
		return s.root
	}

	return filepath.Join(s.root, filepath.FromSlash(path))
}
