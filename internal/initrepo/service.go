package initrepo

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/leanbusqts/agent47/internal/resolve"
	"github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/templates"
)

const agentsFile = "AGENTS.md"

const (
	mutationBackup  = "backup"
	mutationInstall = "install"
	mutationRemove  = "remove"
	mutationRestore = "restore"
)

type Options struct {
	Force      bool
	WorkDir    string
	InstallSet resolve.InstallSet
	Context    []byte
}

type Plan struct {
	Create []string
	Update []string
	Keep   []string
	Remove []string
}

type Service struct {
	sourceFor      func(resolve.InstallSet) templates.Source
	validateSet    func(resolve.InstallSet) error
	openRoot       func(string) (confinedRoot, error)
	beforeMutation func(operation, path string) error
}

type fileSnapshot struct {
	exists bool
	mode   fs.FileMode
	digest [sha256.Size]byte
	info   fs.FileInfo
}

type target struct {
	rel      string
	parent   string
	name     string
	template string
	data     []byte
	before   fileSnapshot
	write    bool
	replace  bool
}

type prepared struct {
	root    string
	plan    Plan
	targets []target
}

type transactionParent struct {
	root    confinedRoot
	prefix  string
	binding fileSnapshot
	temps   []string
}

type transaction struct {
	repository     confinedRoot
	rootPath       string
	rootInfo       fs.FileInfo
	parents        map[string]*transactionParent
	createdParents map[string]bool
}

type committedChange struct {
	target    target
	parent    *transactionParent
	backup    string
	installed fileSnapshot
	created   bool
}

func New(cfg runtime.Config) (*Service, error) {
	loader, err := templates.NewLoader(cfg.TemplateMode, cfg.RepoRoot)
	if err != nil {
		return nil, err
	}

	service := &Service{
		sourceFor: func(set resolve.InstallSet) templates.Source {
			if len(set.Bundles) == 0 {
				return loader.Source
			}
			return loader.BundleSource(set.Bundles)
		},
		openRoot: openConfinedRoot,
	}
	service.validateSet = func(set resolve.InstallSet) error {
		return validateSelectedTemplates(loader.RawSource, service.sourceFor(set), set)
	}
	return service, nil
}

func (s *Service) Plan(workDir string, set resolve.InstallSet, force bool) (Plan, error) {
	return s.PlanWithContext(workDir, set, force, nil)
}

func (s *Service) PlanWithContext(workDir string, set resolve.InstallSet, force bool, contextData []byte) (Plan, error) {
	prepared, err := s.prepare(workDir, set, force, contextData)
	if err != nil {
		return Plan{}, err
	}
	return prepared.plan, nil
}

func (s *Service) Run(ctx context.Context, opts Options) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	prepared, err := s.prepare(opts.WorkDir, opts.InstallSet, opts.Force, opts.Context)
	if err != nil {
		return err
	}
	if !opts.Force {
		return s.runPrepared(ctx, prepared)
	}

	cleanup, err := s.stageLegacyCleanup(ctx, prepared.root)
	if err != nil {
		return err
	}
	restoreCleanup := true
	defer func() {
		if restoreCleanup {
			_ = cleanup.rollback()
		}
		_ = cleanup.close()
	}()

	prepared, err = s.prepare(opts.WorkDir, opts.InstallSet, true, opts.Context)
	if err != nil {
		rollbackErr := cleanup.rollback()
		restoreCleanup = false
		return errors.Join(err, rollbackErr)
	}
	if err := s.runPrepared(ctx, prepared); err != nil {
		rollbackErr := cleanup.rollback()
		restoreCleanup = false
		return errors.Join(err, rollbackErr)
	}
	restoreCleanup = false
	if err := cleanup.finalize(); err != nil {
		return err
	}
	return nil
}

func (s *Service) runPrepared(ctx context.Context, prepared prepared) error {

	txn, err := s.startTransaction(prepared)
	if err != nil {
		return err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = txn.cleanup()
		}
		_ = txn.close()
	}()

	if err := s.stage(ctx, txn, prepared.targets); err != nil {
		return err
	}
	changes, err := s.commit(ctx, txn, prepared.targets)
	if err == nil {
		err = s.verifyCommitted(txn, changes)
	}
	if err == nil {
		return nil
	}

	rollbackErr := s.rollback(txn, changes)
	if rollbackErr != nil {
		_ = txn.cleanupEmptyParents()
		cleanup = false
		return fmt.Errorf("%w; rollback incomplete: %v; recovery files retained in %s", err, rollbackErr, txn.recoveryPaths())
	}
	return err
}

