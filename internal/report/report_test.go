package report

import (
	"strings"
	"testing"
)

func TestReporterPlainOutput(t *testing.T) {
	var b strings.Builder
	r := New(&b, false, false)
	r.Passed("A1", "desc one")
	r.Failed("C7", "desc two", []string{`result.txt: got "a", want "b"`}, true)
	r.Final()

	out := b.String()
	for _, want := range []string{"PASS  A1 \u2014 desc one", "FAIL  C7 \u2014 desc two", "result.txt", "1 passed, 1 failed"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}
	if r.ExitCode() != 1 {
		t.Errorf("exit = %d, want 1", r.ExitCode())
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
	if !strings.Contains(out, "BONUS FAIL  C1") {
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
	if !strings.Contains(b.String(), "FAIL  C1") {
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
