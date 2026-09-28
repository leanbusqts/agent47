package main

import (
	"bytes"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSelectScenarios(t *testing.T) {
	all := []scenario{{name: "one"}, {name: "two"}}
	selected, err := selectScenarios(all, []string{"two"})
	if err != nil || len(selected) != 1 || selected[0].name != "two" {
		t.Fatalf("unexpected selection: %v %v", selected, err)
	}
	if _, err := selectScenarios(all, []string{"missing"}); err == nil {
		t.Fatal("expected unknown scenario error")
	}
}

func TestPowerShellArgs(t *testing.T) {
	got := toPowerShellArgs([]string{"--force", "--non-interactive", "extra"})
	want := []string{"-Force", "-NonInteractive", "extra"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestRunReportsRepoDetectionFailure(t *testing.T) {
	originalDetect := afsverifyDetectRepoRoot
	originalOut, originalErr := afsverifyStdout, afsverifyStderr
	t.Cleanup(func() {
		afsverifyDetectRepoRoot = originalDetect
		afsverifyStdout, afsverifyStderr = originalOut, originalErr
	})
	afsverifyDetectRepoRoot = func() (string, error) { return "", errors.New("boom") }
	var stdout, stderr bytes.Buffer
	afsverifyStdout, afsverifyStderr = &stdout, &stderr
	if status := run(nil); status != 1 {
		t.Fatalf("expected status 1, got %d", status)
	}
	if !strings.Contains(stderr.String(), "boom") {
		t.Fatalf("unexpected stderr: %s", stderr.String())
	}
}
