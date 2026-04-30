package templates

import (
	"io/fs"
	"path"

	agent47embed "github.com/leanbusqts/agent47"
)

type EmbeddedSource struct {
	root string
}

func NewEmbeddedSource() (*EmbeddedSource, error) {
	return NewEmbeddedSourceAt("templates")
}

func NewEmbeddedSourceAt(root string) (*EmbeddedSource, error) {
	return &EmbeddedSource{root: clean(root)}, nil
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
		return s.root
	}
	return path.Join(s.root, cleaned)
}

func clean(value string) string {
	if value == "" || value == "." {
		return "."
	}

	return path.Clean(value)
}
