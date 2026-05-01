package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/fsx"
	"github.com/leanbusqts/agent47/internal/manifest"
	"github.com/leanbusqts/agent47/internal/resolve"
	"github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/skills"
	"github.com/leanbusqts/agent47/internal/templates"
)

const (
	projectRulesDir   = "rules"
	projectSkillsDir  = "skills"
	projectAgents     = "AGENTS.md"
	projectReadme     = "README.md"
	projectPromptsDir = "prompts"
	projectSpecsDir   = "specs"
	projectSpecFile   = "specs/spec.yml"
)

type Service struct {
	FS     fsx.Service
	Loader *templates.Loader
	Out    cli.Output
}

type Options struct {
	Force      bool
	OnlySkills bool
	Yes        bool
	WorkDir    string
	InstallSet resolve.InstallSet
}

type state struct {
	root              string
	stageRoot         string
	backupRoot        string
	createdReadme     bool
	createdSpec       bool
	createdSpecsDir   bool
	createdPromptsDir bool
	rulesDirCreated   bool
	replacedAgents    bool
	replacedSkills    bool
	createdPrompts    []string
	writtenRules      []string
	removedStale      []string
}

func New(cfg runtime.Config, out cli.Output) (*Service, error) {
	loader, err := templates.NewLoader(cfg.TemplateMode, cfg.RepoRoot)
	if err != nil {
		return nil, err
	}

	return &Service{
		FS:     fsx.Service{},
		Loader: loader,
		Out:    out,
	}, nil
}

func (s *Service) Run(ctx context.Context, opts Options) (err error) {
	var m manifest.Manifest
	var effectiveManifest manifest.Manifest
	source := s.Loader.Source

	if opts.WorkDir == "" {
		opts.WorkDir, err = os.Getwd()
		if err != nil {
			return err
		}
	}

	s.Out.Info("Initializing agent environment...")
	if err := ctx.Err(); err != nil {
		return err
	}

	if !opts.OnlySkills {
		manifestData, readErr := s.Loader.Source.ReadFile("manifest.txt")
		if readErr != nil {
			return readErr
		}

		m, err = manifest.Parse(manifestData)
		if err != nil {
			return err
		}
		if len(opts.InstallSet.Bundles) > 0 {
			if err := templates.ValidateAssembly(s.Loader.RawSource, opts.InstallSet.Bundles); err != nil {
				return err
			}
			m, err = templates.AssembleManifest(s.Loader.RawSource, opts.InstallSet.Bundles)
			if err != nil {
				return err
			}
			source = s.Loader.BundleSource(opts.InstallSet.Bundles)
		}
	}
	if !opts.OnlySkills && len(opts.InstallSet.Rules) == 0 && len(opts.InstallSet.Skills) == 0 {
		opts.InstallSet = resolve.InstallSet{
			BaseBundle: true,
			Rules:      append([]string{}, m.RuleTemplates...),
		}
	}
	effectiveManifest = m
	if !opts.OnlySkills {
		effectiveManifest = resolve.AssembleManifest(m, opts.InstallSet)
	}

	if err := s.requireTemplates(source, effectiveManifest, opts); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	st, err := s.prepareState(opts.WorkDir)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = s.rollback(opts.WorkDir, &st)
		}
		_ = os.RemoveAll(st.root)
	}()

	if err := s.stageSkills(source, opts, &st); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if !opts.OnlySkills {
		if err := s.stageRulesAndAgents(source, effectiveManifest, &st); err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
	}

	if err := s.commitSkills(opts.WorkDir, &st); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	if opts.OnlySkills {
		s.Out.OK("Agent skills ready")
		return nil
	}

	if err := s.commitRules(opts.WorkDir, effectiveManifest, opts, &st); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.commitAgents(opts.WorkDir, opts, &st); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := s.commitReadme(opts.WorkDir, &st); err != nil {
		return err
	}
	if err := s.commitSpec(source, opts.WorkDir, &st); err != nil {
		return err
	}
	if err := s.commitPrompts(source, opts.WorkDir, opts.InstallSet, &st); err != nil {
		return err
	}

	s.Out.OK("Agent environment ready")
	return nil
}

