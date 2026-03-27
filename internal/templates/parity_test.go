package templates

import (
	"bytes"
	"path/filepath"
	"sort"
	"testing"

	"github.com/leanbusqts/agent47/internal/runtime"
)

func TestEmbeddedAndFilesystemTemplatesStayInParity(t *testing.T) {
	t.Parallel()

	fsLoader, err := NewLoader(runtime.TemplateModeFilesystem, repoRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	embeddedLoader, err := NewLoader(runtime.TemplateModeEmbedded, "")
	if err != nil {
		t.Fatal(err)
	}

	filesystemPaths, err := collectTemplatePaths(fsLoader.Source, ".")
	if err != nil {
		t.Fatal(err)
	}
	embeddedPaths, err := collectTemplatePaths(embeddedLoader.Source, ".")
	if err != nil {
		t.Fatal(err)
	}

	if !equalStrings(filesystemPaths, embeddedPaths) {
		t.Fatalf("template path mismatch\nfilesystem=%v\nembedded=%v", filesystemPaths, embeddedPaths)
	}

	for _, rel := range filesystemPaths {
		fsInfo, err := fsLoader.Source.Stat(rel)
		if err != nil {
			t.Fatalf("filesystem stat failed for %s: %v", rel, err)
		}
		embeddedInfo, err := embeddedLoader.Source.Stat(rel)
		if err != nil {
			t.Fatalf("embedded stat failed for %s: %v", rel, err)
		}
		if fsInfo.IsDir() != embeddedInfo.IsDir() {
			t.Fatalf("dir/file kind mismatch for %s", rel)
		}
		if fsInfo.IsDir() {
			continue
		}

		fsData, err := fsLoader.Source.ReadFile(rel)
		if err != nil {
			t.Fatalf("filesystem read failed for %s: %v", rel, err)
		}
		embeddedData, err := embeddedLoader.Source.ReadFile(rel)
		if err != nil {
			t.Fatalf("embedded read failed for %s: %v", rel, err)
		}
		if !bytes.Equal(fsData, embeddedData) {
			t.Fatalf("template content mismatch for %s", rel)
		}
	}
}

func collectTemplatePaths(src Source, root string) ([]string, error) {
	entries, err := src.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var out []string
	for _, entry := range entries {
		rel := filepath.ToSlash(filepath.Join(root, entry.Name()))
		if root == "." {
			rel = entry.Name()
		}
		out = append(out, rel)
		if !entry.IsDir() {
			continue
		}
		children, err := collectTemplatePaths(src, rel)
		if err != nil {
			return nil, err
		}
		out = append(out, children...)
	}

	sort.Strings(out)
	return out, nil
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for idx := range left {
		if left[idx] != right[idx] {
			return false
		}
	}
	return true
}
