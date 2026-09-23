//go:build !windows

package report

import "io"

// enableVT is a no-op on non-Windows platforms: terminals there already
// interpret ANSI escape sequences natively.
func enableVT(out io.Writer) {}
