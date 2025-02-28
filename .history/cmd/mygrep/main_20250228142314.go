package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode"
)

// Store multiple capture groups
var capturedGroups []string
var groupCaptured []bool

func main() {
	if len(os.Args) < 3 || os.Args[1] != "-E" {
		fmt.Fprintf(os.Stderr, "usage: mygrep -E <pattern>\n")
		os.Exit(2)
	}

	pattern := os.Args[2]
	fmt.Fprintf(os.Stderr, "Pattern: %s\n", pattern)

	line, err := io.ReadAll(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: read input text: %v\n", err)
		os.Exit(2)
	}

	fmt.Fprintf(os.Stderr, "Input: %s\n", string(line))

	ok, err := matchLine(line, pattern)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	fmt.Fprintf(os.Stderr, "Match result: %v\n", ok)
	fmt.Fprintf(os.Stderr, "Captured groups: %v\n", capturedGroups)

	if !ok {
		os.Exit(1)
	}
}

func matchLine(line []byte, pattern string) (bool, error) {
	text := string(line)

	// Initialize captured groups to handle up to 9 groups
	capturedGroups = make([]string, 10)
	groupCaptured = make([]bool, 10)

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

	//-------------------------------------------------------------
	// Handle backreference \1, \2, etc.
	//-------------------------------------------------------------
	if j+1 < len(pattern) && pattern[j] == '\\' && unicode.IsDigit(rune(pattern[j+1])) {
		groupNumStr := string(pattern[j+1])
		groupNum, err := strconv.Atoi(groupNumStr)
		if err != nil || groupNum >= len(capturedGroups) {
			fmt.Fprintf(os.Stderr, "Invalid backreference: \\%s\n", groupNumStr)
			return false, fmt.Errorf("invalid backreference: \\%s", groupNumStr)
		}

		fmt.Fprintf(os.Stderr, "Checking backreference \\%d at position %d\n", groupNum, i)
		fmt.Fprintf(os.Stderr, "Group %d captured: %v, value: %s\n", groupNum, groupCaptured[groupNum], capturedGroups[groupNum])

		if !groupCaptured[groupNum] {
			fmt.Fprintf(os.Stderr, "Group %d not captured yet\n", groupNum)
			return false, nil
		}

		captured := capturedGroups[groupNum]
		if len(text[i:]) < len(captured) {
			fmt.Fprintf(os.Stderr, "Not enough text left to match backreference \\%d\n", groupNum)
			return false, nil
		}

		if text[i:i+len(captured)] == captured {
			fmt.Fprintf(os.Stderr, "Backreference \\%d matched: %s\n", groupNum, captured)
			return match(text, pattern, i+len(captured), j+2)
		}
		fmt.Fprintf(os.Stderr, "Backreference \\%d failed to match: expected %s, got %s\n",
			groupNum, captured, text[i:min(i+len(captured), len(text))])
		return false, nil
	}

	//-------------------------------------------------------------
	// Handle capturing group ( ... )
	//-------------------------------------------------------------
	if pattern[j] == '(' {
		// Count the number of opening parentheses before this one to determine the group number
		groupNum := 0
		for k := 0; k < j; k++ {
			if pattern[k] == '(' {
				groupNum++
			}
		}

		fmt.Fprintf(os.Stderr, "Trying to match capturing group %d at position %d\n", groupNum, i)

		// Make sure we have enough space
		if groupNum >= len(capturedGroups) {
			newSize := groupNum + 1
			newGroups := make([]string, newSize)
			newCaptured := make([]bool, newSize)
			copy(newGroups, capturedGroups)
			copy(newCaptured, groupCaptured)
			capturedGroups = newGroups
			groupCaptured = newCaptured
		}

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
		end-- // now 'end' is the position of ')'

		content := pattern[j+1 : end]

		// Try to match the entire capturing group
		for length := 0; i+length <= len(text); length++ {
			if i+length > len(text) {
				break
			}

			captured := text[i : i+length]

			// Check if this captured text matches the group pattern
			var matches bool

			// If there are alternatives in the group
			if strings.Contains(content, "|") {
				alts := splitAlternatives(content)
				for _, alt := range alts {
					// Check if the captured text matches this alternative
					if matchesPattern(captured, alt) {
						matches = true
						break
					}
				}
			} else {
				// No alternatives, just check if it matches the pattern
				matches = matchesPattern(captured, content)
			}

			if matches {
				// Store the capture in the correct group
				capturedGroups[groupNum] = captured
				groupCaptured[groupNum] = true
				fmt.Fprintf(os.Stderr, "Captured group %d: %s\n", groupNum, captured)

				// Try to match the rest of the pattern
				if nextOk, err := match(text, pattern, i+length, end+1); err != nil {
					return false, err
				} else if nextOk {
					return true, nil
				}
			}
		}
		return false, nil
	}

	//-------------------------------------------------------------
	// Check if the next token is followed by '+' or '?'
	// We do NOT advance j here until after we do matchSingleChar.
	//-------------------------------------------------------------

	// Try to match one instance of the "next token" (dot, bracket, escape, or literal).
	matched, tokenLen, err := matchSingleChar(text, pattern, i, j)
	if err != nil {
		return false, err
	}

	// If matched a single instance, see if there's a quantifier
	nextPos := j + tokenLen // next position in pattern after the bracket or escape
	if nextPos < len(pattern) && (pattern[nextPos] == '+' || pattern[nextPos] == '?') {
		q := pattern[nextPos]
		if q == '+' {
			// "One or more"
			if !matched {
				return false, nil
			}

			// We already matched one. Let's see how many more we can match greedily
			count := 1
			maxCount := 1

			// Try continuing to match the same "token" as many times as possible
			for i+maxCount < len(text) {
				// Attempt an additional match
				m2, _, err2 := matchSingleChar(text, pattern, i+maxCount, j)
				if err2 != nil {
					return false, err2
				}
				if !m2 {
					break
				}
				maxCount++
			}
			// Now try from maxCount down to 1
			for used := maxCount; used >= count; used-- {
				if ok, _ := match(text, pattern, i+used, nextPos+1); ok {
					return true, nil
				}
			}
			return false, nil

		} else if q == '?' {
			// "Zero or one"
			// For '?', try skipping it first, then try using it if it matched

			// 1) Try ignoring it (skip the optional character)
			if ok, _ := match(text, pattern, i, nextPos+1); ok {
				return true, nil
			}

			// 2) Try using it (only if we matched it)
			if matched {
				return match(text, pattern, i+1, nextPos+1)
			}

			return false, nil
		}
	}

	//-------------------------------------------------------------
	// If there's no quantifier, just check if we matched and move on
	//-------------------------------------------------------------
	if !matched {
		return false, nil
	}

	return match(text, pattern, i+1, nextPos)
}

