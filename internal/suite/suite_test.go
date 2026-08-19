package suite

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

const validFixture = `{
  "schema": 1,
  "suite": "demo",
  "cases": [
    {
      "id": "A1",
      "description": "a case",
      "required": true,
      "setup": {"sample.txt": "hi\n"},
      "command": "go",
      "args": ["run", "{{TARGET}}", "sample.txt", "result.txt"],
      "expect_files": {"result.txt": "hi\n"}
    }
  ]
}`

func writeFixture(t *testing.T, content string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "cases.json")
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoadValid(t *testing.T) {
	s, err := Load(writeFixture(t, validFixture))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.Suite != "demo" || len(s.Cases) != 1 {
		t.Fatalf("bad load: %+v", s)
	}
	c := s.Cases[0]
	if c.ID != "A1" || !c.Required || c.Command != "go" || len(c.Args) != 4 {
		t.Errorf("bad case: %+v", c)
	}
	if got := c.Timeout(); got != 30*time.Second {
		t.Errorf("default timeout = %v, want 30s", got)
	}
	c.TimeoutSec = 5
	if got := c.Timeout(); got != 5*time.Second {
		t.Errorf("custom timeout = %v, want 5s", got)
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.json")); err == nil {
		t.Fatal("want error for missing file")
	}
}

func TestValidateRejects(t *testing.T) {
	tests := []struct {
		name    string
		fixture string
	}{
		{
			name: "empty id",
			fixture: `{"schema": 1, "suite": "x", "cases": [
				{"id": "", "required": true, "command": "go", "expect_exit": 0}]}`,
		},
		{
			name: "missing command",
			fixture: `{"schema": 1, "suite": "x", "cases": [
				{"id": "A1", "required": true, "expect_exit": 0}]}`,
		},
		{
			name: "no expectations",
			fixture: `{"schema": 1, "suite": "x", "cases": [
				{"id": "A1", "required": true, "command": "go"}]}`,
		},
		{
			name: "wrong schema",
			fixture: `{"schema": 99, "suite": "x", "cases": [
				{"id": "A1", "required": true, "command": "go", "expect_exit": 0}]}`,
		},
		{
			name: "duplicate id",
			fixture: `{"schema": 1, "suite": "x", "cases": [
				{"id": "A1", "required": true, "command": "go", "expect_exit": 0},
				{"id": "A1", "required": true, "command": "go", "expect_exit": 0}]}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := Load(writeFixture(t, tt.fixture)); err == nil {
				t.Fatal("want validation error")
			}
		})
	}
}
