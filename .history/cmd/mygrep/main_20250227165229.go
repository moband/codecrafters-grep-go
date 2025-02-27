package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
)

// Global capture for group #1
var group1 string
var group1Captured bool

func main() {
	if len(os.Args) < 3 || os.Args[1] != "-E" {
		fmt.Fprintf(os.Stderr, "usage: mygrep -E <pattern>\n")
		os.Exit(2)
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

	group1 = ""
	group1Captured = false

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

	if j+1 < len(pattern) && pattern[j] == '\\' && pattern[j+1] == '1' {
		if !group1Captured {
			return false, nil
		}
		if len(text[i:]) < len(group1) {
			return false, nil
		}
		if text[i:i+len(group1)] == group1 {
			return match(text, pattern, i+len(group1), j+2)
		}
		return false, nil
	}

	if pattern[j] == '(' {
		depth := 1
		end := j + 1
		for end < len(pattern) && depth > 0 {
			if pattern[end] == '(' {
				depth++
			} else if pattern[end] == ')' {
				depth--
			}
			end++
		}
		if depth > 0 {
			return false, fmt.Errorf("unmatched opening parenthesis")
		}
		end--

		content := pattern[j+1 : end]

		for length := 0; i+length <= len(text); length++ {
			if i+length > len(text) {
				break
			}

			captured := text[i : i+length]

			if strings.Contains(content, "|") {
				alts := splitAlternatives(content)
				for _, alt := range alts {
					if matchesPattern(captured, alt) {
						break
					}
				}
			} else {
				if !matchesPattern(captured, content) {
					continue
				}
			}

			group1 = captured
			group1Captured = true

			if nextOk, err := match(text, pattern, i+length, end+1); err != nil {
				return false, err
			} else if nextOk {
				return true, nil
			}
		}
		return false, nil
	}

	matched, tokenLen, err := matchSingleChar(text, pattern, i, j)
	if err != nil {
		return false, err
	}

	nextPos := j + tokenLen
	if nextPos < len(pattern) && (pattern[nextPos] == '+' || pattern[nextPos] == '?') {
		q := pattern[nextPos]
		if q == '+' {
			if !matched {
				return false, nil
			}

			count := 1
			maxCount := 1

			for i+maxCount < len(text) {
				m2, _, err2 := matchSingleChar(text, pattern, i+maxCount, j)
				if err2 != nil {
					return false, err2
				}
				if !m2 {
					break
				}
				maxCount++
			}
			for used := maxCount; used >= count; used-- {
				if ok, _ := match(text, pattern, i+used, nextPos+1); ok {
					return true, nil
				}
			}
			return false, nil

		} else if q == '?' {
			if ok, _ := match(text, pattern, i, nextPos+1); ok {
				return true, nil
			}

			if matched {
				return match(text, pattern, i+1, nextPos+1)
			}

			return false, nil
		}
	}

	if !matched {
		return false, nil
	}

	return match(text, pattern, i+1, nextPos)
}

func matchesPattern(s, pattern string) bool {
	if strings.ContainsAny(pattern, "+?") || strings.Contains(pattern, "\\w") || strings.Contains(pattern, "\\d") {
		ok, _ := matchGroup(s, pattern)
		return ok
	}

	if len(s) != len(pattern) {
		return false
	}

	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '.' {
			continue
		} else if s[i] != pattern[i] {
			return false
		}
	}

	return true
}

func matchSingleChar(text, pattern string, i, j int) (bool, int, error) {
	if i >= len(text) {
		return false, 0, nil
	}

	if pattern[j] == '.' {
		return true, 1, nil
	}

	if pattern[j] == '\\' && j+1 < len(pattern) {
		switch pattern[j+1] {
		case 'd':
			return unicode.IsDigit(rune(text[i])), 2, nil
		case 'w':
			ch := rune(text[i])
			isWord := unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
			return isWord, 2, nil
		case '\\':
			return (text[i] == '\\'), 2, nil
		default:
			return false, 0, fmt.Errorf("invalid escape sequence: \\%c", pattern[j+1])
		}
	}

	if pattern[j] == '[' {
		closeBracket := strings.IndexByte(pattern[j+1:], ']')
		if closeBracket == -1 {
			return false, 0, nil
		}
		closeBracket += (j + 1)

		isNegative := false
		startIdx := j + 1
		if pattern[startIdx] == '^' {
			isNegative = true
			startIdx++
		}

		chars := pattern[startIdx:closeBracket]
		charInGroup := strings.ContainsRune(chars, rune(text[i]))

		m := (isNegative && !charInGroup) || (!isNegative && charInGroup)
		tokenLen := (closeBracket - j) + 1
		return m, tokenLen, nil
	}

	m := (pattern[j] == text[i])
	return m, 1, nil
}

func matchGroup(candidate, subpattern string) (bool, error) {
	if ok, err := match(candidate, subpattern, 0, 0); err != nil {
		return false, err
	} else if ok {
		pat := "^" + subpattern + "$"
		if finalOk, _ := match(candidate, pat, 0, 0); finalOk {
			return true, nil
		}
	}
	return false, nil
}

func splitAlternatives(pattern string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for _, char := range pattern {
		switch char {
		case '(':
			depth++
			current.WriteRune(char)
		case ')':
			depth--
			current.WriteRune(char)
		case '|':
			if depth == 0 {
				result = append(result, current.String())
				current.Reset()
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}
	return result
}
