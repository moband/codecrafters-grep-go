package main

import (
	"bytes"
	"unicode"
	"unicode/utf8"
)

type PatternMatcher interface {
	CanHandle(pattern string) bool
	Match(line []byte, pattern string) (bool, error)
}

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
	return bytes.ContainsAny(line, pattern[1:len(pattern)-1]), nil
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
	}
	return false, nil
}

// LiteralMatcher handles patterns like a and b
type LiteralMatcher struct{}

func (m *LiteralMatcher) CanHandle(pattern string) bool {
	return utf8.RuneCountInString(pattern) == 1
}

func (m *LiteralMatcher) Match(line []byte, pattern string) (bool, error) {
	return bytes.Contains(line, []byte(pattern)), nil
}
