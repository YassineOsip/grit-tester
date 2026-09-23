package report

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

// spinnerFrames are braille glyphs cycled at ~12 fps for a smooth spin.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// Progress draws a single live progress line while cases execute.
// Callers create one only for terminal output; redirected output should
// not use it, keeping piped/captured output clean.
type Progress struct {
	out io.Writer

	mu      sync.Mutex
	total   int
	done    int
	failed  int
	current string
	stopped bool

	ticker *time.Ticker
	frame  int
}

// NewProgress returns a Progress tracking total cases.
func NewProgress(out io.Writer, total int) *Progress {
	return &Progress{out: out, total: total}
}

// Spin redraws the progress line until Finish is called. Run it in a
// goroutine; it stops when the ticker stops.
func (p *Progress) Spin() {
	p.ticker = time.NewTicker(80 * time.Millisecond)
	for range p.ticker.C {
		p.mu.Lock()
		if p.stopped {
			p.mu.Unlock()
			return
		}
		p.frame++
		p.render()
		p.mu.Unlock()
	}
}

// Started records the case currently executing.
func (p *Progress) Started(id string) {
	p.mu.Lock()
	p.current = id
	p.render()
	p.mu.Unlock()
}

// Done records a finished case; ok=false means it failed.
func (p *Progress) Done(ok bool) {
	p.mu.Lock()
	p.done++
	if !ok {
		p.failed++
	}
	p.render()
	p.mu.Unlock()
}

// Finish stops the spinner and clears the progress line so the table
// starts on a fresh line.
func (p *Progress) Finish() {
	p.mu.Lock()
	if !p.stopped {
		p.stopped = true
		p.clear()
	}
	p.mu.Unlock()
	if p.ticker != nil {
		p.ticker.Stop()
	}
}

// render draws one frame; the caller holds p.mu.
func (p *Progress) render() {
	line := fmt.Sprintf("%s testing %s [%d/%d]", spinnerFrames[p.frame%len(spinnerFrames)], p.current, p.done, p.total)
	if p.failed > 0 {
		line += fmt.Sprintf(" · %d failed", p.failed)
	}
	fmt.Fprintf(p.out, "\r%-80s", line)
}

// clear erases the progress line; the caller holds p.mu.
func (p *Progress) clear() {
	fmt.Fprintf(p.out, "\r%-80s\r", strings.Repeat(" ", 0))
}
