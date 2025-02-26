package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// Usage: echo <input_text> | your_program.sh -E <pattern>
func main() {
	if len(os.Args) < 3 || os.Args[1] != "-E" {
		fmt.Fprintf(os.Stderr, "usage: mygrep -E <pattern>\n")
		os.Exit(2) // 1 means no lines were selected, >1 means error
	}

	pattern := os.Args[2]

	line, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read input text: %v\n", err)
		os.Exit(2)
	}

	ok, err := matchLine(line, pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if !ok {
		os.Exit(1)
	}

}

func matchLine(line []byte, pattern string) (bool, error) {
	return match(string(line), pattern, 0, 0)
}

func match(text, pattern string, i, j int) (bool, error) {
	if j == len(pattern) {
		return true, nil
	}

	if i >= len(text) {
		return false, nil
	}

	if j+1 < len(pattern) && pattern[j] == '\\' {
		matched := false
		switch pattern[j+1] {
		case 'd':
			matched = unicode.IsDigit(rune(text[i]))
		case 'w':
			matched = unicode.IsLetter(rune(text[i])) || unicode.IsDigit(rune(text[i])) || text[i] == '_'
		default:
			return false, nil
		}

		if matched {
			if ok, _ := match(text, pattern, i+1, j+2); ok {
				return true, nil
			}
		}

		return match(text, pattern, i+1, j)
	}

	if pattern[j] == '[' {
		closeBracket := strings.IndexByte(pattern[j:], ']')
		if closeBracket == -1 {
			return false, nil
		}

		closeBracket += j
		if i >= len(text) {
			return false, nil
		}

		isNegative := j+1 < len(pattern) && pattern[j+1] == '^'
		startIdx := j + 1
		if isNegative {
			startIdx++
		}

		chars := pattern[startIdx:closeBracket]
		charInGroup := strings.ContainsRune(chars, rune(text[i]))

		if (isNegative && !charInGroup) || (!isNegative && charInGroup) {
			if ok, _ := match(text, pattern, i+1, closeBracket+1); ok {
				return true, nil
			}
		}
		return match(text, pattern, i+1, j)
	}

	if pattern[j] == text[i] {
		if ok, _ := match(text, pattern, i+1, j+1); ok {
			return true, nil
		}
	}

	return match(text, pattern, i+1, j)
}
