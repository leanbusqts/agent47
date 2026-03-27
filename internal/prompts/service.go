package prompts

import (
	"bytes"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/fsx"
	"github.com/leanbusqts/agent47/internal/runtime"
	"github.com/leanbusqts/agent47/internal/templates"
)

type Service struct {
	FS     fsx.Service
	Loader *templates.Loader
	Out    cli.Output
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

func (s *Service) AddAgentPrompt(workDir string, force bool) error {
	s.Out.Info("Initializing agent prompt...")

	data, err := s.Loader.Source.ReadFile("prompts/agent-prompt.txt")
	if err != nil {
		return fmt.Errorf("Template not found: agent-prompt.txt")
	}

	promptsDir := filepath.Join(workDir, "prompts")
	target := filepath.Join(promptsDir, "agent-prompt.txt")
	existed := s.FS.Exists(target)
	if existed && !force {
		s.Out.Warn("%s already exists, skipping", filepath.ToSlash(target))
		return nil
	}

	if !s.FS.IsDir(promptsDir) {
		if err := s.FS.MkdirAll(promptsDir); err != nil {
			return err
		}
		s.Out.OK("Created directory: prompts/")
	}

	if err := s.FS.WriteFileAtomic(target, data, 0o644); err != nil {
		return err
	}

	if force && existed {
		s.Out.OK("Updated prompt: prompts/agent-prompt.txt")
	} else {
		s.Out.OK("Created prompt: prompts/agent-prompt.txt")
	}
	s.Out.OK("Agent prompt setup complete.")
	return nil
}

func (s *Service) AddSSPrompt() error {
	data, err := s.Loader.Source.ReadFile("prompts/ss-prompt.txt")
	if err != nil {
		return fmt.Errorf("Template not found: ss-prompt.txt")
	}

	for _, candidate := range clipboardCommands {
		toolPath, lookErr := exec.LookPath(candidate.name)
		if lookErr != nil {
			continue
		}

		cmd := exec.Command(toolPath, candidate.args...)
		cmd.Stdin = bytes.NewReader(data)
		if runErr := cmd.Run(); runErr == nil {
			s.Out.OK("Snapshot/spec prompt copied to clipboard")
			return nil
		}
	}

	s.Out.Printf("%s\n", string(data))
	return nil
}

type clipboardCommand struct {
	name string
	args []string
}

var clipboardCommands = []clipboardCommand{
	{name: "pbcopy"},
	{name: "wl-copy"},
	{name: "xclip", args: []string{"-selection", "clipboard"}},
	{name: "clip"},
}
