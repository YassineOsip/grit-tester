// Package report renders per-case results as an aligned table and computes
// the process exit code. Bonus (required:false) failures show in amber as
// "BONUS FAIL" and do not fail the run unless Strict is set.
package report

import (
	"fmt"
	"io"
	"runtime"
	"strings"
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

type outcome struct {
	id          string
	description string
	status      string // PASS, FAIL or BONUS FAIL
	color       string // ANSI code, "" when colors are off
	mismatches  []string
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
func (r *Reporter) Failed(id, description string, mismatches []string, required bool) {
	if !required && !r.strict {
		r.bonusFailed++
		r.rows = append(r.rows, outcome{id: id, description: description, status: "BONUS FAIL", color: amber, mismatches: mismatches})
		return
	}
	r.failed++
	r.rows = append(r.rows, outcome{id: id, description: description, status: "FAIL", color: red, mismatches: mismatches})
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
		indent := strings.Repeat(" ", idWidth+colGap+statusWidth+colGap)
		for _, m := range row.mismatches {
			fmt.Fprintf(r.out, "%s%s\n", indent, m)
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
