package syscmd

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

type Runner interface {
	Run(ctx context.Context, name string, args ...string) (Result, error)
}

type ExecRunner struct{}

func New() ExecRunner {
	return ExecRunner{}
}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) (Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := Result{
		Stdout: strings.TrimRight(stdout.String(), "\n"),
		Stderr: strings.TrimRight(stderr.String(), "\n"),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil {
		return result, fmt.Errorf("run %s %s: %w", name, strings.Join(args, " "), err)
	}
	return result, nil
}
