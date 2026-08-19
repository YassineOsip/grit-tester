package runner

import (
	"os"
	"path/filepath"
)

// Diff is one byte-exact mismatch: where it happened, what was produced
// and what was expected.
type Diff struct {
	Where   string // file path, "stdout" or "exit code"
	Got     string
	Want    string
	Missing bool // expected file does not exist
}

// CompareFiles byte-compares every expected file against dir.
// An empty result means a byte-exact pass.
func CompareFiles(dir string, expect map[string]string) []Diff {
	var diffs []Diff
	for rel, want := range expect {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		got, err := os.ReadFile(full)
		if err != nil {
			diffs = append(diffs, Diff{Where: rel, Want: want, Missing: true})
			continue
		}
		if string(got) != want {
			diffs = append(diffs, Diff{Where: rel, Got: string(got), Want: want})
		}
	}
	return diffs
}

// CompareStdout byte-compares stdout against want; nil want means "no check".
func CompareStdout(got string, want *string) []Diff {
	if want == nil {
		return nil
	}
	if got != *want {
		return []Diff{{Where: "stdout", Got: got, Want: *want}}
	}
	return nil
}
