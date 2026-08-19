package report

import (
	"strings"
	"testing"
)

func lines(out string) []string {
	return strings.Split(strings.TrimRight(out, "\n"), "\n")
}

func TestReporterPlainOutput(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Passed("A1", "desc one")
	r.Failed("C7", "desc two", []string{`result.txt: got "a", want "b"`}, true)
	r.Final()

	out := b.String()
	for _, want := range []string{"ID", "STATUS", "DESCRIPTION", "A1", "PASS", "desc one", "C7", "FAIL", "desc two", "result.txt", "1 passed, 1 failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if r.ExitCode() != 1 {
		t.Errorf("exit = %d, want 1", r.ExitCode())
	}
}

func TestReporterColumnsAligned(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Passed("A1", "short")
	r.Failed("LONG-ID-42", "fail", nil, true)
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

func TestReporterMismatchesIndentedUnderDescription(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Passed("A1", "short")
	r.Failed("B2", "a fail", []string{"stdout: got \"x\", want \"y\""}, true)
	r.Final()

	ls := lines(b.String())
	row, mm := ls[2], ls[3]
	descCol := strings.Index(row, "a fail")
	if got := strings.Index(mm, "stdout"); got != descCol || got < 0 {
		t.Errorf("mismatch not aligned under description:\n%s\n%s", row, mm)
	}
}

func TestReporterBonusDoesNotFail(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Failed("C1", "bonus", nil, false)
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
	r.Failed("C1", "bonus", nil, false)
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
	r.Final()
	if strings.Contains(b.String(), "\x1b[") {
		t.Errorf("no-color output contains ANSI codes: %q", b.String())
	}
}
