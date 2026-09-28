package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/leanbusqts/agent47/internal/cli"
	"github.com/leanbusqts/agent47/internal/doctor"
	"github.com/leanbusqts/agent47/internal/runtime"
)

func (r *Root) runDoctor(ctx context.Context, cfg runtime.Config, args []string) int {
	var opts doctor.Options
	jsonOutput := false

	for _, arg := range args {
		switch arg {
		case "--json":
			jsonOutput = true
		case "--check-update":
			opts.CheckUpdate = true
		case "--check-update-force":
			opts.CheckUpdate = true
			opts.ForceUpdate = true
		case "--fail-on-warn":
			opts.FailOnWarn = true
		default:
			r.out.Diagnosticf("Usage: afs doctor [--json] [--check-update|--check-update-force|--fail-on-warn]\n")
			return 2
		}
	}
	if jsonOutput {
		return r.runDoctorJSON(ctx, cfg, opts)
	}

	service, err := doctor.New(cfg, r.out)
	if err != nil {
		r.out.Err("Failed to initialize doctor service: %v", err)
		return 1
	}
	if err := service.Run(ctx, cfg, opts); err != nil {
		r.out.Err("%v", err)
		return 1
	}
	return 0
}

type doctorJSONReport struct {
	SchemaVersion int      `json:"schema_version"`
	Status        string   `json:"status"`
	Checks        []string `json:"checks"`
	Warnings      []string `json:"warnings"`
	Error         string   `json:"error,omitempty"`
}

func (r *Root) runDoctorJSON(ctx context.Context, cfg runtime.Config, opts doctor.Options) int {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	service, err := doctor.New(cfg, cli.NewOutput(&stdout, &stderr))
	if err == nil {
		err = service.Run(ctx, cfg, opts)
	}

	report := doctorJSONReport{
		SchemaVersion: 1,
		Status:        "ok",
		Checks:        outputLines(stdout.String()),
		Warnings:      warningLines(stderr.String()),
	}
	status := 0
	if len(report.Warnings) > 0 {
		report.Status = "warning"
	}
	if err != nil {
		status = 1
		var warningsErr doctor.WarningsError
		if !errors.As(err, &warningsErr) {
			report.Status = "error"
			report.Error = err.Error()
		}
	}
	data, marshalErr := json.MarshalIndent(report, "", "  ")
	if marshalErr != nil {
		r.out.Err("Failed to encode doctor JSON: %v", marshalErr)
		return 1
	}
	r.out.Printf("%s\n", data)
	return status
}

func outputLines(raw string) []string {
	lines := []string{}
	for _, line := range strings.Split(raw, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func warningLines(raw string) []string {
	lines := outputLines(raw)
	for index := range lines {
		lines[index] = strings.TrimSpace(strings.TrimPrefix(lines[index], "[WARN]"))
	}
	return lines
}
