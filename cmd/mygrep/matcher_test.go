package main

import (
	"strings"
	"testing"
)

// TestMatchLine tests the MatchLine function with various regex patterns
func TestMatchLine(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		pattern string
		want    bool
	}{
		// Basic character matching
		{
			name:    "Match literal character",
			text:    "dog",
			pattern: "d",
			want:    true,
		},
		{
			name:    "Not match literal character",
			text:    "dog",
			pattern: "f",
			want:    false,
		},

		// Digit matching
		{
			name:    "Match digit",
			text:    "123",
			pattern: "\\d",
			want:    true,
		},
		{
			name:    "Not match digit",
			text:    "apple",
			pattern: "\\d",
			want:    false,
		},

		// Word character matching
		{
			name:    "Match word character",
			text:    "word",
			pattern: "\\w",
			want:    true,
		},
		{
			name:    "Not match word character",
			text:    "$!?",
			pattern: "\\w",
			want:    false,
		},

		// Character groups
		{
			name:    "Match positive character group",
			text:    "a",
			pattern: "[abcd]",
			want:    true,
		},
		{
			name:    "Not match positive character group",
			text:    "efgh",
			pattern: "[abcd]",
			want:    false,
		},
		{
			name:    "Match negative character group",
			text:    "apple",
			pattern: "[^xyz]",
			want:    true,
		},
		{
			name:    "Not match negative character group",
			text:    "banana",
			pattern: "[^anb]",
			want:    false,
		},

		// Character class combinations
		{
			name:    "Match single digit with word",
			text:    "sally has 3 apples",
			pattern: "\\d apple",
			want:    true,
		},
		{
			name:    "Not match digit with wrong word",
			text:    "sally has 1 orange",
			pattern: "\\d apple",
			want:    false,
		},
		{
			name:    "Match three digits with word",
			text:    "sally has 124 apples",
			pattern: "\\d\\d\\d apples",
			want:    true,
		},
		{
			name:    "Match digit with three word chars",
			text:    "sally has 3 dogs",
			pattern: "\\d \\w\\w\\ws",
			want:    true,
		},

		// Anchors
		{
			name:    "Match start of string",
			text:    "log",
			pattern: "^log",
			want:    true,
		},
		{
			name:    "Not match start of string",
			text:    "slog",
			pattern: "^log",
			want:    false,
		},
		{
			name:    "Match end of string",
			text:    "cat",
			pattern: "cat$",
			want:    true,
		},
		{
			name:    "Not match end of string",
			text:    "cats",
			pattern: "cat$",
			want:    false,
		},

		// Quantifiers
		{
			name:    "Match one or more with one occurrence",
			text:    "cat",
			pattern: "ca+t",
			want:    true,
		},
		{
			name:    "Match one or more with multiple occurrences",
			text:    "caaats",
			pattern: "ca+t",
			want:    true,
		},
		{
			name:    "Not match one or more with zero occurrences",
			text:    "act",
			pattern: "ca+t",
			want:    false,
		},
		{
			name:    "Match zero or one with one occurrence",
			text:    "cat",
			pattern: "ca?t",
			want:    true,
		},
		{
			name:    "Match zero or one with zero occurrences",
			text:    "act",
			pattern: "ca?t",
			want:    true,
		},
		{
			name:    "Not match zero or one with wrong character",
			text:    "cag",
			pattern: "ca?t",
			want:    false,
		},

		// Wildcard
		{
			name:    "Match wildcard",
			text:    "cat",
			pattern: "c.t",
			want:    true,
		},
		{
			name:    "Not match wildcard",
			text:    "car",
			pattern: "c.t",
			want:    false,
		},
		{
			name:    "Match wildcard with one or more",
			text:    "goøö0Ogol",
			pattern: "g.+gol",
			want:    true,
		},

		// Alternation
		{
			name:    "Match alternation first option",
			text:    "a cat",
			pattern: "a (cat|dog)",
			want:    true,
		},
		{
			name:    "Match alternation second option",
			text:    "a dog and cats",
			pattern: "a (cat|dog) and (cat|dog)s",
			want:    true,
		},
		{
			name:    "Not match alternation",
			text:    "a cow",
			pattern: "a (cat|dog)",
			want:    false,
		},

		// Capturing groups and backreferences
		{
			name:    "Match backreference with same text",
			text:    "cat and cat",
			pattern: "(cat) and \\1",
			want:    true,
		},
		{
			name:    "Not match backreference with different text",
			text:    "cat and dog",
			pattern: "(cat) and \\1",
			want:    false,
		},
		{
			name:    "Match complex backreference",
			text:    "grep 101 is doing grep 101 times",
			pattern: "(\\w\\w\\w\\w \\d\\d\\d) is doing \\1 times",
			want:    true,
		},
		{
			name:    "Not match complex backreference with different first part",
			text:    "$?! 101 is doing $?! 101 times",
			pattern: "(\\w\\w\\w \\d\\d\\d) is doing \\1 times",
			want:    false,
		},
		{
			name:    "Match with positive character group and backreference",
			text:    "abcd is abcd, not efg",
			pattern: "([abcd]+) is \\1, not [^xyz]+",
			want:    true,
		},
		{
			name:    "Match with start and end anchors and backreference",
			text:    "this starts and ends with this",
			pattern: "^(\\w+) starts and ends with \\1$",
			want:    true,
		},
		{
			name:    "Match with quantifiers and backreference",
			text:    "once a dreaaamer, always a dreaaamer",
			pattern: "once a (drea+mer), alwaysz? a \\1",
			want:    true,
		},
		{
			name:    "Match with alternation and backreference",
			text:    "bugs here and bugs there",
			pattern: "(b..s|c..e) here and \\1 there",
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewMatcher()
			got, err := matcher.MatchLine(tt.text, tt.pattern)
			if err != nil {
				t.Errorf("MatchLine() error = %v", err)
				return
			}
			if got != tt.want {
				t.Errorf("MatchLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMatchEdgeCases tests edge cases and error handling
func TestMatchEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		pattern     string
		want        bool
		wantErr     bool
		errContains string
	}{
		{
			name:        "Invalid escape sequence",
			text:        "test",
			pattern:     "\\y",
			want:        false,
			wantErr:     true,
			errContains: "invalid escape sequence",
		},
		{
			name:        "Unmatched opening parenthesis",
			text:        "test",
			pattern:     "(test",
			want:        false,
			wantErr:     true,
			errContains: "unmatched opening parenthesis",
		},
		{
			name:    "Empty text",
			text:    "",
			pattern: "test",
			want:    false,
			wantErr: false,
		},
		{
			name:    "Empty pattern",
			text:    "test",
			pattern: "",
			want:    true,
			wantErr: false,
		},
		{
			name:    "Empty text and pattern",
			text:    "",
			pattern: "",
			want:    true,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matcher := NewMatcher()
			got, err := matcher.MatchLine(tt.text, tt.pattern)

			// Check error condition
			if (err != nil) != tt.wantErr {
				t.Errorf("MatchLine() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check error message if error is expected
			if tt.wantErr && err != nil && tt.errContains != "" {
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("MatchLine() error = %v, want error containing %v", err, tt.errContains)
				}
			}

			// Check result if no error
			if err == nil && got != tt.want {
				t.Errorf("MatchLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestMatchWithHelperFunctions tests the internal helper functions
func TestMatchHelperFunctions(t *testing.T) {
	// Test saveState and restoreState
	t.Run("State saving and restoring", func(t *testing.T) {
		matcher := NewMatcher()
		// Set up initial state
		matcher.capturedGroups[1] = "test"
		matcher.groupCaptured[1] = true
		matcher.nextGroupNum = 2

		// Save the state
		savedState := matcher.saveState()

		// Change the current state
		matcher.capturedGroups[1] = "changed"
		matcher.groupCaptured[1] = false
		matcher.nextGroupNum = 3

		// Restore the state
		matcher.restoreState(savedState)

		// Check if state was properly restored
		if matcher.capturedGroups[1] != "test" {
			t.Errorf("restoreState() capturedGroups = %v, want %v", matcher.capturedGroups[1], "test")
		}
		if !matcher.groupCaptured[1] {
			t.Errorf("restoreState() groupCaptured = %v, want %v", matcher.groupCaptured[1], true)
		}
		if matcher.nextGroupNum != 2 {
			t.Errorf("restoreState() nextGroupNum = %v, want %v", matcher.nextGroupNum, 2)
		}
	})

	// Test splitAlternatives
	t.Run("Split alternatives", func(t *testing.T) {
		matcher := NewMatcher()
		pattern := "cat|dog|(mouse|rat)"
		result := matcher.splitAlternatives(pattern)
		expected := []string{"cat", "dog", "(mouse|rat)"}

		if len(result) != len(expected) {
			t.Errorf("splitAlternatives() got %v elements, want %v", len(result), len(expected))
		}

		for i := range expected {
			if i < len(result) && result[i] != expected[i] {
				t.Errorf("splitAlternatives()[%d] = %v, want %v", i, result[i], expected[i])
			}
		}
	})

	// Test ensureGroupCapacity
	t.Run("Ensure group capacity", func(t *testing.T) {
		matcher := NewMatcher()
		initialCapacity := len(matcher.capturedGroups)

		// Request a capacity larger than initial
		newGroupNum := initialCapacity + 5
		matcher.ensureGroupCapacity(newGroupNum)

		if len(matcher.capturedGroups) <= newGroupNum {
			t.Errorf("ensureGroupCapacity() capturedGroups capacity = %v, want > %v",
				len(matcher.capturedGroups), newGroupNum)
		}
		if len(matcher.groupCaptured) <= newGroupNum {
			t.Errorf("ensureGroupCapacity() groupCaptured capacity = %v, want > %v",
				len(matcher.groupCaptured), newGroupNum)
		}
	})
}