func (s *Service) prepare(workDir string, set resolve.InstallSet, force bool, contextData []byte) (prepared, error) {
	rootPath, err := canonicalRoot(workDir)
	if err != nil {
		return prepared{}, err
	}
	if s.validateSet != nil {
		if err := s.validateSet(set); err != nil {
			return prepared{}, fmt.Errorf("validate selected templates: %w", err)
		}
	}

	source := s.sourceFor(set)
	if source == nil {
		return prepared{}, errors.New("template source is unavailable")
	}
	root, err := s.rootFactory()(rootPath)
	if err != nil {
		return prepared{}, err
	}
	defer root.Close()

	targets, err := buildTargets(root, set.Rules, contextData)
	if err != nil {
		return prepared{}, err
	}
	plan := Plan{}
	for i := range targets {
		item := &targets[i]
		if item.template != "" {
			data, readErr := source.ReadFile(item.template)
			if readErr != nil {
				return prepared{}, templates.MissingTemplateError{Path: item.template}
			}
			item.data = append([]byte(nil), data...)
		}

		var snapshot fileSnapshot
		var snapshotErr error
		if force && item.parent == "rules" {
			snapshot, snapshotErr = inspectRuleTargetForReplacement(root, *item)
		} else {
			snapshot, snapshotErr = inspectTarget(root, *item)
		}
		switch {
		case snapshotErr == nil:
			item.before = snapshot
			item.write = force && item.replace
			if item.write {
				plan.Update = append(plan.Update, item.rel)
			} else {
				plan.Keep = append(plan.Keep, item.rel)
			}
		case errors.Is(snapshotErr, fs.ErrNotExist):
			item.write = true
			plan.Create = append(plan.Create, item.rel)
		default:
			return prepared{}, snapshotErr
		}
	}
	if force {
		removals, err := legacyRemovalPlan(rootPath, set.Rules)
		if err != nil {
			return prepared{}, err
		}
		plan.Remove = removals
	}
	return prepared{root: rootPath, plan: plan, targets: targets}, nil
}

func inspectRuleTargetForReplacement(root confinedRoot, item target) (fileSnapshot, error) {
	rules, err := root.OpenRoot("rules")
	if errors.Is(err, fs.ErrNotExist) {
		return fileSnapshot{}, fs.ErrNotExist
	}
	if err != nil {
		return fileSnapshot{}, err
	}
	defer rules.Close()
	info, err := rules.Lstat(item.name)
	if err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{exists: true, mode: info.Mode(), info: info}, nil
}

func (s *Service) startTransaction(prepared prepared) (*transaction, error) {
	repository, err := s.rootFactory()(prepared.root)
	if err != nil {
		return nil, err
	}
	rootInfo, err := repositoryInfo(repository)
	if err != nil {
		repository.Close()
		return nil, err
	}
	txn := &transaction{
		repository:     repository,
		rootPath:       prepared.root,
		rootInfo:       rootInfo,
		parents:        make(map[string]*transactionParent),
		createdParents: make(map[string]bool),
	}

	writable := writableTargets(prepared.targets)
	if len(writable) == 0 {
		return txn, nil
	}
	if err := txn.addParent(".", repository, fileSnapshot{exists: true, info: rootInfo, mode: rootInfo.Mode()}); err != nil {
		txn.close()
		return nil, err
	}
	for _, parentName := range targetParents(writable) {
		if parentName == "." {
			continue
		}
		made, err := ensureManagedDirectory(repository, parentName)
		if err != nil {
			txn.close()
			return nil, err
		}
		parentRoot, err := repository.OpenRoot(parentName)
		if err != nil {
			txn.close()
			return nil, fmt.Errorf("open managed directory %s: %w", parentName, err)
		}
		parentInfo, err := repository.Lstat(parentName)
		if err != nil || parentInfo.Mode()&os.ModeSymlink != 0 || !parentInfo.IsDir() {
			parentRoot.Close()
			txn.close()
			return nil, fmt.Errorf("managed directory changed during operation: %s", parentName)
		}
		if err := txn.addParent(parentName, parentRoot, fileSnapshot{exists: true, info: parentInfo, mode: parentInfo.Mode()}); err != nil {
			parentRoot.Close()
			if made {
				parentNow, statErr := repository.Lstat(parentName)
				if statErr == nil && os.SameFile(parentInfo, parentNow) {
					_ = repository.Remove(parentName)
				}
			}
			txn.close()
			return nil, err
		}
		txn.createdParents[parentName] = made
	}
	return txn, nil
}

