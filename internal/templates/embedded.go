package templates

import (
	"io/fs"
	"path"

	agent47embed "github.com/leanbusqts/agent47"
)

type EmbeddedSource struct {
}

func NewEmbeddedSource() (*EmbeddedSource, error) {
	return &EmbeddedSource{}, nil
}

func (s *EmbeddedSource) ReadFile(filePath string) ([]byte, error) {
	return fs.ReadFile(agent47embed.TemplatesFS, s.resolve(filePath))
}

func (s *EmbeddedSource) ReadDir(dirPath string) ([]fs.DirEntry, error) {
	return fs.ReadDir(agent47embed.TemplatesFS, s.resolve(dirPath))
}

func (s *EmbeddedSource) Stat(targetPath string) (fs.FileInfo, error) {
	return fs.Stat(agent47embed.TemplatesFS, s.resolve(targetPath))
}

func (s *EmbeddedSource) resolve(value string) string {
	cleaned := clean(value)
	if cleaned == "." {
		return "templates"
	}
	return path.Join("templates", cleaned)
}

func clean(value string) string {
	if value == "" || value == "." {
		return "."
	}

	return path.Clean(value)
}