func (s *Service) requireTemplates(src templates.Source, m manifest.Manifest, opts Options) error {
	if !opts.OnlySkills {
		if _, err := src.Stat(projectAgents); err != nil {
			return templates.MissingTemplateError{Path: projectAgents}
		}
		for _, file := range m.RuleTemplates {
			if _, err := src.Stat(filepath.ToSlash(filepath.Join(projectRulesDir, file))); err != nil {
				return templates.MissingTemplateError{Path: filepath.ToSlash(filepath.Join(projectRulesDir, file))}
			}
		}
		if _, err := src.Stat(projectSpecFile); err != nil {
			return templates.MissingTemplateError{Path: projectSpecFile}
		}
		for _, file := range m.RequiredTemplateFiles {
			switch {
			case file == "manifest.txt":
				continue
			case filepath.Dir(file) == projectPromptsDir:
				if _, err := src.Stat(file); err != nil {
					return templates.MissingTemplateError{Path: file}
				}
			}
		}
	}

	if _, err := src.Stat(projectSkillsDir); err != nil {
		return templates.MissingTemplateError{Path: projectSkillsDir}
	}

	return nil
}

func (s *Service) prepareState(workDir string) (state, error) {
	stageParent := workDir
	if override := os.Getenv("AGENT47_STAGE_ROOT"); override != "" {
		stageParent = override
	}

	root, err := os.MkdirTemp(stageParent, ".agent47-stage-*")
	if err != nil {
		return state{}, err
	}

	stageRoot := filepath.Join(root, "stage")
	backupRoot := filepath.Join(root, "backup")
	if err := os.MkdirAll(stageRoot, 0o755); err != nil {
		return state{}, err
	}
	if err := os.MkdirAll(backupRoot, 0o755); err != nil {
		return state{}, err
	}

	return state{
		root:       root,
		stageRoot:  stageRoot,
		backupRoot: backupRoot,
	}, nil
}

func (s *Service) stageSkills(src templates.Source, opts Options, st *state) error {
	stageSkillsDir := filepath.Join(st.stageRoot, projectSkillsDir)
	if err := os.MkdirAll(stageSkillsDir, 0o755); err != nil {
		return err
	}

	workSkillsDir := filepath.Join(opts.WorkDir, projectSkillsDir)
	if !opts.Force && s.FS.IsDir(workSkillsDir) {
		if err := s.FS.CopyDir(workSkillsDir, stageSkillsDir); err != nil {
			return err
		}
	}

	service := skills.Service{}
	discovered, err := service.Discover(src, projectSkillsDir)
	if err != nil {
		s.Out.Err("%s", err)
		return err
	}

	selectedSkills := make(map[string]bool)
	for _, name := range opts.InstallSet.Skills {
		selectedSkills[name] = true
	}

	for _, skill := range discovered {
		skillName := filepath.Base(filepath.Dir(skill.Location))
		if len(selectedSkills) > 0 && !selectedSkills[skillName] {
			continue
		}

		stageSkillDir := filepath.Join(stageSkillsDir, skillName)
		if err := os.RemoveAll(stageSkillDir); err != nil {
			return err
		}

		existingSkillDir := filepath.Join(workSkillsDir, skillName)
		if !opts.Force && s.FS.IsDir(existingSkillDir) {
			if err := s.FS.CopyDir(existingSkillDir, stageSkillDir); err != nil {
				return err
			}
		} else {
			if err := s.copyTemplateDir(src, filepath.Dir(skill.Location), stageSkillDir); err != nil {
				return err
			}
		}

		skillPath := filepath.Join(stageSkillDir, "SKILL.md")
		body, err := os.ReadFile(skillPath)
		if err != nil {
			templateBody, readErr := src.ReadFile(skill.Location)
			if readErr != nil {
				return readErr
			}
			if writeErr := s.FS.WriteFileAtomic(skillPath, templateBody, 0o644); writeErr != nil {
				return writeErr
			}
			body = templateBody
		}

		skillLocation := filepath.ToSlash(filepath.Join(projectSkillsDir, skillName, "SKILL.md"))
		if _, err := skills.Validate(skillLocation, body); err != nil {
			if opts.Force {
				s.Out.Err("Invalid skill template: %s", skillName)
				return err
			}
			s.Out.Warn("Invalid SKILL.md for %s; preserving existing content", skillName)
			continue
		}
	}

	stagedSkills, err := s.collectAvailableSkillsXMLSkills(stageSkillsDir, opts.Force)
	if err != nil {
		s.Out.Err("%s", err)
		return err
	}

	xmlData, err := service.GenerateAvailableSkillsXML(stagedSkills)
	if err != nil {
		return err
	}

	if err := s.FS.WriteFileAtomic(filepath.Join(stageSkillsDir, "AVAILABLE_SKILLS.xml"), xmlData, 0o644); err != nil {
		return err
	}
	jsonData, err := service.GenerateAvailableSkillsJSON(stagedSkills)
	if err != nil {
		return err
	}
	if err := s.FS.WriteFileAtomic(filepath.Join(stageSkillsDir, "AVAILABLE_SKILLS.json"), jsonData, 0o644); err != nil {
		return err
	}
	summaryData, err := service.GenerateAvailableSkillsSummaryMarkdown(stagedSkills)
	if err != nil {
		return err
	}
	if err := s.FS.WriteFileAtomic(filepath.Join(stageSkillsDir, "SUMMARY.md"), summaryData, 0o644); err != nil {
		return err
	}

	s.Out.OK("Skills setup complete.")
	return nil
}

