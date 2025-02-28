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
var nextGroupNum int // To keep track of the next group number

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

	// Initialize captured groups to handle up to 9 groups
	capturedGroups = make([]string, 10)
	groupCaptured = make([]bool, 10)
	nextGroupNum = 1 // Group numbering starts at 1

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
		// Get the backreference number - support multi-digit backreferences
		groupNumStartIdx := j + 1
		groupNumEndIdx := groupNumStartIdx + 1

		// Find all consecutive digits for backreference number
		for groupNumEndIdx < len(pattern) && unicode.IsDigit(rune(pattern[groupNumEndIdx])) {
			groupNumEndIdx++
		}

		groupNumStr := pattern[groupNumStartIdx:groupNumEndIdx]
		groupNum, err := strconv.Atoi(groupNumStr)
		if err != nil || groupNum == 0 {
			return false, fmt.Errorf("invalid backreference: \\%s", groupNumStr)
		}

		// Make sure we have enough space for this group
		if groupNum >= len(capturedGroups) {
			return false, nil
		}

		if !groupCaptured[groupNum] {
			return false, nil
		}

		captured := capturedGroups[groupNum]
		if len(text[i:]) < len(captured) {
			return false, nil
		}

		if text[i:i+len(captured)] == captured {
			return match(text, pattern, i+len(captured), j+len(groupNumStr)+1)
		}
		return false, nil
	}

	//-------------------------------------------------------------
	// Handle capturing group ( ... )
	//-------------------------------------------------------------
	if pattern[j] == '(' {
		// Get the current group number and increment for the next one
		groupNum := nextGroupNum
		nextGroupNum++

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

		originalContent := pattern[j+1 : end]

		// Try to match the entire capturing group
		for length := 0; i+length <= len(text); length++ {
			if i+length > len(text) {
				break
			}

			captured := text[i : i+length]

			// Save the state of captured groups
			savedGroups := make([]string, len(capturedGroups))
			savedCaptured := make([]bool, len(groupCaptured))
			savedNextGroupNum := nextGroupNum
			copy(savedGroups, capturedGroups)
			copy(savedCaptured, groupCaptured)

			var matched bool

			// First, try handling alternatives if they exist
			if strings.Contains(originalContent, "|") {
				// Get all alternatives
				alts := splitAlternatives(originalContent)

				// Try each alternative
				for _, alt := range alts {
					// Run the full match algorithm on the captured text against this alternative
					altOk, _ := match(captured, "^"+alt+"$", 0, 0)
					if altOk {
						matched = true
						break
					}
				}
			} else {
				// No alternatives, try to match the content directly
				subpatternOk, _ := match(captured, "^"+originalContent+"$", 0, 0)
				matched = subpatternOk
			}

			// If the pattern matched, store this capture and try to continue the match
			if matched {
				// Store the capture in the correct group
				capturedGroups[groupNum] = captured
				groupCaptured[groupNum] = true

				// Try to match the rest of the pattern
				if nextOk, err := match(text, pattern, i+length, end+1); err != nil {
					// Restore the saved state if there's an error
					copy(capturedGroups, savedGroups)
					copy(groupCaptured, savedCaptured)
					nextGroupNum = savedNextGroupNum
					return false, err
				} else if nextOk {
					return true, nil
				}
			}

			// Restore saved state if match doesn't continue
			copy(capturedGroups, savedGroups)
			copy(groupCaptured, savedCaptured)
			nextGroupNum = savedNextGroupNum
		}

		// Reset the group number counter if we failed to match this group
		nextGroupNum = groupNum
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
	// Look for backreferences, capturing groups, quantifiers, or character classes
	if strings.ContainsAny(pattern, "+?") ||
		strings.Contains(pattern, "\\w") ||
		strings.Contains(pattern, "\\d") ||
		strings.Contains(pattern, "(") ||
		strings.Contains(pattern, "\\") && containsBackreference(pattern) {

		// We need to anchor the pattern to ensure proper full matching
		anchoredPattern := "^" + pattern + "$"
		ok, _ := match(s, anchoredPattern, 0, 0)
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

// Helper function to check if a pattern contains a backreference
func containsBackreference(pattern string) bool {
	for i := 0; i < len(pattern)-1; i++ {
		if pattern[i] == '\\' && unicode.IsDigit(rune(pattern[i+1])) {
			return true
		}
	}
	return false
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
		case '(':
			return (text[i] == '('), 2, nil
		case ')':
			return (text[i] == ')'), 2, nil
		case '.':
			return (text[i] == '.'), 2, nil
		case '+':
			return (text[i] == '+'), 2, nil
		case '*':
			return (text[i] == '*'), 2, nil
		case '?':
			return (text[i] == '?'), 2, nil
		case '[':
			return (text[i] == '['), 2, nil
		case ']':
			return (text[i] == ']'), 2, nil
		case '|':
			return (text[i] == '|'), 2, nil
		case '$':
			return (text[i] == '$'), 2, nil
		case '^':
			return (text[i] == '^'), 2, nil
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
	savedNextGroupNum := nextGroupNum
	copy(savedGroups, capturedGroups)
	copy(savedCaptured, groupCaptured)

	// When we're matching a nested pattern, we need to keep the parent's captured groups
	// We don't reset capturedGroups here anymore, so nested backreferences work

	// Process any backreferences in the pattern before matching
	if containsBackreference(subpattern) {
		processedPattern, err := processBackreferences(subpattern)
		if err == nil {
			// If processing succeeded, use the processed pattern
			subpattern = processedPattern
		}
		// If there was an error processing backreferences, fall back to the original pattern
	}

	if ok, err := match(candidate, subpattern, 0, 0); err != nil {
		// Restore the saved groups before returning
		capturedGroups = savedGroups
		groupCaptured = savedCaptured
		nextGroupNum = savedNextGroupNum
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
		nextGroupNum = savedNextGroupNum

		return finalOk, nil
	}

	// Restore the saved groups
	capturedGroups = savedGroups
	groupCaptured = savedCaptured
	nextGroupNum = savedNextGroupNum

	return false, nil
}

func splitAlternatives(pattern string) []string {
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
			// We found an alternative at the top level
			result = append(result, current.String())
			current.Reset()
		} else if char == '\\' && i+1 < len(pattern) {
			// Handle escape sequences like \|, \(, \), etc.
			current.WriteByte(char)
			if i+1 < len(pattern) {
				current.WriteByte(pattern[i+1])
				i++ // Skip the next character
			}
		} else {
			current.WriteByte(char)
		}
	}

	// Add the last alternative
	if current.Len() > 0 {
		result = append(result, current.String())
	}

	return result
}

