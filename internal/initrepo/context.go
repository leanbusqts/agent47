package initrepo

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/leanbusqts/agent47/internal/contextmap"
)

const maxExistingContextBytes = 1024 * 1024

type ContextOptions struct {
	WorkDir string
	Content []byte
	Force   bool
}

type ContextService struct {
	service Service
}

func NewContextService() *ContextService {
	return &ContextService{}
}

func (s *ContextService) Plan(opts ContextOptions) (Plan, contextmap.Action, error) {
	prepared, action, err := s.prepare(opts)
	if err != nil {
		return Plan{}, "", err
	}
	return prepared.plan, action, nil
}

func (s *ContextService) Run(ctx context.Context, opts ContextOptions) (contextmap.Action, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	prepared, action, err := s.prepare(opts)
	if err != nil {
		return "", err
	}
	if action == contextmap.ActionCurrent {
		return action, nil
	}
	if err := s.service.runPrepared(ctx, prepared); err != nil {
		return "", err
	}
	return action, nil
}

func (s *ContextService) prepare(opts ContextOptions) (prepared, contextmap.Action, error) {
	if len(opts.Content) == 0 {
		return prepared{}, "", errors.New("generated context content is required")
	}
	if err := validateGeneratedContextContent(opts.Content); err != nil {
		return prepared{}, "", err
	}
	rootPath, err := canonicalRoot(opts.WorkDir)
	if err != nil {
		return prepared{}, "", err
	}
	root, err := s.service.rootFactory()(rootPath)
	if err != nil {
		return prepared{}, "", err
	}
	defer root.Close()
	if err := validateManagedDirectory(root, ".agent47"); err != nil {
		return prepared{}, "", err
	}

	item := target{
		rel:     contextmap.TargetPath,
		parent:  ".agent47",
		name:    "context.md",
		data:    append([]byte(nil), opts.Content...),
		replace: true,
	}
	action := contextmap.ActionCreate
	parent, err := root.OpenRoot(".agent47")
	if err == nil {
		existing, snapshot, readErr := readManagedFile(parent, item.name, maxExistingContextBytes)
		closeErr := parent.Close()
		if closeErr != nil {
			return prepared{}, "", closeErr
		}
		if readErr == nil {
			item.before = snapshot
			action, err = contextmap.Decide(existing, contextmap.Document{Content: opts.Content}, opts.Force)
			if err != nil {
				return prepared{}, "", err
			}
		} else if !errors.Is(readErr, fs.ErrNotExist) {
			return prepared{}, "", readErr
		}
	} else if !errors.Is(err, fs.ErrNotExist) {
		return prepared{}, "", err
	}

	plan := Plan{}
	switch action {
	case contextmap.ActionCreate:
		item.write = true
		plan.Create = []string{item.rel}
	case contextmap.ActionUpdate:
		item.write = true
		plan.Update = []string{item.rel}
	case contextmap.ActionCurrent:
		plan.Keep = []string{item.rel}
	default:
		return prepared{}, "", errors.New("invalid generated context action")
	}
	return prepared{root: rootPath, plan: plan, targets: []target{item}}, action, nil
}

func validateGeneratedContextContent(content []byte) error {
	metadata, body, err := contextmap.Parse(content)
	bodyDigest := sha256.Sum256(body)
	if err != nil || len(body) == 0 || metadata.BodySHA256 != "sha256:"+hex.EncodeToString(bodyDigest[:]) {
		return errors.New("generated context content is invalid")
	}
	return nil
}

func readManagedFile(root confinedRoot, name string, limit int64) ([]byte, fileSnapshot, error) {
	before, err := root.Lstat(name)
	if err != nil {
		return nil, fileSnapshot{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return nil, fileSnapshot{}, fmt.Errorf("managed target must not be a symlink: %s", filepath.ToSlash(name))
	}
	if !before.Mode().IsRegular() {
		return nil, fileSnapshot{}, fmt.Errorf("managed target must be a regular file: %s", filepath.ToSlash(name))
	}
	if before.Size() > limit {
		return nil, fileSnapshot{}, fmt.Errorf("managed context exceeds %d bytes", limit)
	}
	file, err := root.Open(name)
	if err != nil {
		return nil, fileSnapshot{}, err
	}
	var buffer bytes.Buffer
	hash := sha256.New()
	written, copyErr := io.Copy(io.MultiWriter(&buffer, hash), io.LimitReader(file, limit+1))
	afterOpen, statErr := file.Stat()
	closeErr := file.Close()
	if copyErr != nil {
		return nil, fileSnapshot{}, copyErr
	}
	if written > limit {
		return nil, fileSnapshot{}, fmt.Errorf("managed context exceeds %d bytes", limit)
	}
	if statErr != nil {
		return nil, fileSnapshot{}, statErr
	}
	if closeErr != nil {
		return nil, fileSnapshot{}, closeErr
	}
	afterPath, err := root.Lstat(name)
	if err != nil || afterPath.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, afterOpen) || !os.SameFile(afterOpen, afterPath) || before.Size() != afterPath.Size() || !before.ModTime().Equal(afterPath.ModTime()) {
		return nil, fileSnapshot{}, concurrentChangeError(filepath.ToSlash(name))
	}
	var fileDigest [sha256.Size]byte
	copy(fileDigest[:], hash.Sum(nil))
	data := make([]byte, buffer.Len())
	copy(data, buffer.Bytes())
	return data, fileSnapshot{exists: true, mode: afterPath.Mode(), digest: fileDigest, info: afterPath}, nil
}
