package suite

import "testing"

// TestParseEnvField checks the case "env" map round-trips through JSON:
// justify-style suites need per-case environment variables (e.g. COLUMNS)
// to pin the terminal width the program must align against.
func TestParseEnvField(t *testing.T) {
	data := []byte(`{
		"schema": 1,
		"suite": "env-demo",
		"cases": [{
			"id": "c1",
			"command": "go",
			"args": ["run", "."],
			"env": {"COLUMNS": "40", "LANG": "C"},
			"expect_exit": 0
		}]
	}`)
	s, err := Parse("env.json", data)
	if err != nil {
		t.Fatal(err)
	}
	env := s.Cases[0].Env
	if env["COLUMNS"] != "40" || env["LANG"] != "C" {
		t.Errorf("env = %v, want COLUMNS=40 LANG=C", env)
	}
}

// TestParseRejectsEmptyEnvKey guards against silently building a broken
// "=value" entry from an empty key.
func TestParseRejectsEmptyEnvKey(t *testing.T) {
	data := []byte(`{
		"schema": 1,
		"suite": "bad",
		"cases": [{
			"id": "c1",
			"command": "go",
			"env": {"": "x"},
			"expect_exit": 0
		}]
	}`)
	if _, err := Parse("bad.json", data); err == nil {
		t.Fatal("empty env key must be rejected")
	}
}
