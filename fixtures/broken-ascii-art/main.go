// Command broken is a deliberately naive ascii-art implementation for
// demonstrating grit-tester failure output. It uppercases the input,
// ignores banner selection, and skips newline handling.
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage")
		os.Exit(2)
	}
	text := strings.ToUpper(os.Args[1]) // BUG: destroys lowercase input

	data, err := os.ReadFile("standard.txt")
	if err != nil {
		os.Exit(1)
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
	if lines[0] == "" {
		lines = lines[1:]
	}
	glyphs := make(map[rune][]string)
	for i := 0; i < 95; i++ {
		glyphs[rune(' '+i)] = lines[i*9 : i*9+8]
	}

	for _, part := range strings.Split(text, "\n") {
		for r := 0; r < 8; r++ {
			var row strings.Builder
			for _, c := range part {
				if g, ok := glyphs[c]; ok {
					row.WriteString(g[r])
				}
			}
			fmt.Println(row.String())
		}
	}
}
