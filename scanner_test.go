package main

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func compareTokens(left, right Token) (bool, string) {
	var reasons []string
	if left.Type != right.Type {
		why := fmt.Sprintf("type %s != %s", tokenTypeToPrintable[left.Type], tokenTypeToPrintable[right.Type])
		reasons = append(reasons, why)
	}
	if left.Lexeme != right.Lexeme {
		why := fmt.Sprintf("lexeme %q != %q", left.Lexeme, right.Lexeme)
		reasons = append(reasons, why)
	}
	if !reflect.DeepEqual(left.Literal, right.Literal) {
		why := fmt.Sprintf("literal %+q != %+q", left.Literal, right.Literal)
		reasons = append(reasons, why)
	}
	if left.Line != right.Line {
		why := fmt.Sprintf("line %d != %d", left.Line, right.Line)
		reasons = append(reasons, why)
	}

	if len(reasons) != 0 {
		return false, strings.Join(reasons, "\n")
	}
	return true, ""
}

func TestScanner_ScanTokens(t *testing.T) {
	testCases := map[string]struct {
		src      string
		expected []Token
	}{
		"number-with-decimal": {
			src:      "10.10",
			expected: []Token{{NUMBER, "10.10", 10.1, 1}},
		},
		"numbers-whitespace-delimited": {
			src: "1 2",
			expected: []Token{
				{NUMBER, "1", 1.0, 1},
				{NUMBER, "2", 2.0, 1},
			},
		},
		"numbers-and-operator": {
			src: "1* 3",
			expected: []Token{
				{NUMBER, "1", 1.0, 1},
				{STAR, "*", nil, 1},
				{NUMBER, "3", 3.0, 1},
			},
		},
		"string": {
			src: "\"string\"",
			expected: []Token{
				{STRING, "\"string\"", "string", 1},
			},
		},
		"multiline-string": {
			src: "\"line 1\nline 2\"",
			expected: []Token{
				{STRING, "\"line 1\nline 2\"", "line 1\nline 2", 2},
			},
		},
		"identifier": {
			src: "myVar",
			expected: []Token{
				{IDENTIFIER, "myVar", nil, 1},
			},
		},
		"keyword": {
			src: "and",
			expected: []Token{
				{AND, "and", nil, 1},
			},
		},
		"2 character operators": {
			src: "!!====>=><=<",
			expected: []Token{
				{BANG, "!", nil, 1},
				{BANG_EQUAL, "!=", nil, 1},
				{EQUAL_EQUAL, "==", nil, 1},
				{EQUAL, "=", nil, 1},
				{GREATER_EQUAL, ">=", nil, 1},
				{GREATER, ">", nil, 1},
				{LESS_EQUAL, "<=", nil, 1},
				{LESS, "<", nil, 1},
			},
		},
		"toks separated by comments": {
			src: "1 / // k\n2",
			expected: []Token{
				{NUMBER, "1", 1.0, 1},
				{SLASH, "/", nil, 1},
				{NUMBER, "2", 2.0, 2},
			},
		},
		"ignore newline but increment line": {
			src: "\n1",
			expected: []Token{
				{NUMBER, "1", 1.0, 2},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			s := &Scanner{}
			actual := s.ScanTokens(tc.src)
			if len(actual) != len(tc.expected) {
				t.Errorf("expected %d tokens, got %d", len(tc.expected), len(actual))
			}
			for i := range actual {
				same, why := compareTokens(actual[i], tc.expected[i])
				if !same {
					t.Errorf("token %d incorrect:\n%s", i, why)
				}
			}
		})
	}
}
