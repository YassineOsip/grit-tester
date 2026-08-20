// Command tester runs declarative JSON test suites against an
// implementation of a 01-edu project.
//
// Usage:
//
//	tester validate <cases.json>
//	tester run --suite <name> --target <dir> [-j N] [--strict] [--no-color]
//	tester run --cases <cases.json> --target <dir> [...]
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/yassineosip/grit-tester/internal/report"
	"github.com/yassineosip/grit-tester/internal/runner"
	"github.com/yassineosip/grit-tester/internal/suite"
	"github.com/yassineosip/grit-tester/suites"
)

const usage = `tester — run test suites against 01-edu project implementations

Usage:
  tester validate <cases.json>
  tester run --suite <name> --target <dir> [-j N] [--strict] [--no-color]
  tester run --cases <cases.json> --target <dir> [-j N] [--strict] [--no-color]
`

func main() {
	os.Exit(cli(os.Args[1:]))
}

func cli(args []string) int {
	if len(args) >= 1 && args[0] == "validate" {
		return validate(args[1:])
	}
	// "tester run" is the default command; the word "run" is optional.
	if len(args) >= 1 && args[0] == "run" {
		args = args[1:]
	}
	return runCases(args)
}

func validate(args []string) int {
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "usage: tester validate <cases.json>")
		return 2
	}
	s, err := suite.Load(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, "INVALID:", err)
		return 1
	}
	fmt.Printf("valid: suite %q, %d case(s)\n", s.Suite, len(s.Cases))
	return 0
}

func runCases(args []string) int {
	fs := flag.NewFlagSet("tester run", flag.ContinueOnError)
	suiteName := fs.String("suite", "", "suite name (built-in, or suites/<name>/cases.json)")
	casesPath := fs.String("cases", "", "path to a cases.json (alternative to --suite)")
	target := fs.String("target", ".", "path to the implementation under test")
	jobs := fs.Int("j", runtime.GOMAXPROCS(0), "run N cases concurrently")
	strict := fs.Bool("strict", false, "bonus (required:false) failures fail the run")
	noColor := fs.Bool("no-color", false, "disable colored output")
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }

	if err := fs.Parse(args); err != nil {
		return 2
	}

	var s *suite.Suite
	var err error
	switch {
	case *casesPath != "":
		s, err = suite.Load(*casesPath)
	case *suiteName != "":
		s, err = loadSuite(*suiteName)
	default:
		fmt.Fprintln(os.Stderr, "error: provide --suite or --cases")
		return 2
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	absTarget, err := filepath.Abs(*target)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}

	rep := report.New(os.Stdout, !*noColor, *strict)
	var live *report.Progress
	if isTTY(os.Stdout) {
		live = report.NewProgress(os.Stdout, len(s.Cases))
		go live.Spin()
	}

	start := time.Now()
	runSuite(rep, live, s, absTarget, *jobs)

	rep.Final()
	fmt.Printf("suite: %s (%d cases) in %s\n", s.Suite, len(s.Cases), time.Since(start).Round(time.Millisecond))
	return rep.ExitCode()
}

// isTTY reports whether f is attached to a terminal. Piped or redirected
// output returns false, so live progress never pollutes captured output.
func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// loadSuite resolves a suite by name. A local suites/<name>/cases.json
// file wins — contributors can test new suites without rebuilding — and
// the suites embedded in the binary are the fallback, so the installed
// tool works from any directory.
func loadSuite(name string) (*suite.Suite, error) {
	local := filepath.Join("suites", name, "cases.json")
	if _, err := os.Stat(local); err == nil {
		return suite.Load(local)
	}
	data, err := suites.Files.ReadFile(name + "/cases.json")
	if err != nil {
		return nil, fmt.Errorf("suite %q not found (looked for %s and for an embedded suite of that name)", name, local)
	}
	return suite.Parse(name, data)
}

