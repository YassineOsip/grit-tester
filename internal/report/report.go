// Package report renders per-case results as an aligned table and computes
// the process exit code. Bonus (required:false) failures show in amber as
// "BONUS FAIL" and do not fail the run unless Strict is set.
package report

import (
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"

	"github.com/yassineosip/grit-tester/internal/runner"
)

const (
	green = "\x1b[32m"
	red   = "\x1b[31m"
	amber = "\x1b[33m"
	reset = "\x1b[0m"
)

// statusWidth is the width of the STATUS column; "BONUS FAIL" is the
// longest possible value.
const statusWidth = 10

// colGap is the number of spaces between table columns.
const colGap = 2

// wrapWidth hard-wraps long content lines so failure blocks stay readable
// on narrow terminals.
const wrapWidth = 90

// Failure carries the structured detail shown under a failing row.
type Failure struct {
	Setup  map[string]string // input files the case created
	Diffs  []runner.Diff     // structured got/want mismatches
	Err    string            // setup/run error (no got/want sides)
	Stderr string            // captured stderr, if any
}

type outcome struct {
	id          string
	description string
	status      string // PASS, FAIL or BONUS FAIL
	color       string // ANSI code, "" when colors are off
	failure     Failure
}

// Reporter accumulates case outcomes and renders them as a table when
// Final is called.
type Reporter struct {
	out         io.Writer
	color       bool
	strict      bool
	rows        []outcome
	passed      int
	failed      int
	bonusFailed int
}

// New creates a Reporter writing to out. When color is enabled on Windows,
// the console is switched to virtual-terminal mode so ANSI codes render.
func New(out io.Writer, color, strict bool) *Reporter {
	r := &Reporter{out: out, color: color, strict: strict}
	if color && runtime.GOOS == "windows" {
		enableVT(out)
	}
	return r
}

// paint wraps s in an ANSI color; a no-op when colors are disabled.
func (r *Reporter) paint(colorCode, s string) string {
	if !r.color {
		return s
	}
	return colorCode + s + reset
}

// Passed records a passing case.
func (r *Reporter) Passed(id, description string) {
	r.passed++
	r.rows = append(r.rows, outcome{id: id, description: description, status: "PASS", color: green})
}

// Failed records a failing case. required:false cases are bonus: they print
// in amber and don't count toward the exit code unless strict.
func (r *Reporter) Failed(id, description string, f Failure, required bool) {
	if !required && !r.strict {
		r.bonusFailed++
		r.rows = append(r.rows, outcome{id: id, description: description, status: "BONUS FAIL", color: amber, failure: f})
		return
	}
	r.failed++
	r.rows = append(r.rows, outcome{id: id, description: description, status: "FAIL", color: red, failure: f})
}

// Final renders the results table and the summary line.
func (r *Reporter) Final() {
	idWidth := len("ID")
	for _, row := range r.rows {
		if len(row.id) > idWidth {
			idWidth = len(row.id)
		}
	}

	fmt.Fprintf(r.out, "%-*s%s%s%s\n",
		idWidth+colGap, "ID",
		padRight("STATUS", statusWidth),
		strings.Repeat(" ", colGap),
		"DESCRIPTION")

	for _, row := range r.rows {
		fmt.Fprintf(r.out, "%-*s%s%s%s\n",
			idWidth+colGap, row.id,
			r.paint(row.color, padRight(row.status, statusWidth)),
			strings.Repeat(" ", colGap),
			row.description)
		if row.status != "PASS" {
			r.renderFailure(row.failure, idWidth)
		}
	}
	fmt.Fprintln(r.out)

	line := "== " + r.Summary() + " =="
	switch {
	case r.failed > 0:
		fmt.Fprintln(r.out, r.paint(red, line))
	case r.bonusFailed > 0:
		fmt.Fprintln(r.out, r.paint(amber, line))
	default:
		fmt.Fprintln(r.out, r.paint(green, line))
	}
}

// renderFailure prints the structured detail under a failing row:
// run/setup errors, stderr, the case input, then every got/want pair.
func (r *Reporter) renderFailure(f Failure, idWidth int) {
	gutter := strings.Repeat(" ", idWidth+colGap+statusWidth+colGap) + "│ "

	if f.Err != "" {
		fmt.Fprintf(r.out, "%s%s\n", gutter, r.paint(red, f.Err))
	}
	if f.Stderr != "" {
		stderr := strings.TrimSpace(f.Stderr)
		if len(stderr) > 400 {
			stderr = stderr[:400] + "..."
		}
		fmt.Fprintf(r.out, "%s%s\n", gutter, "stderr: "+stderr)
	}

	keys := make([]string, 0, len(f.Setup))
	for k := range f.Setup {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		r.renderContent(gutter, "input ("+k+")", "", f.Setup[k])
	}

	diffs := append([]runner.Diff(nil), f.Diffs...)
	sort.Slice(diffs, func(i, j int) bool { return diffs[i].Where < diffs[j].Where })
	for _, d := range diffs {
		got := d.Got
		if d.Missing {
			got = "(missing)"
		}
		r.renderContent(gutter, "got ("+d.Where+")", red, got)
		r.renderContent(gutter, "want", green, d.Want)
	}
}

// renderContent prints a labeled text block: the label on its own line,
// content lines indented under it, and a marker for trailing newlines.
func (r *Reporter) renderContent(gutter, label, colorCode, text string) {
	fmt.Fprintf(r.out, "%s%s\n", gutter, r.paint(colorCode, label))
	for _, line := range wrapContent(text) {
		fmt.Fprintf(r.out, "%s  %s\n", gutter, line)
	}
	if n := trailingNewlines(text); n > 0 {
		if n == 1 {
			fmt.Fprintf(r.out, "%s  \\n\n", gutter)
		} else {
			fmt.Fprintf(r.out, "%s  \\n (x%d)\n", gutter, n)
		}
	}
}

// wrapContent splits text into display lines, hard-wrapping at wrapWidth.
func wrapContent(text string) []string {
	var out []string
	for _, logical := range strings.Split(text, "\n") {
		for len(logical) > wrapWidth {
			out = append(out, logical[:wrapWidth])
			logical = logical[wrapWidth:]
		}
		out = append(out, logical)
	}
	for len(out) > 0 && out[len(out)-1] == "" {
		out = out[:len(out)-1]
	}
	return out
}

// trailingNewlines counts the \n characters at the end of s.
func trailingNewlines(s string) int {
	n := 0
	for i := len(s) - 1; i >= 0 && s[i] == '\n'; i-- {
		n++
	}
	return n
}

// Summary describes the run in one line.
func (r *Reporter) Summary() string {
	s := fmt.Sprintf("%d passed, %d failed", r.passed, r.failed)
	if r.bonusFailed > 0 && !r.strict {
		s += fmt.Sprintf(", %d bonus failed", r.bonusFailed)
	}
	return s
}

// ExitCode returns 1 when any required case failed, else 0.
func (r *Reporter) ExitCode() int {
	if r.failed > 0 {
		return 1
	}
	return 0
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}
