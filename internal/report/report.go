// Package report renders per-case results in color and computes the
// process exit code. Bonus (required:false) failures show in amber as
// "BONUS FAIL" and do not fail the run unless Strict is set.
package report

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"syscall"
	"unsafe"
)

const (
	green = "\x1b[32m"
	red   = "\x1b[31m"
	amber = "\x1b[33m"
	reset = "\x1b[0m"
)

// Reporter accumulates case outcomes and prints them as they arrive.
type Reporter struct {
	out         io.Writer
	color       bool
	strict      bool
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
	fmt.Fprintf(r.out, "%s  PASS  %s\n", r.paint(green, "\u2713"), withDesc(id, description))
}

// Failed records a failing case. required:false cases are bonus: they print
// in amber and don't count toward the exit code unless strict.
func (r *Reporter) Failed(id, description string, mismatches []string, required bool) {
	switch {
	case !required && !r.strict:
		r.bonusFailed++
		fmt.Fprintf(r.out, "%s  BONUS FAIL  %s\n", r.paint(amber, "!"), withDesc(id, description))
	default:
		r.failed++
		fmt.Fprintf(r.out, "%s  FAIL  %s\n", r.paint(red, "\u2717"), withDesc(id, description))
	}
	for _, m := range mismatches {
		fmt.Fprintf(r.out, "          %s\n", m)
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

// Final prints the summary line.
func (r *Reporter) Final() {
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

// ExitCode returns 1 when any required case failed, else 0.
func (r *Reporter) ExitCode() int {
	if r.failed > 0 {
		return 1
	}
	return 0
}

func withDesc(id, description string) string {
	if description == "" {
		return id
	}
	return id + " \u2014 " + description
}

// enableVT switches the given console to virtual-terminal processing so
// ANSI escape sequences render on Windows 10+. Pure stdlib (syscall);
// failures are ignored — colors simply degrade.
func enableVT(out io.Writer) {
	f, ok := out.(*os.File)
	if !ok {
		return
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	getMode := kernel32.NewProc("GetConsoleMode")
	setMode := kernel32.NewProc("SetConsoleMode")
	const enableVirtualTerminalProcessing = 0x0004

	var mode uint32
	if rc, _, _ := getMode.Call(f.Fd(), uintptr(unsafe.Pointer(&mode))); rc == 0 {
		return
	}
	_, _, _ = setMode.Call(f.Fd(), uintptr(mode|enableVirtualTerminalProcessing))
}