func (t *transaction) addParent(name string, root confinedRoot, binding fileSnapshot) error {
	token, err := randomToken()
	if err != nil {
		return err
	}
	t.parents[name] = &transactionParent{root: root, prefix: ".agent47-init-" + token + "-", binding: binding}
	return nil
}

func (s *Service) stage(ctx context.Context, txn *transaction, targets []target) error {
	for _, item := range writableTargets(targets) {
		if err := ctx.Err(); err != nil {
			return err
		}
		parent := txn.parents[item.parent]
		mode := item.before.mode.Perm()
		if mode == 0 {
			mode = 0o644
		}
		stagePath := parent.tempPath("stage", item.name)
		if err := writeExclusive(parent.root, stagePath, item.data, mode); err != nil {
			return fmt.Errorf("stage %s: %w", item.rel, err)
		}
		parent.temps = append(parent.temps, stagePath)
	}
	for _, item := range writableTargets(targets) {
		if err := txn.verifyBinding(item.parent); err != nil {
			return err
		}
		current, err := inspectFile(txn.parents[item.parent].root, item.name)
		if err := compareCurrent(item.before, current, err, item.rel); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) commit(ctx context.Context, txn *transaction, targets []target) ([]committedChange, error) {
	changes := make([]committedChange, 0, len(targets))
	for _, item := range writableTargets(targets) {
		if err := ctx.Err(); err != nil {
			return changes, err
		}
		parent := txn.parents[item.parent]
		if err := txn.verifyBinding(item.parent); err != nil {
			return changes, err
		}
		current, err := inspectFile(parent.root, item.name)
		if err := compareCurrent(item.before, current, err, item.rel); err != nil {
			return changes, err
		}

		stagePath := parent.tempPath("stage", item.name)
		stageSnapshot, stageErr := inspectFile(parent.root, stagePath)
		if stageErr != nil || stageSnapshot.digest != sha256.Sum256(item.data) {
			return changes, fmt.Errorf("staged file changed before install: %s", item.rel)
		}
		change := committedChange{target: item, parent: parent, created: !item.before.exists}
		if item.before.exists {
			change.backup = parent.tempPath("backup", item.name)
			if err := s.mutate(mutationBackup, item.rel); err != nil {
				return changes, err
			}
			if err := txn.verifyBinding(item.parent); err != nil {
				return changes, err
			}
			if err := parent.root.Link(item.name, change.backup); err != nil {
				return changes, fmt.Errorf("backup %s: %w", item.rel, err)
			}
			parent.temps = append(parent.temps, change.backup)
			backup, backupErr := inspectFile(parent.root, change.backup)
			if backupErr != nil || !sameSnapshot(item.before, backup, true) {
				_ = parent.root.Remove(change.backup)
				return changes, concurrentChangeError(item.rel)
			}
			current, err = inspectFile(parent.root, item.name)
			if err != nil || !sameSnapshot(item.before, current, true) || !os.SameFile(backup.info, current.info) {
				_ = parent.root.Remove(change.backup)
				return changes, concurrentChangeError(item.rel)
			}
		} else if err := ensureAbsent(parent.root, item.name, item.rel); err != nil {
			return changes, err
		}

		if err := s.mutate(mutationInstall, item.rel); err != nil {
			if change.backup != "" {
				_ = parent.root.Remove(change.backup)
			}
			return changes, err
		}
		if err := txn.verifyBinding(item.parent); err != nil {
			if change.backup != "" {
				_ = parent.root.Remove(change.backup)
			}
			return changes, err
		}

		if item.before.exists {
			if err := parent.root.Rename(stagePath, item.name); err != nil {
				_ = parent.root.Remove(change.backup)
				return changes, fmt.Errorf("install %s: %w", item.rel, err)
			}
		} else {
			if err := parent.root.Link(stagePath, item.name); err != nil {
				if errors.Is(err, fs.ErrExist) {
					return changes, concurrentChangeError(item.rel)
				}
				return changes, fmt.Errorf("create %s: %w", item.rel, err)
			}
		}
		change.installed = stageSnapshot
		changes = append(changes, change)
		if !item.before.exists {
			if err := parent.root.Remove(stagePath); err != nil {
				return changes, fmt.Errorf("finalize %s: %w", item.rel, err)
			}
		}
		if err := parent.root.Sync(); err != nil {
			return changes, err
		}
		installed, err := inspectFile(parent.root, item.name)
		if err != nil || installed.digest != sha256.Sum256(item.data) {
			err = concurrentChangeError(item.rel)
			return changes, err
		}
		changes[len(changes)-1].installed = installed
		if err := txn.verifyBinding(item.parent); err != nil {
			return changes, err
		}
	}
	return changes, nil
}

func (s *Service) verifyCommitted(txn *transaction, changes []committedChange) error {
	for _, change := range changes {
		if err := txn.verifyBinding(change.target.parent); err != nil {
			return err
		}
		current, err := inspectFile(change.parent.root, change.target.name)
		if err != nil || !sameSnapshot(change.installed, current, true) {
			return concurrentChangeError(change.target.rel)
		}
		if change.backup != "" {
			backup, backupErr := inspectFile(change.parent.root, change.backup)
			if backupErr != nil || !sameSnapshot(change.target.before, backup, true) {
				return concurrentChangeError(change.target.rel)
			}
		}
	}
	return nil
}

func (s *Service) rollback(txn *transaction, changes []committedChange) error {
	var rollbackErrors []error
	for i := len(changes) - 1; i >= 0; i-- {
		change := changes[i]
		if err := txn.verifyBinding(change.target.parent); err != nil {
			rollbackErrors = append(rollbackErrors, err)
			continue
		}
		current, err := inspectFile(change.parent.root, change.target.name)
		if err != nil || !sameSnapshot(change.installed, current, true) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback conflict for %s: installed file changed; left untouched", change.target.rel))
			continue
		}

		if change.created {
			if err := s.mutate(mutationRemove, change.target.rel); err != nil {
				rollbackErrors = append(rollbackErrors, err)
				continue
			}
			current, err = inspectFile(change.parent.root, change.target.name)
			if err != nil || !sameSnapshot(change.installed, current, true) {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback conflict for %s: installed file changed; left untouched", change.target.rel))
				continue
			}
			if err := change.parent.root.Remove(change.target.name); err != nil {
				rollbackErrors = append(rollbackErrors, err)
			}
			continue
		}

		backup, backupErr := inspectFile(change.parent.root, change.backup)
		if backupErr != nil || !sameIdentity(change.target.before, backup) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback conflict for %s: backup changed identity; left untouched", change.target.rel))
			continue
		}
		if err := s.mutate(mutationRestore, change.target.rel); err != nil {
			rollbackErrors = append(rollbackErrors, err)
			continue
		}
		current, err = inspectFile(change.parent.root, change.target.name)
		if err != nil || !sameSnapshot(change.installed, current, true) {
			rollbackErrors = append(rollbackErrors, fmt.Errorf("rollback conflict for %s: installed file changed; left untouched", change.target.rel))
			continue
		}
		if err := change.parent.root.Rename(change.backup, change.target.name); err != nil {
			rollbackErrors = append(rollbackErrors, err)
		}
	}
	for _, parent := range txn.parents {
		if err := parent.root.Sync(); err != nil {
			rollbackErrors = append(rollbackErrors, err)
		}
	}
	return errors.Join(rollbackErrors...)
}