// processBackreferences expands any backreferences in the pattern with their captured values
func processBackreferences(pattern string) (string, error) {
	var result strings.Builder
	i := 0

	for i < len(pattern) {
		if i+1 < len(pattern) && pattern[i] == '\\' && unicode.IsDigit(rune(pattern[i+1])) {
			// Get the backreference number - support multi-digit backreferences
			groupNumStartIdx := i + 1
			groupNumEndIdx := groupNumStartIdx + 1

			// Find all consecutive digits for backreference number
			for groupNumEndIdx < len(pattern) && unicode.IsDigit(rune(pattern[groupNumEndIdx])) {
				groupNumEndIdx++
			}

			groupNumStr := pattern[groupNumStartIdx:groupNumEndIdx]
			groupNum, err := strconv.Atoi(groupNumStr)
			if err != nil || groupNum == 0 {
				return "", fmt.Errorf("invalid backreference: \\%s", groupNumStr)
			}

			// Check if group exists
			if groupNum >= len(capturedGroups) {
				return "", fmt.Errorf("backreference to non-existent group: \\%s", groupNumStr)
			}

			if !groupCaptured[groupNum] {
				return "", fmt.Errorf("backreference to uncaptured group: \\%s", groupNumStr)
			}

			// Append the captured text
			result.WriteString(capturedGroups[groupNum])
			i = groupNumEndIdx
		} else if i+1 < len(pattern) && pattern[i] == '\\' {
			// Handle other escape sequences
			switch pattern[i+1] {
			case 'w', 'd', '(', ')', '.', '+', '*', '?', '[', ']', '|', '$', '^', '\\', ',':
				// Pass through the escape sequence
				result.WriteByte('\\')
				result.WriteByte(pattern[i+1])
			default:
				// For unknown escapes, just keep them as is
				result.WriteByte('\\')
				result.WriteByte(pattern[i+1])
			}
			i += 2
		} else {
			result.WriteByte(pattern[i])
			i++
		}
	}

	return result.String(), nil
}
