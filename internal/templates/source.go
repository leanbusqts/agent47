package templates

import (
	"io/fs"
)

type Source interface {
	ReadFile(path string) ([]byte, error)
	ReadDir(path string) ([]fs.DirEntry, error)
	Stat(path string) (fs.FileInfo, error)
}

type LoaderSource interface {
	Source
}