func (s *Service) mutate(operation, path string) error {
	if s.beforeMutation == nil {
		return nil
	}
	return s.beforeMutation(operation, path)
}

func (s *Service) rootFactory() func(string) (confinedRoot, error) {
	if s.openRoot != nil {
		return s.openRoot
	}
	return openConfinedRoot
}

func (t *transaction) verifyBinding(parent string) error {
	rootNow, err := os.Lstat(t.rootPath)
	if err != nil || rootNow.Mode()&os.ModeSymlink != 0 || !rootNow.IsDir() || !os.SameFile(t.rootInfo, rootNow) {
		return errors.New("repository root changed during init")
	}
	if parent == "." {
		return nil
	}
	parentNow, err := t.repository.Lstat(parent)
	if err != nil || parentNow.Mode()&os.ModeSymlink != 0 || !parentNow.IsDir() || !sameIdentity(t.parents[parent].binding, fileSnapshot{exists: true, info: parentNow}) {
		return fmt.Errorf("managed directory changed during operation: %s", parent)
	}
	return nil
}

func (t *transaction) cleanup() error {
	var cleanupErrors []error
	for _, parent := range t.parents {
		for _, path := range parent.temps {
			if err := parent.root.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				cleanupErrors = append(cleanupErrors, err)
			}
		}
		if err := parent.root.Sync(); err != nil {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	for parent, made := range t.createdParents {
		if !made {
			continue
		}
		parentNow, err := t.repository.Lstat(parent)
		if err == nil && sameIdentity(t.parents[parent].binding, fileSnapshot{exists: true, info: parentNow}) {
			_ = t.repository.Remove(parent)
		}
	}
	return errors.Join(cleanupErrors...)
}

func (t *transaction) cleanupEmptyParents() error {
	var cleanupErrors []error
	for parent, made := range t.createdParents {
		if !made {
			continue
		}
		parentNow, err := t.repository.Lstat(parent)
		if err != nil {
			cleanupErrors = append(cleanupErrors, err)
			continue
		}
		if !sameIdentity(t.parents[parent].binding, fileSnapshot{exists: true, info: parentNow}) {
			cleanupErrors = append(cleanupErrors, fmt.Errorf("managed directory changed during rollback: %s", parent))
			continue
		}
		for _, path := range t.parents[parent].temps {
			if err := t.parents[parent].root.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
				cleanupErrors = append(cleanupErrors, err)
			}
		}
		if err := t.repository.Remove(parent); err != nil && !errors.Is(err, fs.ErrNotExist) {
			cleanupErrors = append(cleanupErrors, err)
		}
	}
	if err := t.repository.Sync(); err != nil {
		cleanupErrors = append(cleanupErrors, err)
	}
	return errors.Join(cleanupErrors...)
}

