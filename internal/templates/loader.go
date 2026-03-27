package templates

import (
	"fmt"
	"path/filepath"

	"github.com/leanbusqts/agent47/internal/runtime"
)

type Loader struct {
	Source Source
}

func NewLoader(mode runtime.TemplateMode, repoRoot string) (*Loader, error) {
	switch mode {
	case runtime.TemplateModeEmbedded:
		source, err := NewEmbeddedSource()
		if err != nil {
			return nil, err
		}
		return &Loader{Source: source}, nil
	case runtime.TemplateModeFilesystem:
		if repoRoot == "" {
			return nil, fmt.Errorf("filesystem template mode requires repo root")
		}
		return &Loader{Source: NewFilesystemSource(filepath.Join(repoRoot, "templates"))}, nil
	default:
		return nil, fmt.Errorf("unknown template mode: %q", mode)
	}
}
