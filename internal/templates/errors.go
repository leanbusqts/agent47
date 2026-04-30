package templates

import "fmt"

type MissingTemplateError struct {
	Path string
}

func (e MissingTemplateError) Error() string {
	return fmt.Sprintf("template not found: %s", e.Path)
}

type MissingBundleManifestError struct {
	Path string
}

func (e MissingBundleManifestError) Error() string {
	return fmt.Sprintf("missing bundle manifest: %s", e.Path)
}

type InvalidBundleManifestError struct {
	Path   string
	Detail string
}

func (e InvalidBundleManifestError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("invalid bundle manifest: %s", e.Path)
	}
	return fmt.Sprintf("invalid bundle manifest %s: %s", e.Path, e.Detail)
}

type AssemblyConflictError struct {
	Path   string
	Detail string
}

func (e AssemblyConflictError) Error() string {
	if e.Detail == "" {
		return fmt.Sprintf("assembly conflict for %s", e.Path)
	}
	return fmt.Sprintf("assembly conflict for %s: %s", e.Path, e.Detail)
}
