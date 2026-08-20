package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/yassineosip/grit-tester/internal/suite"
)

func TestPrepareExpectFilesSubstitutesPlaceholders(t *testing.T) {
	c := suite.Case{ExpectFiles: map[string]string{
		"{{TARGET}}/out.txt":  "hi",
		"{{CASE_DIR}}/in.txt": "yo",
		"plain.txt":           "ok",
	}}
	target := filepath.Join(t.TempDir(), "impl")
	dir := t.TempDir()
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	expect, cleanup := prepareExpectFiles(c, target, dir)
	defer cleanup()

	if got := expect[filepath.Join(target, "out.txt")]; got != "hi" {
		t.Errorf("{{TARGET}} key = %q, want %q", got, "hi")
	}
	if got := expect[filepath.Join(dir, "in.txt")]; got != "yo" {
		t.Errorf("{{CASE_DIR}} key = %q, want %q", got, "yo")
	}
	if got := expect["plain.txt"]; got != "ok" {
		t.Errorf("plain key = %q, want %q", got, "ok")
	}
}

func TestPrepareExpectFilesCleanupRemovesTargetFiles(t *testing.T) {
	target := t.TempDir()
	f := filepath.Join(target, "out.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	c := suite.Case{ExpectFiles: map[string]string{"{{TARGET}}/out.txt": "x"}}
	_, cleanup := prepareExpectFiles(c, target, t.TempDir())
	cleanup()
	if _, err := os.Stat(f); !os.IsNotExist(err) {
		t.Errorf("target file must be cleaned up, stat err = %v", err)
	}
}

// TestExecuteCasePassesEnv runs a case whose command is this test binary
// itself with -test.run=TestGritEnvHelper: the helper writes a marker file
// only when the env var flowed through executeCase -> runner.Run (the file
// avoids the helper's testing-framework "PASS" line polluting stdout). When
// go test runs the helper directly the env is unset and it stays silent.
func TestExecuteCasePassesEnv(t *testing.T) {
	c := suite.Case{
		ID:      "env",
		Command: os.Args[0],
		Args:    []string{"-test.run=TestGritEnvHelper"},
		Env:     map[string]string{"GRIT_ENV_TEST": "present"},
		ExpectFiles: map[string]string{
			"env.txt": "present",
		},
	}
	ok, diffs, errMsg, _ := executeCase(t.TempDir(), c)
	if !ok {
		t.Fatalf("env case failed: %s: %v", errMsg, diffs)
	}
}

func TestGritEnvHelper(t *testing.T) {
	if os.Getenv("GRIT_ENV_TEST") == "present" {
		if err := os.WriteFile("env.txt", []byte("present"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