// Helper function to check if a string matches a pattern
func matchesPattern(s, pattern string) bool {
	// For patterns with quantifiers or character classes, we need to use the match function
	if strings.ContainsAny(pattern, "+?") || strings.Contains(pattern, "\\w") || strings.Contains(pattern, "\\d") {
		ok, _ := matchGroup(s, pattern)
		return ok
	}

	// For simple patterns like "b..s", we can do a direct check
	if len(s) != len(pattern) {
		return false
	}

	for i := 0; i < len(pattern); i++ {
		if pattern[i] == '.' {
			// Any character is allowed
			continue
		} else if s[i] != pattern[i] {
			return false
		}
	}

	return true
}

// matchSingleChar
//
// Returns (matched bool, tokenLength int, err error)
//
// - `matched` is whether text[i] matches the single pattern "token" at pattern[j..]
// - `tokenLength` is how many characters of the pattern we used for that single token
//
// (So if pattern is "[abcd]", `tokenLength` will be the length from `j` up to `]`,
//
//	but does NOT include a trailing '+' or '?'. We want match() to see that next.)
func matchSingleChar(text, pattern string, i, j int) (bool, int, error) {
	if i >= len(text) {
		return false, 0, nil
	}

	// '.' => matches any single char
	if pattern[j] == '.' {
		return true, 1, nil
	}

	// If it's a backslash-escape like \d, \w, etc. (but not \1, \2, etc.)
	if pattern[j] == '\\' && j+1 < len(pattern) {
		if unicode.IsDigit(rune(pattern[j+1])) {
			// Handled in match()
			return false, 0, nil
		}

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

	// If pattern[j] == '[', parse a bracket expression.
	if pattern[j] == '[' {
		closeBracket := strings.IndexByte(pattern[j+1:], ']')
		if closeBracket == -1 {
			// No closing bracket
			return false, 0, nil
		}
		closeBracket += (j + 1) // offset from j+1

		// Check if it's negative class: [^xyz]
		isNegative := false
		startIdx := j + 1
		if pattern[startIdx] == '^' {
			isNegative = true
			startIdx++
		}

		// The actual characters inside the brackets
		chars := pattern[startIdx:closeBracket]
		charInGroup := strings.ContainsRune(chars, rune(text[i]))

		m := (isNegative && !charInGroup) || (!isNegative && charInGroup)
		tokenLen := (closeBracket - j) + 1 // e.g. "[abcd]" => length 6
		return m, tokenLen, nil
	}

	// Otherwise, it's a literal character
	m := (pattern[j] == text[i])
	return m, 1, nil
}

// matchGroup tries to match subpattern fully against candidate.
func matchGroup(candidate, subpattern string) (bool, error) {
	// Save the current captured groups
	savedGroups := make([]string, len(capturedGroups))
	savedCaptured := make([]bool, len(groupCaptured))
	copy(savedGroups, capturedGroups)
	copy(savedCaptured, groupCaptured)

	// Reset captured groups for this match
	capturedGroups = make([]string, 10)
	groupCaptured = make([]bool, 10)

	if ok, err := match(candidate, subpattern, 0, 0); err != nil {
		// Restore the saved groups before returning
		capturedGroups = savedGroups
		groupCaptured = savedCaptured
		return false, err
	} else if ok {
		// We want to ensure it doesn't match only partially.
		// The naive `match` can succeed even if candidate is partially matched.
		// So let's ensure we consumed the entire candidate with ^...$ approach:
		pat := "^" + subpattern + "$"
		finalOk, _ := match(candidate, pat, 0, 0)

		// Restore the saved groups
		capturedGroups = savedGroups
		groupCaptured = savedCaptured

		return finalOk, nil
	}

	// Restore the saved groups
	capturedGroups = savedGroups
	groupCaptured = savedCaptured

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

// Helper function to get the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
