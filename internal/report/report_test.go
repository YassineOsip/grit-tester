package report

import (
	"strings"
	"testing"

	"github.com/yassineosip/grit-tester/internal/runner"
)

func lines(out string) []string {
	return strings.Split(strings.TrimRight(out, "\n"), "\n")
}

func TestReporterPlainOutput(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Passed("A1", "desc one")
	r.Failed("C7", "desc two", Failure{
		Setup: map[string]string{"in.txt": "hello"},
		Diffs: []runner.Diff{{Where: "result.txt", Got: "a", Want: "b"}},
	}, true)
	r.Final()

	out := b.String()
	for _, want := range []string{
		"ID", "STATUS", "DESCRIPTION",
		"A1", "PASS", "desc one",
		"C7", "FAIL", "desc two",
		"input (in.txt)", "got (result.txt)", "want",
		"1 passed, 1 failed",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if r.ExitCode() != 1 {
		t.Errorf("exit = %d, want 1", r.ExitCode())
	}
}

func TestReporterFailureBlockIsStructured(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Failed("A1", "fail", Failure{
		Setup: map[string]string{"sample.txt": "hello\n"},
		Diffs: []runner.Diff{{Where: "result.txt", Got: "hello\n", Want: "hi\n"}},
	}, true)
	r.Final()

	out := b.String()
	// input, got and want each get their own labeled block, in order.
	iInput := strings.Index(out, "input (sample.txt)")
	iGot := strings.Index(out, "got (result.txt)")
	iWant := strings.Index(out, "want")
	if iInput < 0 || iGot < 0 || iWant < 0 {
		t.Fatalf("missing labeled blocks:\n%s", out)
	}
	if !(iInput < iGot && iGot < iWant) {
		t.Errorf("block order wrong (input %d, got %d, want %d):\n%s", iInput, iGot, iWant, out)
	}
	if !strings.Contains(out, `  \n`) {
		t.Errorf("trailing newline marker missing:\n%s", out)
	}
}

func TestReporterColumnsAligned(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Passed("A1", "short")
	r.Failed("LONG-ID-42", "fail", Failure{}, true)
	r.Final()

	ls := lines(b.String())
	if len(ls) < 3 {
		t.Fatalf("want header + 2 rows, got:\n%s", b.String())
	}
	row1, row2 := ls[1], ls[2]
	if got := strings.Index(row1, "PASS"); got != strings.Index(row2, "FAIL") || got < 0 {
		t.Errorf("status column misaligned:\n%s\n%s", row1, row2)
	}
	if got := strings.Index(row1, "short"); got != strings.Index(row2, "fail") || got < 0 {
		t.Errorf("description column misaligned:\n%s\n%s", row1, row2)
	}
}

func TestReporterDetailsIndentedUnderDescription(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Passed("A1", "short")
	r.Failed("B2", "a fail", Failure{
		Diffs: []runner.Diff{{Where: "stdout", Got: "x", Want: "y"}},
	}, true)
	r.Final()

	ls := lines(b.String())
	row, detail := ls[2], ls[3]
	descCol := strings.Index(row, "a fail")
	// "│" is a 3-byte UTF-8 rune, so the label follows at descCol + len("│ ").
	if !strings.HasPrefix(detail, strings.Repeat(" ", descCol)+"│ got (stdout)") {
		t.Errorf("detail not aligned under description (col %d):\n%s\n%s", descCol, row, detail)
	}
}

func TestReporterMissingFileMarked(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Failed("C1", "fail", Failure{
		Diffs: []runner.Diff{{Where: "result.txt", Want: "hi", Missing: true}},
	}, true)
	r.Final()

	if !strings.Contains(b.String(), "(missing)") {
		t.Errorf("missing marker absent:\n%s", b.String())
	}
}

func TestReporterErrAndStderrShown(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Failed("C1", "fail", Failure{Err: "run: timeout", Stderr: "boom\n"}, true)
	r.Final()

	out := b.String()
	if !strings.Contains(out, "run: timeout") {
		t.Errorf("error line missing:\n%s", out)
	}
	if !strings.Contains(out, "stderr: boom") {
		t.Errorf("stderr line missing:\n%s", out)
	}
}

func TestReporterBonusDoesNotFail(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Failed("C1", "bonus", Failure{}, false)
	r.Final()

	if r.ExitCode() != 0 {
		t.Errorf("exit = %d, want 0 for bonus-only failure", r.ExitCode())
	}
	out := b.String()
	if !strings.Contains(out, "BONUS FAIL") {
		t.Errorf("missing BONUS FAIL:\n%s", out)
	}
	if !strings.Contains(out, "1 bonus failed") {
		t.Errorf("missing bonus summary:\n%s", out)
	}
}

func TestReporterStrictCountsBonus(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, true)
	r.Failed("C1", "bonus", Failure{}, false)
	r.Final()

	if r.ExitCode() != 1 {
		t.Errorf("strict exit = %d, want 1", r.ExitCode())
	}
	if !strings.Contains(b.String(), "FAIL") {
		t.Errorf("strict should print FAIL:\n%s", b.String())
	}
	if !strings.Contains(b.String(), "1 failed") {
		t.Errorf("strict summary wrong:\n%s", b.String())
	}
}

func TestReporterAllGreen(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Passed("A1", "")
	r.Passed("A2", "")
	r.Final()

	if r.ExitCode() != 0 {
		t.Errorf("exit = %d, want 0", r.ExitCode())
	}
	if !strings.Contains(b.String(), "2 passed, 0 failed") {
		t.Errorf("summary wrong:\n%s", b.String())
	}
}

func TestReporterNoColorOmitsANSI(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Passed("A1", "")
	r.Failed("A2", "", Failure{Err: "boom"}, true)
	r.Final()
	if strings.Contains(b.String(), "\x1b[") {
		t.Errorf("no-color output contains ANSI codes: %q", b.String())
	}
}
