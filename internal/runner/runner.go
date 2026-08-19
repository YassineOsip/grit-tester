// Package runner executes one test case's command in an isolated temp
// directory and captures its output.
package runner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Result is the outcome of running one command.
type Result struct {
	Stdout string
	Stderr string
	Exit   int
}

// Run executes command+args with cwd=dir, inheriting the environment plus
// env, and kills the process after timeout. A non-zero exit code is NOT an
// error (it is reported in Result.Exit); only timeouts and launch failures
// return an error.
func Run(ctx context.Context, dir string, timeout time.Duration, env []string, command string, args []string) (Result, error) {
	cctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(cctx, command, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	res := Result{}
	err := cmd.Run()
	res.Stdout, res.Stderr = stdout.String(), stderr.String()

	if err == nil {
		return res, nil
	}
	if cctx.Err() == context.DeadlineExceeded {
		return res, fmt.Errorf("timeout after %s", timeout)
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		res.Exit = exitErr.ExitCode()
		return res, nil // ran to completion, just with a non-zero exit
	}
	return res, err // launch failure (command not found, ...)
}

// PrepareDir creates a temp dir under parent (or the system temp dir when
// parent is empty) and writes the setup files into it. Setup paths are
// relative and may not escape the temp dir.
func PrepareDir(parent string, setup map[string]string) (string, error) {
	dir, err := os.MkdirTemp(parent, "grit-case-")
	if err != nil {
		return "", err
	}
	for rel, content := range setup {
		if err := writeSetupFile(dir, rel, content); err != nil {
			os.RemoveAll(dir)
			return "", err
		}
	}
	return dir, nil
}

// writeSetupFile writes one setup file under dir, refusing to escape it.
func writeSetupFile(dir, rel, content string) error {
	if rel == "" || filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return fmt.Errorf("setup path must be a safe relative path: %q", rel)
	}
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return err
	}
	return os.WriteFile(full, []byte(content), 0o644)
}