func (t *transaction) close() error {
	var closeErrors []error
	for name, parent := range t.parents {
		if name == "." {
			continue
		}
		if err := parent.root.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	if t.repository != nil {
		if err := t.repository.Close(); err != nil {
			closeErrors = append(closeErrors, err)
		}
	}
	return errors.Join(closeErrors...)
}

func (t *transaction) recoveryPaths() string {
	paths := make([]string, 0, len(t.parents))
	for name, parent := range t.parents {
		if name == "." {
			paths = append(paths, filepath.Join(t.rootPath, parent.prefix+"*"))
		} else {
			paths = append(paths, filepath.Join(t.rootPath, name, parent.prefix+"*"))
		}
	}
	sort.Strings(paths)
	return strings.Join(paths, ", ")
}

func buildTargets(root confinedRoot, ruleNames []string, contextData []byte) ([]target, error) {
	if err := validateManagedDirectory(root, "rules"); err != nil {
		return nil, err
	}
	if len(contextData) > 0 {
		if err := validateGeneratedContextContent(contextData); err != nil {
			return nil, err
		}
		if err := validateManagedDirectory(root, ".agent47"); err != nil {
			return nil, err
		}
	}

	targets := []target{{rel: agentsFile, parent: ".", name: agentsFile, template: agentsFile, replace: true}}
	seen := make(map[string]bool, len(ruleNames))
	for _, name := range ruleNames {
		if err := validateRuleName(name); err != nil {
			return nil, err
		}
		if seen[name] {
			continue
		}
		seen[name] = true
		rel := filepath.ToSlash(filepath.Join("rules", name))
		targets = append(targets, target{rel: rel, parent: "rules", name: name, template: rel, replace: true})
	}
	if len(contextData) > 0 {
		targets = append(targets, target{rel: ".agent47/context.md", parent: ".agent47", name: "context.md", data: append([]byte(nil), contextData...)})
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].rel < targets[j].rel })
	return targets, nil
}

