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

	if j+1 < len(pattern) && pattern[j+1] == '+' {
		matched, err := matchSingleChar(text, pattern, i, j)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}

		count := 1
		maxCount := 0

		for i+maxCount < len(text) {
			nextMatched, err := matchSingleChar(text, pattern, i+maxCount, j)
			if err != nil {
				return false, err
			}
			if !nextMatched {
				break
			}
			maxCount++
		}

		for count = maxCount; count >= 1; count-- {
			if ok, _ := match(text, pattern, i+count, j+2); ok {
				return true, nil
			}
		}

		return false, nil
	}

	if j+1 < len(pattern) && pattern[j+1] == '?' {
		matched, err := matchSingleChar(text, pattern, i, j)
		if err != nil {
			return false, err
		}

		patternAdvance := 1
		if j+1 < len(pattern) && pattern[j] == '\\' {
			patternAdvance = 2
		} else if pattern[j] == '[' {
			closeBracket := strings.IndexByte(pattern[j:], ']')
			if closeBracket == -1 {
				return false, nil
			}
			patternAdvance = closeBracket + 1
		}

		if matched {
			if ok, _ := match(text, pattern, i+1, j+patternAdvance+1); ok {
				return true, nil
			}
		}

		return match(text, pattern, i, j+patternAdvance+1)
	}

	matched, err := matchSingleChar(text, pattern, i, j)
	if err != nil {
		return false, err
	}

	if matched {
		patternAdvance := 1

		if j+1 < len(pattern) && pattern[j] == '\\' {
			patternAdvance = 2
		} else if pattern[j] == '[' {
			closeBracket := strings.IndexByte(pattern[j:], ']')
			if closeBracket == -1 {
				return false, nil
			}
			patternAdvance = closeBracket + 1
		}

		return match(text, pattern, i+1, j+patternAdvance)
	}

	return false, nil
}

func matchSingleChar(text, pattern string, i, j int) (bool, error) {
	if i >= len(text) {
		return false, nil
	}

	if pattern[j] == '.' {
		return true, nil
	}

	if j+1 < len(pattern) && pattern[j] == '\\' {
		switch pattern[j+1] {
		case 'd':
			return unicode.IsDigit(rune(text[i])), nil
		case 'w':
			return unicode.IsLetter(rune(text[i])) || unicode.IsDigit(rune(text[i])) || text[i] == '_', nil
		case '\\':
			return text[i] == '\\', nil
		default:
			return false, fmt.Errorf("invalid escape sequence: %c", pattern[j+1])
		}
	}

	if pattern[j] == '[' {
		closeBracket := strings.IndexByte(pattern[j:], ']')
		if closeBracket == -1 {
			return false, nil
		}

		closeBracket += j

		isNegative := j+1 < len(pattern) && pattern[j+1] == '^'
		startIdx := j + 1
		if isNegative {
			startIdx++
		}

		chars := pattern[startIdx:closeBracket]
		charInGroup := strings.ContainsRune(chars, rune(text[i]))

		return (isNegative && !charInGroup) || (!isNegative && charInGroup), nil
	}

	return pattern[j] == text[i], nil
}
