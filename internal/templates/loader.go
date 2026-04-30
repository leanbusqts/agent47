package templates

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/leanbusqts/agent47/internal/runtime"
)

type Loader struct {
	Source    Source
	RawSource Source
}

func NewLoader(mode runtime.TemplateMode, repoRoot string) (*Loader, error) {
	switch mode {
	case runtime.TemplateModeEmbedded:
		rawSource, err := NewEmbeddedSource()
		if err != nil {
			return nil, err
		}
		baseSource, err := NewEmbeddedSourceAt("templates/base")
		if err != nil {
			return nil, err
		}
		if hasBaseLayout(baseSource) {
			source, err := newCatalogSource(baseSource, rawSource)
			if err != nil {
				return nil, err
			}
			return &Loader{
				Source:    source,
				RawSource: rawSource,
			}, nil
		}
		return &Loader{Source: rawSource, RawSource: rawSource}, nil
	case runtime.TemplateModeFilesystem:
		if repoRoot == "" {
			return nil, fmt.Errorf("filesystem template mode requires repo root")
		}
		rawSource := NewFilesystemSource(filepath.Join(repoRoot, "templates"))
		baseSource := NewFilesystemSource(filepath.Join(repoRoot, "templates", "base"))
		if hasBaseLayout(baseSource) {
			source, err := newCatalogSource(baseSource, rawSource)
			if err != nil {
				return nil, err
			}
			return &Loader{
				Source:    source,
				RawSource: rawSource,
			}, nil
		}
		return &Loader{Source: rawSource, RawSource: rawSource}, nil
	default:
		return nil, fmt.Errorf("unknown template mode: %q", mode)
	}
}

func (l *Loader) BundleSource(bundleIDs []string) Source {
	if l == nil || l.RawSource == nil {
		return nil
	}
	return AssembleSource(l.RawSource, bundleIDs)
}

func newCatalogSource(baseSource, rawSource Source) (Source, error) {
	bundleIDs, err := DiscoverBundleIDs(rawSource)
	if err != nil {
		return nil, err
	}
	return NewOverlaySource(baseSource, AssembleSource(rawSource, bundleIDs)), nil
}

func DiscoverBundleIDs(rawSource Source) ([]string, error) {
	entries, err := rawSource.ReadDir("bundles")
	if err != nil {
		if isNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var bundleIDs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		bundleIDs = append(bundleIDs, entry.Name())
	}
	sort.Strings(bundleIDs)
	return bundleIDs, nil
}
