package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompareFilesExact(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "result.txt"), []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if cr := CompareFiles(dir, map[string]string{"result.txt": "hi\n"}); cr != nil {
		t.Errorf("want nil, got %v", cr.Mismatches)
	}
}

func TestCompareFilesMismatchShowsBothSides(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "result.txt"), []byte("ho\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cr := CompareFiles(dir, map[string]string{"result.txt": "hi\n"})
	if cr == nil || len(cr.Mismatches) != 1 {
		t.Fatalf("want 1 mismatch, got %+v", cr)
	}
	if cr.Mismatches[0] != `result.txt: got "ho\n", want "hi\n"` {
		t.Errorf("mismatch text = %q", cr.Mismatches[0])
	}
}

func TestCompareFilesMissing(t *testing.T) {
	cr := CompareFiles(t.TempDir(), map[string]string{"result.txt": "hi\n"})
	if cr == nil {
		t.Fatal("want mismatch for missing file")
	}
}

func TestCompareStdout(t *testing.T) {
	want := "out\n"
	if cr := CompareStdout("out\n", &want); cr != nil {
		t.Error("want nil for exact stdout")
	}
	if cr := CompareStdout("other\n", &want); cr == nil {
		t.Error("want mismatch for wrong stdout")
	}
	if cr := CompareStdout("ignored", nil); cr != nil {
		t.Error("nil want means no check")
	}
}
