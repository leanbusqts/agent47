package cli

import (
	"bytes"
	"testing"
)

func TestOutputWritesExpectedPrefixes(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	out := NewOutput(&stdout, &stderr)

	out.Printf("plain %s", "text")
	out.Info("info")
	out.Warn("warn")
	out.OK("ok")
	out.Err("err")

	if stdout.String() != "plain text[INFO] info\n[WARN] warn\n[OK] ok\n" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if stderr.String() != "[ERR] err\n" {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