func (s *Service) stageRulesAndAgents(src templates.Source, m manifest.Manifest, st *state) error {
	stageRulesDir := filepath.Join(st.stageRoot, projectRulesDir)
	if err := os.MkdirAll(stageRulesDir, 0o755); err != nil {
		return err
	}

	for _, file := range m.RuleTemplates {
		data, err := src.ReadFile(filepath.ToSlash(filepath.Join(projectRulesDir, file)))
		if err != nil {
			return err
		}
		if err := s.FS.WriteFileAtomic(filepath.Join(stageRulesDir, file), data, 0o644); err != nil {
			return err
		}
	}

	agentsData, err := src.ReadFile(projectAgents)
	if err != nil {
		return err
	}

	return s.FS.WriteFileAtomic(filepath.Join(st.stageRoot, projectAgents), agentsData, 0o644)
}

func (s *Service) commitSkills(workDir string, st *state) error {
	target := filepath.Join(workDir, projectSkillsDir)
	stage := filepath.Join(st.stageRoot, projectSkillsDir)
	backup := filepath.Join(st.backupRoot, projectSkillsDir)

	if s.FS.Exists(target) && !s.FS.IsDir(target) {
		return fmt.Errorf("%s exists and is not a directory", projectSkillsDir)
	}

	st.replacedSkills = true
	if s.FS.IsDir(target) {
		if err := os.Rename(target, backup); err != nil {
			st.replacedSkills = false
			return err
		}
	}
	if err := os.Rename(stage, target); err != nil {
		return err
	}

	return nil
}

func (s *Service) commitRules(workDir string, m manifest.Manifest, opts Options, st *state) error {
	rulesDir := filepath.Join(workDir, projectRulesDir)
	if !s.FS.IsDir(rulesDir) {
		if err := os.Mkdir(rulesDir, 0o755); err != nil {
			return err
		}
		st.rulesDirCreated = true
		s.Out.OK("Created directory: %s/", projectRulesDir)
	}

	backupRulesDir := filepath.Join(st.backupRoot, projectRulesDir)
	if err := os.MkdirAll(backupRulesDir, 0o755); err != nil {
		return err
	}

	if opts.Force {
		entries, err := filepath.Glob(filepath.Join(rulesDir, "*.yaml"))
		if err != nil {
			return err
		}
		sort.Strings(entries)
		for _, existingRule := range entries {
			file := filepath.Base(existingRule)
			if !m.ContainsRuleTemplate(file) {
				if err := s.backupFile(existingRule, filepath.Join(backupRulesDir, file)); err != nil {
					return err
				}
				if err := os.Remove(existingRule); err != nil {
					return err
				}
				st.removedStale = append(st.removedStale, file)
				s.Out.OK("Removed stale managed rule: %s", filepath.ToSlash(filepath.Join(projectRulesDir, file)))
			}
		}
	}

	for _, file := range m.RuleTemplates {
		target := filepath.Join(rulesDir, file)
		stage := filepath.Join(st.stageRoot, projectRulesDir, file)
		if s.FS.Exists(target) {
			if opts.Force {
				if err := s.backupFile(target, filepath.Join(backupRulesDir, file)); err != nil {
					return err
				}
				if err := os.Remove(target); err != nil {
					return err
				}
				if err := os.Rename(stage, target); err != nil {
					return err
				}
				st.writtenRules = append(st.writtenRules, file)
				s.Out.OK("Updated: %s", filepath.ToSlash(filepath.Join(projectRulesDir, file)))
			} else {
				_ = os.Remove(stage)
				s.Out.Warn("%s already exists, skipping", filepath.ToSlash(filepath.Join(projectRulesDir, file)))
			}
			continue
		}

		if err := os.Rename(stage, target); err != nil {
			return err
		}
		st.writtenRules = append(st.writtenRules, file)
		s.Out.OK("Copied: %s", filepath.ToSlash(filepath.Join(projectRulesDir, file)))
	}

	return nil
}

