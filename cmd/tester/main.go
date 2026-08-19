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

	"github.com/yassineosip/grit-tester/internal/report"
	"github.com/yassineosip/grit-tester/internal/runner"
	"github.com/yassineosip/grit-tester/internal/suite"
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
	suiteName := fs.String("suite", "", "suite name (suites/<name>/cases.json)")
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
		s, err = suite.Load(filepath.Join("suites", *suiteName, "cases.json"))
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
	runSuite(rep, s, absTarget, *jobs)

	rep.Final()
	fmt.Printf("suite: %s (%d cases)\n", s.Suite, len(s.Cases))
	return rep.ExitCode()
}

// runSuite executes every case on a worker pool and reports in case order,
// so output is stable regardless of how the cases finish.
func runSuite(rep *report.Reporter, s *suite.Suite, target string, jobs int) {
	if jobs < 1 {
		jobs = 1
	}

	type outcome struct {
		ok         bool
		mismatches []string
	}
	results := make([]outcome, len(s.Cases))

	casesCh := make(chan int)
	var wg sync.WaitGroup
	for w := 0; w < jobs; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range casesCh {
				ok, mm := executeCase(target, s.Cases[i])
				results[i] = outcome{ok: ok, mismatches: mm}
			}
		}()
	}
	for i := range s.Cases {
		casesCh <- i
	}
	close(casesCh)
	wg.Wait()

	for i := range results {
		c := s.Cases[i]
		if results[i].ok {
			rep.Passed(c.ID, c.Description)
		} else {
			rep.Failed(c.ID, c.Description, results[i].mismatches, c.Required)
		}
	}
}

// executeCase prepares an isolated temp dir, runs the case's command and
// applies every expectation. Returns ok plus the list of mismatches.
func executeCase(target string, c suite.Case) (bool, []string) {
	dir, err := runner.PrepareDir("", c.Setup)
	if err != nil {
		return false, []string{"setup: " + err.Error()}
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
		return false, []string{"run: " + err.Error()}
	}

	var mismatches []string
	if c.ExpectExit != nil && res.Exit != *c.ExpectExit {
		mismatches = append(mismatches, fmt.Sprintf("exit code: got %d, want %d", res.Exit, *c.ExpectExit))
	}
	if cr := runner.CompareFiles(dir, c.ExpectFiles); cr != nil {
		mismatches = append(mismatches, cr.Mismatches...)
	}
	if cr := runner.CompareStdout(res.Stdout, c.ExpectStdout); cr != nil {
		mismatches = append(mismatches, cr.Mismatches...)
	}
	if len(mismatches) > 0 && res.Stderr != "" {
		stderr := res.Stderr
		if len(stderr) > 400 {
			stderr = stderr[:400] + "..."
		}
		mismatches = append(mismatches, "stderr: "+strings.TrimSpace(stderr))
	}
	return len(mismatches) == 0, mismatches
}
