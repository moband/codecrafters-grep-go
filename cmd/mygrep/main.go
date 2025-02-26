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
	text := string(line)

	if len(pattern) > 0 && pattern[0] == '^' {
		return match(text, pattern, 0, 0)
	}

	for startPos := 0; startPos < len(text); startPos++ {
		if matched, err := match(text, pattern, startPos, 0); err != nil {
			return false, err
		} else if matched {
			return true, nil
		}
	}

	return false, nil
}

func match(text, pattern string, i, j int) (bool, error) {
	if j == len(pattern) {
		return true, nil
	}

	if j == len(pattern)-1 && pattern[j] == '$' {
		return i == len(text), nil
	}

	if i >= len(text) {
		return false, nil
	}

	if j == 0 && pattern[j] == '^' {
		return match(text, pattern, i, j+1)
	}

	if j+1 < len(pattern) && pattern[j] == '\\' {
		matched := false
		switch pattern[j+1] {
		case 'd':
			matched = unicode.IsDigit(rune(text[i]))
		case 'w':
			matched = unicode.IsLetter(rune(text[i])) || unicode.IsDigit(rune(text[i])) || text[i] == '_'
		case '\\':
			matched = text[i] == '\\'
		default:
			return false, fmt.Errorf("invalid escape sequence: %c", pattern[j+1])
		}

		if matched {
			return match(text, pattern, i+1, j+2)
		}
		return false, nil
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
			return match(text, pattern, i+1, closeBracket+1)
		}
		return false, nil
	}

	if pattern[j] == text[i] {
		return match(text, pattern, i+1, j+1)
	}

	return false, nil
}
