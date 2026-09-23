package report

import (
	"strings"
	"testing"
)

func TestProgressRendersAndClears(t *testing.T) {
	var b strings.Builder
	p := NewProgress(&b, 82)
	p.Started("C7")
	p.Done(false)

	out := b.String()
	for _, want := range []string{"testing C7", "[1/82]", "1 failed", "\r"} {
		if !strings.Contains(out, want) {
			t.Errorf("progress missing %q:\n%q", want, out)
		}
	}
	if strings.Contains(out, "\x1b[") {
		t.Errorf("progress must not contain ANSI codes: %q", out)
	}

	p.Finish()
	if !strings.HasSuffix(b.String(), "\r") {
		t.Errorf("finish must clear the line with a carriage return: %q", b.String())
	}
}

func TestProgressCounts(t *testing.T) {
	var b strings.Builder
	p := NewProgress(&b, 3)
	p.Started("A1")
	p.Done(true)
	p.Started("A2")
	p.Done(false)
	p.Started("A3")
	p.Done(true)
	p.Finish()

	out := b.String()
	if !strings.Contains(out, "[3/3]") {
		t.Errorf("want final count 3/3:\n%q", out)
	}
	if !strings.Contains(out, "1 failed") {
		t.Errorf("want 1 failed:\n%q", out)
	}
}
