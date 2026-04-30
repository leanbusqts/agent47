package templates

import (
	"io/fs"
	"path/filepath"
	"sort"
)

type PrefixedSource struct {
	base   Source
	prefix string
}

func NewPrefixedSource(base Source, prefix string) *PrefixedSource {
	return &PrefixedSource{base: base, prefix: filepath.ToSlash(prefix)}
}

func (s *PrefixedSource) ReadFile(path string) ([]byte, error) {
	return s.base.ReadFile(s.resolve(path))
}

func (s *PrefixedSource) ReadDir(path string) ([]fs.DirEntry, error) {
	return s.base.ReadDir(s.resolve(path))
}

func (s *PrefixedSource) Stat(path string) (fs.FileInfo, error) {
	return s.base.Stat(s.resolve(path))
}

func (s *PrefixedSource) resolve(path string) string {
	if path == "" || path == "." {
		return s.prefix
	}
	return filepath.ToSlash(filepath.Join(s.prefix, filepath.FromSlash(path)))
}

type UnionSource struct {
	sources    []Source
	hideAtRoot map[string]bool
}

func NewUnionSource(hideAtRoot []string, sources ...Source) *UnionSource {
	hidden := make(map[string]bool, len(hideAtRoot))
	for _, name := range hideAtRoot {
		hidden[name] = true
	}
	return &UnionSource{sources: sources, hideAtRoot: hidden}
}

func (s *UnionSource) ReadFile(path string) ([]byte, error) {
	if path == "manifest.txt" {
		return nil, fs.ErrNotExist
	}
	var lastErr error
	for _, source := range s.sources {
		data, err := source.ReadFile(path)
		if err == nil {
			return data, nil
		}
		if !isNotExist(err) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

func (s *UnionSource) ReadDir(path string) ([]fs.DirEntry, error) {
	merged := make(map[string]fs.DirEntry)
	var lastErr error

	for _, source := range s.sources {
		entries, err := source.ReadDir(path)
		if err != nil {
			if isNotExist(err) {
				lastErr = err
				continue
			}
			return nil, err
		}
		for _, entry := range entries {
			if path == "." && s.hideAtRoot[entry.Name()] {
				continue
			}
			if path == "." && entry.Name() == "manifest.txt" {
				continue
			}
			if _, ok := merged[entry.Name()]; ok {
				continue
			}
			merged[entry.Name()] = entry
		}
	}

	if len(merged) == 0 && lastErr != nil {
		return nil, lastErr
	}

	names := make([]string, 0, len(merged))
	for name := range merged {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]fs.DirEntry, 0, len(names))
	for _, name := range names {
		result = append(result, merged[name])
	}
	return result, nil
}

func (s *UnionSource) Stat(path string) (fs.FileInfo, error) {
	if path == "manifest.txt" {
		return nil, fs.ErrNotExist
	}
	var lastErr error
	for _, source := range s.sources {
		info, err := source.Stat(path)
		if err == nil {
			return info, nil
		}
		if !isNotExist(err) {
			return nil, err
		}
		lastErr = err
	}
	return nil, lastErr
}

func AssembleSource(raw Source, bundleIDs []string) Source {
	sources := make([]Source, 0, len(bundleIDs)+1)
	for _, bundleID := range bundleIDs {
		if bundleID == "" || bundleID == "base" {
			continue
		}
		sources = append(sources, NewPrefixedSource(raw, filepath.ToSlash(filepath.Join("bundles", bundleID))))
	}
	sources = append(sources, NewPrefixedSource(raw, "base"))
	return NewUnionSource([]string{"base", "bundles"}, sources...)
}
