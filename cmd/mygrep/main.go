package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type Matcher struct {
	capturedGroups []string
	groupCaptured  []bool
	nextGroupNum   int
}

func NewMatcher() *Matcher {
	return &Matcher{
		capturedGroups: make([]string, 10),
		groupCaptured:  make([]bool, 10),
		nextGroupNum:   1,
	}
}

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

	matcher := NewMatcher()
	ok, err := matcher.MatchLine(string(line), pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	if !ok {
		os.Exit(1)
	}
}

func (m *Matcher) MatchLine(text, pattern string) (bool, error) {
	if len(pattern) > 0 && pattern[0] == '^' {
		return m.match(text, pattern, 0, 0)
	}

	for startPos := 0; startPos < len(text); startPos++ {
		if matched, err := m.match(text, pattern, startPos, 0); err != nil {
			return false, err
		} else if matched {
			return true, nil
		}
	}
	return false, nil
}

func (m *Matcher) match(text, pattern string, i, j int) (bool, error) {
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
		return m.match(text, pattern, i, j+1)
	}

	if j+1 < len(pattern) && pattern[j] == '\\' && unicode.IsDigit(rune(pattern[j+1])) {
		return m.matchBackreference(text, pattern, i, j)
	}

	if pattern[j] == '(' {
		return m.matchCapturingGroup(text, pattern, i, j)
	}

	matched, tokenLen, err := m.matchSingleChar(text, pattern, i, j)
	if err != nil {
		return false, err
	}

	nextPos := j + tokenLen
	if nextPos < len(pattern) && (pattern[nextPos] == '+' || pattern[nextPos] == '?') {
		return m.matchWithQuantifier(text, pattern, i, j, matched, nextPos)
	}

	if !matched {
		return false, nil
	}

	return m.match(text, pattern, i+1, nextPos)
}

func (m *Matcher) matchBackreference(text, pattern string, i, j int) (bool, error) {
	groupNumStartIdx := j + 1
	groupNumEndIdx := groupNumStartIdx + 1

	for groupNumEndIdx < len(pattern) && unicode.IsDigit(rune(pattern[groupNumEndIdx])) {
		groupNumEndIdx++
	}

	groupNumStr := pattern[groupNumStartIdx:groupNumEndIdx]
	groupNum, err := strconv.Atoi(groupNumStr)
	if err != nil || groupNum == 0 {
		return false, fmt.Errorf("invalid backreference: \\%s", groupNumStr)
	}

	if groupNum >= len(m.capturedGroups) {
		return false, nil
	}

	if !m.groupCaptured[groupNum] {
		return false, nil
	}

	captured := m.capturedGroups[groupNum]
	if len(text[i:]) < len(captured) {
		return false, nil
	}

	if text[i:i+len(captured)] == captured {
		return m.match(text, pattern, i+len(captured), j+len(groupNumStr)+1)
	}
	return false, nil
}

func (m *Matcher) matchCapturingGroup(text, pattern string, i, j int) (bool, error) {
	groupNum := m.nextGroupNum
	m.nextGroupNum++

	m.ensureGroupCapacity(groupNum)

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

	originalContent := pattern[j+1 : end]

	for length := 0; i+length <= len(text); length++ {
		savedState := m.saveState()

		if i+length > len(text) {
			break
		}
		captured := text[i : i+length]

		var matched bool
		if strings.Contains(originalContent, "|") {
			matched = m.tryAlternatives(captured, originalContent)
		} else {
			subpatternOk, _ := m.match(captured, "^"+originalContent+"$", 0, 0)
			matched = subpatternOk
		}

		if matched {
			m.capturedGroups[groupNum] = captured
			m.groupCaptured[groupNum] = true

			if nextOk, err := m.match(text, pattern, i+length, end+1); err != nil {
				m.restoreState(savedState)
				return false, err
			} else if nextOk {
				return true, nil
			}
		}

		m.restoreState(savedState)
	}

	m.nextGroupNum = groupNum
	return false, nil
}