func validateSelectedTemplates(raw templates.Source, selected templates.Source, set resolve.InstallSet) error {
	if raw == nil || selected == nil {
		return errors.New("template source is unavailable")
	}
	if err := templates.ValidateAssembly(raw, set.Bundles); err != nil {
		return err
	}
	assembled, err := templates.AssembleManifest(raw, set.Bundles)
	if err != nil {
		return err
	}
	for _, rule := range set.Rules {
		if err := validateRuleName(rule); err != nil {
			return err
		}
		target := filepath.ToSlash(filepath.Join("rules", rule))
		info, statErr := selected.Stat(target)
		if statErr != nil || !info.Mode().IsRegular() {
			return templates.MissingTemplateError{Path: target}
		}
	}
	if !contains(assembled.ManagedTargets, agentsFile) {
		return fmt.Errorf("assembled manifest does not manage %s", agentsFile)
	}
	if !sameStringSet(assembled.GeneratedTargets, []string{".agent47/context.md"}) {
		return errors.New("assembled manifest generated target contract is invalid")
	}
	if !sameStringSet(assembled.ForceCleanupTargets, []string{"rules/", "skills/", "prompts/", "specs/spec.yml", ".agents/specs/spec.yml"}) {
		return errors.New("assembled manifest force cleanup contract is invalid")
	}
	agentsInfo, err := selected.Stat(agentsFile)
	if err != nil || !agentsInfo.Mode().IsRegular() {
		return templates.MissingTemplateError{Path: agentsFile}
	}
	for _, path := range assembled.RequiredTemplateFiles {
		if path == "manifest.txt" {
			continue
		}
		info, err := selected.Stat(path)
		if err != nil || !info.Mode().IsRegular() {
			return templates.MissingTemplateError{Path: path}
		}
	}
	for _, path := range assembled.RequiredTemplateDirs {
		info, err := selected.Stat(path)
		if err != nil || !info.IsDir() {
			return templates.MissingTemplateError{Path: path}
		}
	}
	return nil
}

func validateRuleName(name string) error {
	if name == "" || filepath.Base(name) != name || strings.ContainsAny(name, `/\\`) || filepath.Ext(name) != ".yaml" {
		return fmt.Errorf("unsafe rule template name: %q", name)
	}
	for _, char := range name {
		valid := char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("._-", char)
		if !valid {
			return fmt.Errorf("unsafe rule template name: %q", name)
		}
	}
	return nil
}

func canonicalRoot(workDir string) (string, error) {
	if workDir == "" {
		return "", errors.New("work directory is required")
	}
	absRoot, err := filepath.Abs(workDir)
	if err != nil {
		return "", err
	}
	root, err := filepath.EvalSymlinks(absRoot)
	if err != nil {
		return "", fmt.Errorf("resolve work directory: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("work directory must be a directory")
	}
	return filepath.Clean(root), nil
}

func ensureManagedDirectory(root confinedRoot, name string) (bool, error) {
	info, err := root.Lstat(name)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return false, fmt.Errorf("managed directory must not be a symlink: %s", name)
		}
		if !info.IsDir() {
			return false, fmt.Errorf("managed path must be a directory: %s", name)
		}
		return false, nil
	}
	if !errors.Is(err, fs.ErrNotExist) {
		return false, err
	}
	if err := root.Mkdir(name, 0o755); err != nil {
		if !errors.Is(err, fs.ErrExist) {
			return false, err
		}
		info, statErr := root.Lstat(name)
		if statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return false, fmt.Errorf("managed directory changed during operation: %s", name)
		}
		return false, nil
	}
	if err := root.Sync(); err != nil {
		return true, err
	}
	return true, nil
}

func validateManagedDirectory(root confinedRoot, name string) error {
	info, err := root.Lstat(name)
	if err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("managed directory must not be a symlink: %s", name)
		}
		if !info.IsDir() {
			return fmt.Errorf("managed path must be a directory: %s", name)
		}
		return nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return err
}

