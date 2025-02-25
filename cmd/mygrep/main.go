package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"unicode"
	"unicode/utf8"
)

// PatternMatcher is an interface for pattern matchers
type PatternMatcher interface {
	CanHandle(pattern string) bool
	Match(line []byte, pattern string) (bool, error)
}

// PatternMatcherRegistry is a registry of pattern matchers
type PatternMatcherRegistry struct {
	matchers []PatternMatcher
}

func NewPatternMatcherRegistry() *PatternMatcherRegistry {
	return &PatternMatcherRegistry{
		matchers: []PatternMatcher{
			&CharacterGroupMatcher{},
			&CharacterClassMatcher{},
			&LiteralMatcher{},
		},
	}
}

func (r *PatternMatcherRegistry) Match(line []byte, pattern string) (bool, error) {
	for _, matcher := range r.matchers {
		if matcher.CanHandle(pattern) {
			return matcher.Match(line, pattern)
		}
	}

	return false, nil
}

// CharacterGroupMatcher handles patterns like [abc]
type CharacterGroupMatcher struct{}

func (m *CharacterGroupMatcher) CanHandle(pattern string) bool {
	return len(pattern) >= 3 && pattern[0] == '[' && pattern[len(pattern)-1] == ']'
}

func (m *CharacterGroupMatcher) Match(line []byte, pattern string) (bool, error) {

	if len(pattern) >= 4 && pattern[1] == '^' {
		chars := pattern[2 : len(pattern)-1]
		return bytes.IndexFunc(line, func(r rune) bool {
			return !bytes.ContainsRune([]byte(chars), r)
		}) >= 0, nil
	}

	chars := pattern[1 : len(pattern)-1]
	return bytes.ContainsAny(line, chars), nil
}

// CharacterClassMatcher handles patterns like \d and \w
type CharacterClassMatcher struct{}

func (m *CharacterClassMatcher) CanHandle(pattern string) bool {
	return len(pattern) == 2 && pattern[0] == '\\'
}

func (m *CharacterClassMatcher) Match(line []byte, pattern string) (bool, error) {
	switch pattern[1] {
	case 'd':
		return bytes.ContainsFunc(line, unicode.IsDigit), nil
	case 'w':
		return bytes.ContainsFunc(line, func(r rune) bool {
			return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_'
		}), nil
	default:
		return false, nil
	}
}

// LiteralMatcher handles patterns like a and b
type LiteralMatcher struct{}

func (m *LiteralMatcher) CanHandle(pattern string) bool {
	return utf8.RuneCountInString(pattern) == 1
}

func (m *LiteralMatcher) Match(line []byte, pattern string) (bool, error) {
	return bytes.Contains(line, []byte(pattern)), nil
}

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

	matcher := NewPatternMatcherRegistry()
	return matcher.Match(line, pattern)
}
