package templates

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"

	"github.com/leanbusqts/agent47/internal/manifest"
)

func AssembleManifest(src Source, bundles []string) (manifest.Manifest, error) {
	data, err := src.ReadFile("manifest.txt")
	if err != nil {
		return manifest.Manifest{}, err
	}

	assembled, err := manifest.Parse(data)
	if err != nil {
		return manifest.Manifest{}, err
	}

	for _, bundleID := range bundles {
		if bundleID == "" || bundleID == "base" {
			continue
		}
		bundleManifestPath := filepath.ToSlash(filepath.Join("bundles", bundleID, "manifest.txt"))
		bundleData, err := src.ReadFile(bundleManifestPath)
		if err != nil {
			if isNotExist(err) {
				return manifest.Manifest{}, MissingBundleManifestError{Path: bundleManifestPath}
			}
			return manifest.Manifest{}, err
		}

		bundleManifest, err := manifest.ParsePartial(bundleData)
		if err != nil {
			return manifest.Manifest{}, InvalidBundleManifestError{
				Path:   bundleManifestPath,
				Detail: err.Error(),
			}
		}
		if isEmptyManifest(bundleManifest) {
			return manifest.Manifest{}, InvalidBundleManifestError{
				Path:   bundleManifestPath,
				Detail: "manifest has no entries",
			}
		}

		assembled = mergeManifest(assembled, bundleManifest)
	}

	if err := assembled.Validate(); err != nil {
		return manifest.Manifest{}, InvalidBundleManifestError{
			Path:   "assembled manifest",
			Detail: err.Error(),
		}
	}
	return assembled, nil
}

func mergeManifest(base manifest.Manifest, addition manifest.Manifest) manifest.Manifest {
	return manifest.Manifest{
		RuleTemplates:         uniqStrings(append(base.RuleTemplates, addition.RuleTemplates...)),
		ManagedTargets:        uniqStrings(append(base.ManagedTargets, addition.ManagedTargets...)),
		PreservedTargets:      uniqStrings(append(base.PreservedTargets, addition.PreservedTargets...)),
		RequiredTemplateFiles: uniqStrings(append(base.RequiredTemplateFiles, addition.RequiredTemplateFiles...)),
		RequiredTemplateDirs:  uniqStrings(append(base.RequiredTemplateDirs, addition.RequiredTemplateDirs...)),
	}
}

func uniqStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func isEmptyManifest(m manifest.Manifest) bool {
	return len(m.RuleTemplates) == 0 &&
		len(m.ManagedTargets) == 0 &&
		len(m.PreservedTargets) == 0 &&
		len(m.RequiredTemplateFiles) == 0 &&
		len(m.RequiredTemplateDirs) == 0
}

type assemblyLayer struct {
	name   string
	source Source
}

func ValidateAssembly(raw Source, bundles []string) error {
	layers := assemblyLayers(raw, bundles)
	seen := map[string]assemblyPath{}

	for _, layer := range layers {
		if layer.source == nil {
			continue
		}
		if err := walkAssemblySource(layer.source, ".", func(path string, kind entryKind, data []byte) error {
			if shouldIgnoreAssemblyPath(path) {
				return nil
			}
			existing, ok := seen[path]
			if !ok {
				seen[path] = assemblyPath{
					owner: layer.name,
					kind:  kind,
					data:  append([]byte(nil), data...),
				}
				return nil
			}

			if existing.kind != kind {
				return AssemblyConflictError{
					Path:   path,
					Detail: fmt.Sprintf("%s defines %s, %s defines %s", existing.owner, existing.kind, layer.name, kind),
				}
			}
			if kind == entryDir {
				return nil
			}
			if !bytes.Equal(existing.data, data) {
				return AssemblyConflictError{
					Path:   path,
					Detail: fmt.Sprintf("%s and %s provide different content", existing.owner, layer.name),
				}
			}
			return nil
		}); err != nil {
			return err
		}
	}

	return nil
}

type entryKind string

const (
	entryFile entryKind = "file"
	entryDir  entryKind = "directory"
)

type assemblyPath struct {
	owner string
	kind  entryKind
	data  []byte
}

func assemblyLayers(raw Source, bundles []string) []assemblyLayer {
	layers := make([]assemblyLayer, 0, len(bundles))
	layers = append(layers, assemblyLayer{
		name:   "base",
		source: NewPrefixedSource(raw, "base"),
	})
	for _, bundleID := range bundles {
		if bundleID == "" || bundleID == "base" {
			continue
		}
		layers = append(layers, assemblyLayer{
			name:   bundleID,
			source: NewPrefixedSource(raw, filepath.ToSlash(filepath.Join("bundles", bundleID))),
		})
	}
	return layers
}

func walkAssemblySource(src Source, root string, visit func(path string, kind entryKind, data []byte) error) error {
	info, err := src.Stat(root)
	if err != nil {
		if isNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		data, err := src.ReadFile(root)
		if err != nil {
			return err
		}
		return visit(cleanAssemblyPath(root), entryFile, data)
	}
	return walkAssemblyDir(src, root, visit)
}

func walkAssemblyDir(src Source, dir string, visit func(path string, kind entryKind, data []byte) error) error {
	entries, err := src.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		child := childAssemblyPath(dir, entry.Name())
		if entry.IsDir() {
			if err := visit(cleanAssemblyPath(child), entryDir, nil); err != nil {
				return err
			}
			if err := walkAssemblyDir(src, child, visit); err != nil {
				return err
			}
			continue
		}

		data, err := src.ReadFile(child)
		if err != nil {
			return err
		}
		if err := visit(cleanAssemblyPath(child), entryFile, data); err != nil {
			return err
		}
	}
	return nil
}

func childAssemblyPath(parent, name string) string {
	if parent == "." || parent == "" {
		return name
	}
	return filepath.ToSlash(filepath.Join(parent, name))
}

func cleanAssemblyPath(path string) string {
	cleaned := filepath.ToSlash(path)
	if cleaned == "." || cleaned == "" {
		return "."
	}
	return cleaned
}

func shouldIgnoreAssemblyPath(path string) bool {
	return path == "manifest.txt"
}
