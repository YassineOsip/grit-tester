// Package suite defines the declarative case format grit-tester runs:
// a JSON file with a list of cases, each describing a command to execute
// and byte-exact expectations to compare against.
package suite

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// CurrentSchema is the version marker every cases.json must carry.
const CurrentSchema = 1

// Suite is a named collection of test cases.
type Suite struct {
	Schema int    `json:"schema"`
	Suite  string `json:"suite"`
	Cases  []Case `json:"cases"`
}

// Case is one executable test: prepare files, run a command, compare.
type Case struct {
	ID           string            `json:"id"`
	Description  string            `json:"description,omitempty"`
	Required     bool              `json:"required"`
	Setup        map[string]string `json:"setup,omitempty"` // path -> content
	Command      string            `json:"command"`
	Args         []string          `json:"args,omitempty"`
	Workdir      string            `json:"workdir,omitempty"` // may contain {{TARGET}}
	TimeoutSec   int               `json:"timeout_sec,omitempty"`
	ExpectFiles  map[string]string `json:"expect_files,omitempty"` // path -> byte-exact content
	ExpectStdout *string           `json:"expect_stdout,omitempty"`
	ExpectExit   *int              `json:"expect_exit,omitempty"`
}

// Load reads and validates a cases.json file.
func Load(path string) (*Suite, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(path, data)
}

// Parse decodes and validates a cases.json payload; name is used in error
// messages only.
func Parse(name string, data []byte) (*Suite, error) {
	var s Suite
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("parse %s: %w", name, err)
	}
	if err := s.Validate(); err != nil {
		return nil, fmt.Errorf("validate %s: %w", name, err)
	}
	return &s, nil
}

// Validate checks structural rules so failures surface before any case runs.
func (s *Suite) Validate() error {
	if s.Schema != CurrentSchema {
		return fmt.Errorf("unsupported schema %d (want %d)", s.Schema, CurrentSchema)
	}
	if s.Suite == "" {
		return fmt.Errorf("suite name is required")
	}
	if len(s.Cases) == 0 {
		return fmt.Errorf("no cases defined")
	}

	seen := make(map[string]bool, len(s.Cases))
	for i := range s.Cases {
		c := &s.Cases[i]
		if c.ID == "" {
			return fmt.Errorf("case %d: id is required", i)
		}
		if seen[c.ID] {
			return fmt.Errorf("case %s: duplicate id", c.ID)
		}
		seen[c.ID] = true
		if c.Command == "" {
			return fmt.Errorf("case %s: command is required", c.ID)
		}
		if len(c.ExpectFiles) == 0 && c.ExpectStdout == nil && c.ExpectExit == nil {
			return fmt.Errorf("case %s: needs at least one expectation (expect_files, expect_stdout or expect_exit)", c.ID)
		}
	}
	return nil
}

// Timeout returns the case timeout, defaulting to 30 seconds.
func (c *Case) Timeout() time.Duration {
	if c.TimeoutSec <= 0 {
		return 30 * time.Second
	}
	return time.Duration(c.TimeoutSec) * time.Second
}