func (s *Service) commitAgents(workDir string, opts Options, st *state) error {
	target := filepath.Join(workDir, projectAgents)
	stage := filepath.Join(st.stageRoot, projectAgents)
	backup := filepath.Join(st.backupRoot, projectAgents)

	if s.FS.Exists(target) {
		if opts.Force {
			st.replacedAgents = true
			if err := os.Rename(target, backup); err != nil {
				st.replacedAgents = false
				return err
			}
			if err := os.Rename(stage, target); err != nil {
				return err
			}
			s.Out.OK("Updated: %s", projectAgents)
			return nil
		}

		_ = os.Remove(stage)
		s.Out.Warn("%s already exists, skipping", projectAgents)
		return nil
	}

	st.replacedAgents = true
	if err := os.Rename(stage, target); err != nil {
		st.replacedAgents = false
		return err
	}
	s.Out.OK("Copied: %s", projectAgents)
	return nil
}

func (s *Service) commitReadme(workDir string, st *state) error {
	target := filepath.Join(workDir, projectReadme)
	if s.FS.Exists(target) {
		return nil
	}

	if err := s.FS.WriteFileAtomic(target, nil, 0o644); err != nil {
		return err
	}
	st.createdReadme = true
	s.Out.OK("%s created", projectReadme)
	return nil
}

func (s *Service) commitSpec(src templates.Source, workDir string, st *state) error {
	target := filepath.Join(workDir, projectSpecFile)
	if s.FS.Exists(target) {
		return nil
	}

	data, err := src.ReadFile(projectSpecFile)
	if err != nil {
		return err
	}

	specsDir := filepath.Join(workDir, projectSpecsDir)
	if !s.FS.IsDir(specsDir) {
		if err := s.FS.MkdirAll(specsDir); err != nil {
			return err
		}
		st.createdSpecsDir = true
		s.Out.OK("Created directory: %s/", projectSpecsDir)
	}

	if err := s.FS.WriteFileAtomic(target, data, 0o644); err != nil {
		return err
	}
	st.createdSpec = true
	s.Out.OK("%s created", projectSpecFile)
	return nil
}

func (s *Service) commitPrompts(src templates.Source, workDir string, set resolve.InstallSet, st *state) error {
	if len(set.Prompts) == 0 {
		return nil
	}

	promptsDir := filepath.Join(workDir, projectPromptsDir)
	if !s.FS.IsDir(promptsDir) {
		if err := s.FS.MkdirAll(promptsDir); err != nil {
			return err
		}
		st.createdPromptsDir = true
		s.Out.OK("Created directory: %s/", projectPromptsDir)
	}

	for _, prompt := range set.Prompts {
		target := filepath.Join(promptsDir, prompt)
		if s.FS.Exists(target) {
			continue
		}

		data, err := src.ReadFile(filepath.ToSlash(filepath.Join(projectPromptsDir, prompt)))
		if err != nil {
			return err
		}
		if err := s.FS.WriteFileAtomic(target, data, 0o644); err != nil {
			return err
		}
		st.createdPrompts = append(st.createdPrompts, prompt)
		s.Out.OK("Created prompt: %s", filepath.ToSlash(filepath.Join(projectPromptsDir, prompt)))
	}

	return nil
}

