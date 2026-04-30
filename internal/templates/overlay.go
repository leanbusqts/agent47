package templates

import (
	"errors"
	"io/fs"
	"path/filepath"
	"sort"
)

type OverlaySource struct {
	primary  Source
	fallback Source
}

func NewOverlaySource(primary, fallback Source) *OverlaySource {
	return &OverlaySource{primary: primary, fallback: fallback}
}

func (s *OverlaySource) ReadFile(path string) ([]byte, error) {
	data, err := s.primary.ReadFile(path)
	if err == nil {
		return data, nil
	}
	if !isNotExist(err) {
		return nil, err
	}
	return s.fallback.ReadFile(path)
}

func (s *OverlaySource) ReadDir(path string) ([]fs.DirEntry, error) {
	merged := make(map[string]fs.DirEntry)

	primaryEntries, primaryErr := s.primary.ReadDir(path)
	if primaryErr == nil {
		for _, entry := range primaryEntries {
			merged[entry.Name()] = entry
		}
	} else if !isNotExist(primaryErr) {
		return nil, primaryErr
	}

	fallbackEntries, fallbackErr := s.fallback.ReadDir(path)
	if fallbackErr != nil && !isNotExist(fallbackErr) {
		return nil, fallbackErr
	}
	for _, entry := range fallbackEntries {
		if path == "." && entry.Name() == "base" {
			continue
		}
		if _, ok := merged[entry.Name()]; ok {
			continue
		}
		merged[entry.Name()] = entry
	}

	if len(merged) == 0 && (primaryErr != nil || fallbackErr != nil) {
		if primaryErr != nil && !isNotExist(primaryErr) {
			return nil, primaryErr
		}
		return nil, fallbackErr
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

func (s *OverlaySource) Stat(path string) (fs.FileInfo, error) {
	info, err := s.primary.Stat(path)
	if err == nil {
		return info, nil
	}
	if !isNotExist(err) {
		return nil, err
	}
	return s.fallback.Stat(path)
}

func isNotExist(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, fs.ErrNotExist)
}

func hasBaseLayout(src Source) bool {
	info, err := src.Stat(".")
	if err != nil || !info.IsDir() {
		return false
	}
	_, err = src.Stat("manifest.txt")
	return err == nil
}

type RootFilteredSource struct {
	base   Source
	hidden map[string]bool
}

func NewRootFilteredSource(base Source, hiddenAtRoot ...string) *RootFilteredSource {
	hidden := make(map[string]bool, len(hiddenAtRoot))
	for _, name := range hiddenAtRoot {
		hidden[name] = true
	}
	return &RootFilteredSource{base: base, hidden: hidden}
}

func (s *RootFilteredSource) ReadFile(path string) ([]byte, error) {
	if s.isHiddenRootPath(path) {
		return nil, fs.ErrNotExist
	}
	return s.base.ReadFile(path)
}

func (s *RootFilteredSource) ReadDir(path string) ([]fs.DirEntry, error) {
	entries, err := s.base.ReadDir(path)
	if err != nil {
		return nil, err
	}
	if path != "." {
		return entries, nil
	}

	filtered := make([]fs.DirEntry, 0, len(entries))
	for _, entry := range entries {
		if s.hidden[entry.Name()] {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered, nil
}

func (s *RootFilteredSource) Stat(path string) (fs.FileInfo, error) {
	if s.isHiddenRootPath(path) {
		return nil, fs.ErrNotExist
	}
	return s.base.Stat(path)
}

func (s *RootFilteredSource) isHiddenRootPath(path string) bool {
	if path == "" || path == "." {
		return false
	}
	if filepath.Dir(filepath.ToSlash(path)) != "." {
		return false
	}
	return s.hidden[filepath.Base(filepath.ToSlash(path))]
}