func (m *Matcher) matchWithQuantifier(text, pattern string, i, j int, matched bool, nextPos int) (bool, error) {
	q := pattern[nextPos]

	if q == '+' {
		if !matched {
			return false, nil
		}

		count := 1
		maxCount := 1

		for i+maxCount < len(text) {
			m2, _, err2 := m.matchSingleChar(text, pattern, i+maxCount, j)
			if err2 != nil {
				return false, err2
			}
			if !m2 {
				break
			}
			maxCount++
		}

		for used := maxCount; used >= count; used-- {
			if ok, _ := m.match(text, pattern, i+used, nextPos+1); ok {
				return true, nil
			}
		}
		return false, nil
	} else if q == '?' {
		if ok, _ := m.match(text, pattern, i, nextPos+1); ok {
			return true, nil
		}

		if matched {
			return m.match(text, pattern, i+1, nextPos+1)
		}

		return false, nil
	}

	return false, nil
}

func (m *Matcher) matchSingleChar(text, pattern string, i, j int) (bool, int, error) {
	if i >= len(text) {
		return false, 0, nil
	}

	if pattern[j] == '.' {
		return true, 1, nil
	}

	if pattern[j] == '\\' && j+1 < len(pattern) {
		if unicode.IsDigit(rune(pattern[j+1])) {
			return false, 0, nil
		}

		return m.matchEscapeSequence(text, pattern, i, j)
	}

	if pattern[j] == '[' {
		return m.matchCharacterClass(text, pattern, i, j)
	}

	return (pattern[j] == text[i]), 1, nil
}

func (m *Matcher) matchEscapeSequence(text, pattern string, i, j int) (bool, int, error) {
	switch pattern[j+1] {
	case 'd':
		return unicode.IsDigit(rune(text[i])), 2, nil
	case 'w':
		ch := rune(text[i])
		isWord := unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_'
		return isWord, 2, nil
	case '\\', '(', ')', '.', '+', '*', '?', '[', ']', '|', '$', '^':
		return (text[i] == pattern[j+1]), 2, nil
	default:
		return false, 0, fmt.Errorf("invalid escape sequence: \\%c", pattern[j+1])
	}
}

func (m *Matcher) matchCharacterClass(text, pattern string, i, j int) (bool, int, error) {
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

	matched := (isNegative && !charInGroup) || (!isNegative && charInGroup)
	tokenLen := (closeBracket - j) + 1
	return matched, tokenLen, nil
}

func (m *Matcher) tryAlternatives(text, pattern string) bool {
	alts := m.splitAlternatives(pattern)

	for _, alt := range alts {
		altOk, _ := m.match(text, "^"+alt+"$", 0, 0)
		if altOk {
			return true
		}
	}
	return false
}

func (m *Matcher) splitAlternatives(pattern string) []string {
	var result []string
	var current strings.Builder
	depth := 0

	for i := 0; i < len(pattern); i++ {
		char := pattern[i]

		if char == '(' {
			depth++
			current.WriteByte(char)
		} else if char == ')' {
			depth--
			current.WriteByte(char)
		} else if char == '|' && depth == 0 {
			result = append(result, current.String())
			current.Reset()
		} else if char == '\\' && i+1 < len(pattern) {
			current.WriteByte(char)
			if i+1 < len(pattern) {
				current.WriteByte(pattern[i+1])
				i++
			}
		} else {
			current.WriteByte(char)
		}
	}

	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

func (m *Matcher) saveState() *Matcher {
	savedGroups := make([]string, len(m.capturedGroups))
	savedCaptured := make([]bool, len(m.groupCaptured))
	copy(savedGroups, m.capturedGroups)
	copy(savedCaptured, m.groupCaptured)

	return &Matcher{
		capturedGroups: savedGroups,
		groupCaptured:  savedCaptured,
		nextGroupNum:   m.nextGroupNum,
	}
}

func (m *Matcher) restoreState(saved *Matcher) {
	copy(m.capturedGroups, saved.capturedGroups)
	copy(m.groupCaptured, saved.groupCaptured)
	m.nextGroupNum = saved.nextGroupNum
}

func (m *Matcher) ensureGroupCapacity(groupNum int) {
	if groupNum >= len(m.capturedGroups) {
		newSize := groupNum + 1
		newGroups := make([]string, newSize)
		newCaptured := make([]bool, newSize)
		copy(newGroups, m.capturedGroups)
		copy(newCaptured, m.groupCaptured)
		m.capturedGroups = newGroups
		m.groupCaptured = newCaptured
	}
}