func (s *Service) rollback(workDir string, st *state) error {
	if st.createdReadme {
		_ = os.Remove(filepath.Join(workDir, projectReadme))
	}
	if st.createdSpec {
		_ = os.Remove(filepath.Join(workDir, projectSpecFile))
	}
	if st.createdSpecsDir {
		_ = os.Remove(filepath.Join(workDir, projectSpecsDir))
	}
	for _, prompt := range st.createdPrompts {
		_ = os.Remove(filepath.Join(workDir, projectPromptsDir, prompt))
	}
	if st.createdPromptsDir {
		_ = os.Remove(filepath.Join(workDir, projectPromptsDir))
	}

	if st.replacedAgents {
		target := filepath.Join(workDir, projectAgents)
		backup := filepath.Join(st.backupRoot, projectAgents)
		if s.FS.Exists(backup) {
			_ = os.Remove(target)
			_ = os.Rename(backup, target)
		} else {
			_ = os.Remove(target)
		}
	}

	if st.replacedSkills {
		target := filepath.Join(workDir, projectSkillsDir)
		backup := filepath.Join(st.backupRoot, projectSkillsDir)
		_ = os.RemoveAll(target)
		if s.FS.IsDir(backup) {
			_ = os.Rename(backup, target)
		}
	}

	for _, file := range st.writtenRules {
		target := filepath.Join(workDir, projectRulesDir, file)
		backup := filepath.Join(st.backupRoot, projectRulesDir, file)
		if s.FS.Exists(backup) {
			_ = os.Remove(target)
			_ = os.Rename(backup, target)
		} else {
			_ = os.Remove(target)
		}
	}

	for _, file := range st.removedStale {
		target := filepath.Join(workDir, projectRulesDir, file)
		backup := filepath.Join(st.backupRoot, projectRulesDir, file)
		if s.FS.Exists(backup) {
			_ = os.Rename(backup, target)
		}
	}

	if st.rulesDirCreated {
		_ = os.Remove(filepath.Join(workDir, projectRulesDir))
	}

	return nil
}

func (s *Service) copyTemplateDir(src templates.Source, srcPath, dstPath string) error {
	if err := os.MkdirAll(dstPath, 0o755); err != nil {
		return err
	}

	entries, err := src.ReadDir(srcPath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childSrc := filepath.ToSlash(filepath.Join(srcPath, entry.Name()))
		childDst := filepath.Join(dstPath, entry.Name())
		if entry.IsDir() {
			if err := s.copyTemplateDir(src, childSrc, childDst); err != nil {
				return err
			}
			continue
		}

		data, err := src.ReadFile(childSrc)
		if err != nil {
			return err
		}
		if err := s.FS.WriteFileAtomic(childDst, data, 0o644); err != nil {
			return err
		}
	}

	return nil
}

func (s *Service) backupFile(src, dst string) error {
	if !s.FS.Exists(src) || s.FS.Exists(dst) {
		return nil
	}
	return s.FS.CopyFile(src, dst)
}

func (s *Service) collectAvailableSkillsXMLSkills(stageSkillsDir string, force bool) ([]skills.Skill, error) {
	entries, err := os.ReadDir(stageSkillsDir)
	if err != nil {
		return nil, err
	}

	discovered := make([]skills.Skill, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		skillPath := filepath.Join(stageSkillsDir, entry.Name(), "SKILL.md")
		body, err := os.ReadFile(skillPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}

		fm, err := skills.Validate(filepath.ToSlash(filepath.Join(projectSkillsDir, entry.Name(), "SKILL.md")), body)
		if err != nil {
			if force {
				return nil, err
			}
			continue
		}

		discovered = append(discovered, skills.Skill{
			Name:          fm.Name,
			Description:   fm.Description,
			Compatibility: fm.Compatibility,
			Metadata:      fm.Metadata,
			Location:      filepath.ToSlash(filepath.Join(projectSkillsDir, entry.Name(), "SKILL.md")),
		})
	}

	sort.Slice(discovered, func(i, j int) bool {
		return discovered[i].Location < discovered[j].Location
	})

	if len(discovered) == 0 {
		return nil, fmt.Errorf("no valid skill templates found in %s", projectSkillsDir)
	}

	return discovered, nil
}
