package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestMain turns the test binary into a controllable child process when
// TESTER_HELPER=1, keeping the tests cross-platform (no shell builtins).
func TestMain(m *testing.M) {
	if os.Getenv("TESTER_HELPER") == "1" {
		switch os.Getenv("TESTER_HELPER_MODE") {
		case "hello":
			fmt.Print("hello\n")
			os.Exit(0)
		case "exit7":
			fmt.Print("partial output")
			os.Exit(7)
		case "sleep5":
			time.Sleep(5 * time.Second)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func helperEnv(mode string) []string {
	return []string{"TESTER_HELPER=1", "TESTER_HELPER_MODE=" + mode}
}

func TestRunCapturesStdoutAndExit(t *testing.T) {
	res, err := Run(context.Background(), t.TempDir(), 5*time.Second, helperEnv("hello"), os.Args[0], nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Stdout != "hello\n" {
		t.Errorf("stdout = %q, want %q", res.Stdout, "hello\n")
	}
	if res.Exit != 0 {
		t.Errorf("exit = %d, want 0", res.Exit)
	}
}

func TestRunNonZeroExitIsNotAnError(t *testing.T) {
	res, err := Run(context.Background(), t.TempDir(), 5*time.Second, helperEnv("exit7"), os.Args[0], nil)
	if err != nil {
		t.Fatalf("non-zero exit must not be an error, got %v", err)
	}
	if res.Exit != 7 {
		t.Errorf("exit = %d, want 7", res.Exit)
	}
	if res.Stdout != "partial output" {
		t.Errorf("stdout = %q", res.Stdout)
	}
}

func TestRunTimesOut(t *testing.T) {
	start := time.Now()
	_, err := Run(context.Background(), t.TempDir(), 200*time.Millisecond, helperEnv("sleep5"), os.Args[0], nil)
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("want timeout error, got %v", err)
	}
	if elapsed := time.Since(start); elapsed > 4*time.Second {
		t.Errorf("timeout took %v, too slow", elapsed)
	}
}

func TestRunLaunchFailure(t *testing.T) {
	_, err := Run(context.Background(), t.TempDir(), 5*time.Second, nil, "definitely-not-a-real-binary-xyz", nil)
	if err == nil {
		t.Fatal("want error for missing binary")
	}
}

func TestPrepareDirWritesNestedFiles(t *testing.T) {
	dir, err := PrepareDir(t.TempDir(), map[string]string{"a/b.txt": "hi"})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	data, err := os.ReadFile(filepath.Join(dir, "a", "b.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hi" {
		t.Errorf("got %q, want %q", data, "hi")
	}
}

func TestPrepareDirRejectsEscape(t *testing.T) {
	for _, bad := range []string{"../evil.txt", "a/../../evil.txt"} {
		if _, err := PrepareDir(t.TempDir(), map[string]string{bad: "x"}); err == nil {
			t.Errorf("want error for escaping path %q", bad)
		}
	}
}
