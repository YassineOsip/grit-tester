// Package suites embeds the built-in test suites inside the tester
// binary, so a single `go install` or release download is all that is
// needed to run them from any directory — no repo clone required.
//
// To add a project: create suites/<name>/cases.json. The go:embed glob
// picks it up automatically on the next build.
package suites

import "embed"

//go:embed */cases.json
var Files embed.FS
