package cli

import (
	"fmt"
	"io"
)

type Output struct {
	stdout io.Writer
	stderr io.Writer
}

func NewOutput(stdout, stderr io.Writer) Output {
	return Output{
		stdout: stdout,
		stderr: stderr,
	}
}

func (o Output) Printf(format string, args ...any) {
	fmt.Fprintf(o.stdout, format, args...)
}

func (o Output) Info(format string, args ...any) {
	fmt.Fprintf(o.stdout, "[INFO] "+format+"\n", args...)
}

func (o Output) Warn(format string, args ...any) {
	fmt.Fprintf(o.stdout, "[WARN] "+format+"\n", args...)
}

func (o Output) OK(format string, args ...any) {
	fmt.Fprintf(o.stdout, "[OK] "+format+"\n", args...)
}

func (o Output) Err(format string, args ...any) {
	fmt.Fprintf(o.stderr, "[ERR] "+format+"\n", args...)
}
