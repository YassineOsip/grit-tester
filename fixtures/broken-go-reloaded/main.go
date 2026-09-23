// A deliberately naive go-reloaded implementation used to demonstrate
// grit-tester's failing output. It:
//   - handles only (up)/(low) WITHOUT counts,
//   - splits purely on spaces (misses "(up,2)" and glued punctuation),
//   - ignores punctuation spacing, quotes and a->an,
//   - always appends a trailing newline.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: go run . <in> <out>")
		os.Exit(1)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		os.Exit(1)
	}

	words := strings.Fields(string(data))
	for i := 0; i < len(words); i++ {
		switch words[i] {
		case "(up)":
			words[i-1] = strings.ToUpper(words[i-1])
			words = append(words[:i], words[i+1:]...)
			i--
		case "(low)":
			words[i-1] = strings.ToLower(words[i-1])
			words = append(words[:i], words[i+1:]...)
			i--
		}
	}

	os.WriteFile(os.Args[2], []byte(strings.Join(words, " ")+"\n"), 0o644)
}
