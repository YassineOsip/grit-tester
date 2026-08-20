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
	if diffs := CompareFiles(dir, map[string]string{"result.txt": "hi\n"}); len(diffs) != 0 {
		t.Errorf("want no diffs, got %+v", diffs)
	}
}

func TestCompareFilesMismatchShowsBothSides(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "result.txt"), []byte("ho\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diffs := CompareFiles(dir, map[string]string{"result.txt": "hi\n"})
	if len(diffs) != 1 {
		t.Fatalf("want 1 diff, got %+v", diffs)
	}
	d := diffs[0]
	if d.Where != "result.txt" || d.Got != "ho\n" || d.Want != "hi\n" || d.Missing {
		t.Errorf("diff = %+v", d)
	}
}

func TestCompareFilesMissing(t *testing.T) {
	diffs := CompareFiles(t.TempDir(), map[string]string{"result.txt": "hi\n"})
	if len(diffs) != 1 {
		t.Fatalf("want 1 diff for missing file, got %+v", diffs)
	}
	if !diffs[0].Missing || diffs[0].Where != "result.txt" || diffs[0].Want != "hi\n" {
		t.Errorf("diff = %+v", diffs[0])
	}
}

func TestCompareFilesAbsolutePath(t *testing.T) {
	f := filepath.Join(t.TempDir(), "out.txt")
	if err := os.WriteFile(f, []byte("hi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	diffs := CompareFiles(t.TempDir(), map[string]string{f: "hi\n"})
	if len(diffs) != 0 {
		t.Errorf("absolute-path compare must pass, got %+v", diffs)
	}
	diffs = CompareFiles(t.TempDir(), map[string]string{f: "ho\n"})
	if len(diffs) != 1 || diffs[0].Got != "hi\n" || diffs[0].Want != "ho\n" {
		t.Errorf("absolute-path mismatch = %+v", diffs)
	}
}

func TestCompareStdout(t *testing.T) {
	want := "out\n"
	if diffs := CompareStdout("out\n", &want); len(diffs) != 0 {
		t.Errorf("want no diffs for exact stdout, got %+v", diffs)
	}
	diffs := CompareStdout("other\n", &want)
	if len(diffs) != 1 || diffs[0].Where != "stdout" || diffs[0].Got != "other\n" || diffs[0].Want != "out\n" {
		t.Errorf("diffs = %+v", diffs)
	}
	if diffs := CompareStdout("ignored", nil); diffs != nil {
		t.Errorf("nil want means no check, got %+v", diffs)
	}
}
