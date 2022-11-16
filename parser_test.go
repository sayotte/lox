package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestParser_Parse(t *testing.T) {
	testCases := map[string]struct {
		inTokens       []Token
		expected       []Stmt
		errExpected    bool
		expectedErrStr string
	}{
		"primary string": {
			inTokens: []Token{
				{
					Type:    STRING,
					Literal: "string",
				},
				{Type: SEMICOLON},
			},
			expected: []Stmt{ExprStmt{Literal{
				Value: "string",
			}}},
		},
		"parenthetical false": {
			inTokens: []Token{
				{Type: LEFT_PAREN},
				{Type: FALSE},
				{Type: RIGHT_PAREN},
				{Type: SEMICOLON},
			},
			expected: []Stmt{ExprStmt{Grouping{
				Expression: Literal{Value: false},
			}}},
		},
		"unary bang": {
			inTokens: []Token{
				{Type: BANG},
				{Type: NUMBER, Literal: 123.0},
				{Type: SEMICOLON},
			},
			expected: []Stmt{ExprStmt{Unary{
				Operator: Token{Type: BANG},
				Right:    Literal{Value: 123.0},
			}}},
		},
		"factor multiply": {
			inTokens: []Token{
				{Type: NUMBER, Literal: 1.0},
				{Type: STAR},
				{Type: NUMBER, Literal: 2.0},
				{Type: SEMICOLON},
			},
			expected: []Stmt{ExprStmt{Binary{
				Left:     Literal{Value: 1.0},
				Operator: Token{Type: STAR},
				Right:    Literal{Value: 2.0},
			}}},
		},
		"term plus": {
			inTokens: []Token{
				{Type: NUMBER, Literal: 1.0},
				{Type: PLUS},
				{Type: NUMBER, Literal: 2.0},
				{Type: SEMICOLON},
			},
			expected: []Stmt{ExprStmt{Binary{
				Left:     Literal{Value: 1.0},
				Operator: Token{Type: PLUS},
				Right:    Literal{Value: 2.0},
			}}},
		},
		"comparison greater": {
			inTokens: []Token{
				{Type: NUMBER, Literal: 1.0},
				{Type: GREATER},
				{Type: NUMBER, Literal: 2.0},
				{Type: SEMICOLON},
			},
			expected: []Stmt{ExprStmt{Binary{
				Left:     Literal{Value: 1.0},
				Operator: Token{Type: GREATER},
				Right:    Literal{Value: 2.0},
			}}},
		},
		"equality equal equal": {
			inTokens: []Token{
				{Type: NUMBER, Literal: 1.0},
				{Type: EQUAL_EQUAL},
				{Type: NUMBER, Literal: 2.0},
				{Type: SEMICOLON},
			},
			expected: []Stmt{ExprStmt{Binary{
				Left:     Literal{Value: 1.0},
				Operator: Token{Type: EQUAL_EQUAL},
				Right:    Literal{Value: 2.0},
			}}},
		},
		"equality comparison term factor grouping unary": {
			// !(1 + 1) == 2 * 2 > 3
			inTokens: []Token{
				{Type: BANG},
				{Type: LEFT_PAREN},
				{Type: NUMBER, Literal: 1.0},
				{Type: PLUS},
				{Type: NUMBER, Literal: 1.1},
				{Type: RIGHT_PAREN},
				{Type: EQUAL_EQUAL},
				{Type: NUMBER, Literal: 2.0},
				{Type: STAR},
				{Type: NUMBER, Literal: 2.1},
				{Type: GREATER},
				{Type: NUMBER, Literal: 3.0},
				{Type: SEMICOLON},
			},
			expected: []Stmt{ExprStmt{Binary{
				Left: Unary{
					Operator: Token{Type: BANG},
					Right: Grouping{
						Expression: Binary{
							Left:     Literal{Value: 1.0},
							Operator: Token{Type: PLUS},
							Right:    Literal{Value: 1.1},
						},
					},
				},
				Operator: Token{Type: EQUAL_EQUAL},
				Right: Binary{
					Left: Binary{
						Left:     Literal{Value: 2.0},
						Operator: Token{Type: STAR},
						Right:    Literal{Value: 2.1},
					},
					Operator: Token{Type: GREATER},
					Right:    Literal{3.0},
				},
			}}},
		},
		"variable declaration": {
			inTokens: []Token{
				{Type: VAR},
				{Type: IDENTIFIER, Lexeme: "myVar"},
				{Type: EQUAL},
				{Type: NUMBER, Literal: 1},
				{Type: SEMICOLON},
			},
			expected: []Stmt{VariableStmt{
				Name:        Token{Type: IDENTIFIER, Lexeme: "myVar"},
				Initializer: Literal{Value: 1},
			}},
		},
		"block": {
			inTokens: []Token{
				{Type: LEFT_BRACE},
				{Type: VAR},
				{Type: IDENTIFIER, Lexeme: "myVar"},
				{Type: SEMICOLON},
				{Type: RIGHT_BRACE},
			},
			expected: []Stmt{
				BlockStmt{
					Statements: []Stmt{
						VariableStmt{
							Name: Token{Type: IDENTIFIER, Lexeme: "myVar"},
						},
					},
				},
			},
		},
		"if/else": {
			// if (a) print a; else print b;
			inTokens: []Token{
				{Type: IF},
				{Type: LEFT_PAREN},
				{Type: IDENTIFIER, Lexeme: "a"},
				{Type: RIGHT_PAREN},
				{Type: PRINT},
				{Type: IDENTIFIER, Lexeme: "a"},
				{Type: SEMICOLON},
				{Type: ELSE},
				{Type: PRINT},
				{Type: IDENTIFIER, Lexeme: "b"},
				{Type: SEMICOLON},
			},
			expected: []Stmt{
				IfStmt{
					Condition: Variable{Name: Token{Type: IDENTIFIER, Lexeme: "a"}, Unique: 0},
					Then:      PrintStmt{Variable{Name: Token{Type: IDENTIFIER, Lexeme: "a"}, Unique: 1}},
					Else:      PrintStmt{Variable{Name: Token{Type: IDENTIFIER, Lexeme: "b"}, Unique: 2}},
				},
			},
		},
		"empty for": {
			inTokens: []Token{
				{Type: FOR},
				{Type: LEFT_PAREN},
				{Type: SEMICOLON},
				{Type: SEMICOLON},
				{Type: RIGHT_PAREN},
				{Type: PRINT},
				{Type: IDENTIFIER, Lexeme: "a"},
				{Type: SEMICOLON},
			},
			expected: []Stmt{
				WhileStmt{
					Condition: Literal{Value: true},
					Body:      PrintStmt{Expression: Variable{Name: Token{Type: IDENTIFIER, Lexeme: "a"}}},
				},
			},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			p := &Parser{Tokens: tc.inTokens}
			actual, err := p.Parse()
			if !tc.errExpected && err != nil {
				t.Errorf("unexpected error: %s", err)
			} else if tc.errExpected && err == nil {
				t.Error("expected error, didn't get one")
			} else if tc.errExpected && err != nil && !strings.Contains(err.Error(), tc.expectedErrStr) {
				t.Errorf("expected error containing %q, got %q", tc.expectedErrStr, err)
			} else if !reflect.DeepEqual(actual, tc.expected) {
				t.Errorf("%s != %s", actual, tc.expected)
			}
		})
	}
}