// prepareExpectFiles substitutes {{TARGET}} and {{CASE_DIR}} in
// expect_files keys and returns the resolved map plus a cleanup function.
// Files the case was expected to create inside the implementation
// directory are removed after comparison so suites leave the target
// untouched.
func prepareExpectFiles(c suite.Case, target, dir string) (map[string]string, func()) {
	expect := make(map[string]string, len(c.ExpectFiles))
	var created []string
	for k, v := range c.ExpectFiles {
		k = strings.ReplaceAll(k, "{{TARGET}}", target)
		k = strings.ReplaceAll(k, "{{CASE_DIR}}", dir)
		k = filepath.FromSlash(k)
		if filepath.IsAbs(k) {
			created = append(created, k)
		}
		expect[k] = v
	}
	return expect, func() {
		for _, f := range created {
			os.Remove(f)
		}
	}
}

// runSuite executes every case on a worker pool and reports in case order,
// so output is stable regardless of how the cases finish. When live is
// non-nil, worker activity feeds the progress line.
func runSuite(rep *report.Reporter, live *report.Progress, s *suite.Suite, target string, jobs int) {
	if jobs < 1 {
		jobs = 1
	}

	type outcome struct {
		ok     bool
		diffs  []runner.Diff
		errMsg string
		stderr string
	}
	results := make([]outcome, len(s.Cases))

	casesCh := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < jobs; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range casesCh {
				if live != nil {
					live.Started(s.Cases[i].ID)
				}
				ok, diffs, errMsg, stderr := executeCase(target, s.Cases[i])
				results[i] = outcome{ok: ok, diffs: diffs, errMsg: errMsg, stderr: stderr}
				if live != nil {
					live.Done(ok)
				}
			}
		}()
	}
	for i := range s.Cases {
		casesCh <- i
	}
	close(casesCh)
	wg.Wait()

	if live != nil {
		live.Finish()
	}

	for i := range results {
		c := s.Cases[i]
		if results[i].ok {
			rep.Passed(c.ID, c.Description)
		} else {
			rep.Failed(c.ID, c.Description, report.Failure{
				Setup:  c.Setup,
				Diffs:  results[i].diffs,
				Err:    results[i].errMsg,
				Stderr: results[i].stderr,
			}, c.Required)
		}
	}
}

// executeCase prepares an isolated temp dir, runs the case's command and
// applies every expectation. Returns ok, the structured diffs, a setup/run
// error message (if any) and captured stderr.
func executeCase(target string, c suite.Case) (bool, []runner.Diff, string, string) {
	dir, err := runner.PrepareDir("", c.Setup)
	if err != nil {
		return false, nil, "setup: " + err.Error(), ""
	}
	defer os.RemoveAll(dir)

	workdir := strings.ReplaceAll(c.Workdir, "{{TARGET}}", target)
	workdir = strings.ReplaceAll(workdir, "{{CASE_DIR}}", dir)
	if workdir == "" {
		workdir = dir
	}
	command := strings.ReplaceAll(c.Command, "{{TARGET}}", target)
	command = strings.ReplaceAll(command, "{{CASE_DIR}}", dir)
	args := make([]string, len(c.Args))
	for i, a := range c.Args {
		a = strings.ReplaceAll(a, "{{TARGET}}", target)
		args[i] = strings.ReplaceAll(a, "{{CASE_DIR}}", dir)
	}

	res, err := runner.Run(context.Background(), workdir, c.Timeout(), nil, command, args)
	if err != nil {
		return false, nil, "run: " + err.Error(), ""
	}

	var diffs []runner.Diff
	if c.ExpectExit != nil && res.Exit != *c.ExpectExit {
		diffs = append(diffs, runner.Diff{Where: "exit code", Got: fmt.Sprint(res.Exit), Want: fmt.Sprint(*c.ExpectExit)})
	}
	expect, cleanup := prepareExpectFiles(c, target, dir)
	defer cleanup()
	diffs = append(diffs, runner.CompareFiles(dir, expect)...)
	diffs = append(diffs, runner.CompareStdout(res.Stdout, c.ExpectStdout)...)
	if len(diffs) > 0 {
		return false, diffs, "", res.Stderr
	}
	return true, nil, "", ""
}