func inspectFile(root confinedRoot, path string) (fileSnapshot, error) {
	before, err := root.Lstat(path)
	if err != nil {
		return fileSnapshot{}, err
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return fileSnapshot{}, fmt.Errorf("managed target must not be a symlink: %s", filepath.ToSlash(path))
	}
	if !before.Mode().IsRegular() {
		return fileSnapshot{}, fmt.Errorf("managed target must be a regular file: %s", filepath.ToSlash(path))
	}
	file, err := root.Open(path)
	if err != nil {
		return fileSnapshot{}, err
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	afterOpen, statErr := file.Stat()
	closeErr := file.Close()
	if copyErr != nil {
		return fileSnapshot{}, copyErr
	}
	if statErr != nil {
		return fileSnapshot{}, statErr
	}
	if closeErr != nil {
		return fileSnapshot{}, closeErr
	}
	afterPath, err := root.Lstat(path)
	if err != nil || afterPath.Mode()&os.ModeSymlink != 0 || !os.SameFile(before, afterOpen) || !os.SameFile(afterOpen, afterPath) || before.Size() != afterPath.Size() || !before.ModTime().Equal(afterPath.ModTime()) {
		return fileSnapshot{}, concurrentChangeError(filepath.ToSlash(path))
	}
	var digest [sha256.Size]byte
	copy(digest[:], hash.Sum(nil))
	return fileSnapshot{exists: true, mode: afterPath.Mode(), digest: digest, info: afterPath}, nil
}

func inspectTarget(root confinedRoot, item target) (fileSnapshot, error) {
	if item.parent == "." {
		return inspectFile(root, item.name)
	}
	parent, err := root.OpenRoot(item.parent)
	if err != nil {
		return fileSnapshot{}, err
	}
	defer parent.Close()
	return inspectFile(parent, item.name)
}

func writeExclusive(root confinedRoot, path string, data []byte, mode fs.FileMode) error {
	file, err := root.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode.Perm())
	if err != nil {
		return err
	}
	written := 0
	for written < len(data) {
		n, writeErr := file.Write(data[written:])
		if writeErr != nil {
			_ = file.Close()
			return writeErr
		}
		written += n
	}
	if err := file.Sync(); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func compareCurrent(expected fileSnapshot, current fileSnapshot, currentErr error, path string) error {
	if expected.exists {
		if currentErr != nil || !sameSnapshot(expected, current, true) {
			return concurrentChangeError(path)
		}
		return nil
	}
	if currentErr == nil || !errors.Is(currentErr, fs.ErrNotExist) {
		return concurrentChangeError(path)
	}
	return nil
}

func ensureAbsent(root confinedRoot, name, rel string) error {
	_, err := root.Lstat(name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	return concurrentChangeError(rel)
}

func sameSnapshot(expected, actual fileSnapshot, identity bool) bool {
	if !expected.exists || !actual.exists || expected.mode != actual.mode || expected.digest != actual.digest {
		return false
	}
	return !identity || sameIdentity(expected, actual)
}

func sameIdentity(expected, actual fileSnapshot) bool {
	return expected.exists && actual.exists && expected.info != nil && actual.info != nil && os.SameFile(expected.info, actual.info)
}

func repositoryInfo(root confinedRoot) (fs.FileInfo, error) {
	info, err := root.Stat(".")
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("work directory must be a directory")
	}
	return info, nil
}

func writableTargets(targets []target) []target {
	result := make([]target, 0, len(targets))
	for _, item := range targets {
		if item.write {
			result = append(result, item)
		}
	}
	return result
}

func targetParents(targets []target) []string {
	seen := map[string]bool{}
	var result []string
	for _, item := range targets {
		if !seen[item.parent] {
			seen[item.parent] = true
			result = append(result, item.parent)
		}
	}
	sort.Strings(result)
	return result
}

func (p *transactionParent) tempPath(kind, name string) string {
	return p.prefix + kind + "-" + name
}

func randomToken() (string, error) {
	buffer := make([]byte, 12)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func concurrentChangeError(path string) error {
	return fmt.Errorf("managed target changed concurrently: %s", filepath.ToSlash(path))
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func sameStringSet(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	leftCopy := append([]string(nil), left...)
	rightCopy := append([]string(nil), right...)
	sort.Strings(leftCopy)
	sort.Strings(rightCopy)
	for index := range leftCopy {
		if leftCopy[index] != rightCopy[index] {
			return false
		}
	}
	return true
}
