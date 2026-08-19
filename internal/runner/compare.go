package runner

import (
	"fmt"
	"os"
	"path/filepath"
)

// CompareResult lists every mismatch found; nil means a byte-exact pass.
type CompareResult struct {
	Mismatches []string
}

// CompareFiles byte-compares every expected file against dir.
func CompareFiles(dir string, expect map[string]string) *CompareResult {
	var mismatches []string
	for rel, want := range expect {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		got, err := os.ReadFile(full)
		if err != nil {
			mismatches = append(mismatches, fmt.Sprintf("%s: missing (%v)", rel, err))
			continue
		}
		if string(got) != want {
			mismatches = append(mismatches, fmt.Sprintf("%s: got %q, want %q", rel, string(got), want))
		}
	}
	if len(mismatches) == 0 {
		return nil
	}
	return &CompareResult{Mismatches: mismatches}
}

// CompareStdout byte-compares stdout against want; nil want means "no check".
func CompareStdout(got string, want *string) *CompareResult {
	if want == nil {
		return nil
	}
	if got != *want {
		return &CompareResult{Mismatches: []string{fmt.Sprintf("stdout: got %q, want %q", got, *want)}}
	}
	return nil
}
