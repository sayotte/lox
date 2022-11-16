package main

import (
	"fmt"
	"strconv"
	"unicode"
)

type TokenType int

const (
	// Single-character tokens
	LEFT_PAREN TokenType = iota
	RIGHT_PAREN
	LEFT_BRACE
	RIGHT_BRACE
	COMMA
	DOT
	MINUS
	PLUS
	SEMICOLON
	SLASH
	STAR

	// 1-2 character tokens
	BANG
	BANG_EQUAL
	EQUAL
	EQUAL_EQUAL
	GREATER
	GREATER_EQUAL
	LESS
	LESS_EQUAL

	// Literals
	IDENTIFIER
	STRING
	NUMBER

	// Keywords
	AND
	CLASS
	ELSE
	FALSE
	FUN
	FOR
	IF
	NIL
	OR
	PRINT
	RETURN
	SUPER
	THIS
	TRUE
	VAR
	WHILE

	EOF
)

var tokenTypeToPrintable = map[TokenType]string{
	// Single-character tokens
	LEFT_PAREN:  "LEFT_PAREN",
	RIGHT_PAREN: "RIGHT_PAREN",
	LEFT_BRACE:  "LEFT_BRACE",
	RIGHT_BRACE: "RIGHT_BRACE",
	COMMA:       "COMMA",
	DOT:         "DOT",
	MINUS:       "MINUS",
	PLUS:        "PLUS",
	SEMICOLON:   "SEMICOLON",
	SLASH:       "SLASH",
	STAR:        "STAR",

	// 1-2 character tokens
	BANG:          "BANG",
	BANG_EQUAL:    "BANG_EQUAL",
	EQUAL:         "EQUAL",
	EQUAL_EQUAL:   "EQUAL_EQUAL",
	GREATER:       "GREATER",
	GREATER_EQUAL: "GREATER_EQUAL",
	LESS:          "LESS",
	LESS_EQUAL:    "LESS_EQUAL",

	// Literals
	IDENTIFIER: "IDENTIFIER",
	STRING:     "STRING",
	NUMBER:     "NUMBER",

	// Keywords
	AND:    "AND",
	CLASS:  "CLASS",
	ELSE:   "ELSE",
	FALSE:  "FALSE",
	FUN:    "FUN",
	FOR:    "FOR",
	IF:     "IF",
	NIL:    "NIL",
	OR:     "OR",
	PRINT:  "PRINT",
	RETURN: "RETURN",
	SUPER:  "SUPER",
	THIS:   "THIS",
	TRUE:   "TRUE",
	VAR:    "VAR",
	WHILE:  "WHILE",

	EOF: "EOF",
}

var identifierToTokenType = map[string]TokenType{
	// Keywords
	"and":    AND,
	"class":  CLASS,
	"else":   ELSE,
	"false":  FALSE,
	"fun":    FUN,
	"for":    FOR,
	"if":     IF,
	"nil":    NIL,
	"or":     OR,
	"print":  PRINT,
	"return": RETURN,
	"super":  SUPER,
	"this":   THIS,
	"true":   TRUE,
	"var":    VAR,
	"while":  WHILE,
}

type Token struct {
	Type    TokenType
	Lexeme  string
	Literal interface{}
	Line    int
}

func (t Token) String() string {
	return tokenTypeToPrintable[t.Type] + " '" + t.Lexeme + "' " + strconv.Itoa(t.Line)
}

type Scanner struct {
	srcRunes []rune
	start    int
	current  int
	line     int
	Tokens   []Token
}

func (s *Scanner) ScanTokens(src string) []Token {
	*s = Scanner{} // reset to zero value
	s.line = 1
	s.Tokens = make([]Token, 0, 8)
	s.srcRunes = []rune(src)

	for !s.isAtEnd() {
		s.start = s.current
		s.scanToken()
	}

	//s.Tokens = append(s.Tokens, Token{EOF, "", nil, s.line})

	return s.Tokens
}

func (s *Scanner) isAtEnd() bool {
	return s.current >= len(s.srcRunes)
}

func (s *Scanner) scanToken() {
	r := s.advance()
	switch r {
	// Single-character tokens
	case '(':
		s.addToken(LEFT_PAREN, nil)
	case ')':
		s.addToken(RIGHT_PAREN, nil)
	case '{':
		s.addToken(LEFT_BRACE, nil)
	case '}':
		s.addToken(RIGHT_BRACE, nil)
	case ',':
		s.addToken(COMMA, nil)
	case '.':
		s.addToken(DOT, nil)
	case '-':
		s.addToken(MINUS, nil)
	case '+':
		s.addToken(PLUS, nil)
	case ';':
		s.addToken(SEMICOLON, nil)
	case '*':
		s.addToken(STAR, nil)

	// 1-2 character tokens
	case '!':
		if s.matchNext('=') {
			s.addToken(BANG_EQUAL, nil)
		} else {
			s.addToken(BANG, nil)
		}
	case '=':
		if s.matchNext('=') {
			s.addToken(EQUAL_EQUAL, nil)
		} else {
			s.addToken(EQUAL, nil)
		}
	case '<':
		if s.matchNext('=') {
			s.addToken(LESS_EQUAL, nil)
		} else {
			s.addToken(LESS, nil)
		}
	case '>':
		if s.matchNext('=') {
			s.addToken(GREATER_EQUAL, nil)
		} else {
			s.addToken(GREATER, nil)
		}

	// comments and slash
	case '/':
		if s.matchNext('/') {
			for s.peek() != '\n' && !s.isAtEnd() {
				s.advance()
			}
		} else {
			s.addToken(SLASH, nil)
		}

	// ignore whitespace (mostly)
	case ' ':
		fallthrough
	case '\r':
		fallthrough
	case '\t':
		break
	case '\n':
		s.line++
		break

	// literals
	case '"':
		s.scanString()

	default:
		if unicode.IsDigit(r) {
			s.scanNumber()
		} else if unicode.IsLetter(r) {
			s.scanIdentifier()
		} else {
			panic(fmt.Sprintf("unexpected character %q", r))
		}
	}

}

func (s *Scanner) advance() rune {
	ret := s.srcRunes[s.current]
	s.current++
	return ret
}

func (s *Scanner) matchNext(expected rune) bool {
	if s.isAtEnd() {
		return false
	}
	if s.srcRunes[s.current] != expected {
		return false
	}
	s.current++
	return true
}

func (s *Scanner) peek() rune {
	if s.isAtEnd() {
		return 0
	}
	return s.srcRunes[s.current]
}

func (s *Scanner) peekNext() rune {
	if s.current+1 >= len(s.srcRunes) {
		return 0
	}
	return s.srcRunes[s.current+1]
}

func (s *Scanner) scanString() {
	for s.peek() != '"' && !s.isAtEnd() {
		if s.peek() == '\n' {
			s.line++
		}
		s.advance()
	}

	if s.isAtEnd() {
		panic(fmt.Sprintf("Unterminated string starting on line %d", s.line))
	}

	s.advance() // consume the terminating '"'

	literal := string(s.srcRunes[s.start+1 : s.current-1]) // note we trim the leading/trailing quotes
	s.addToken(STRING, literal)
}

func (s *Scanner) scanNumber() {
	for unicode.IsDigit(s.peek()) {
		s.advance()
	}
	if s.peek() == '.' && unicode.IsDigit(s.peekNext()) {
		s.advance() // consume the '.'
		for unicode.IsDigit(s.peek()) {
			s.advance()
		}
	}
	str := string(s.srcRunes[s.start:s.current])
	literal, err := strconv.ParseFloat(str, 64)
	if err != nil {
		panic(fmt.Sprintf("error parsing float %q on line %d: %s", str, s.line, err))
	}
	s.addToken(NUMBER, literal)
}

func (s *Scanner) scanIdentifier() {
	for unicode.IsLetter(s.peek()) || unicode.IsDigit(s.peek()) {
		s.advance()
	}
	lexeme := string(s.srcRunes[s.start:s.current])
	tokenType, found := identifierToTokenType[lexeme]
	if !found {
		tokenType = IDENTIFIER
	}

	s.addToken(tokenType, nil)
}

func (s *Scanner) addToken(typ TokenType, literal interface{}) {
	lexeme := string(s.srcRunes[s.start:s.current])
	s.Tokens = append(s.Tokens, Token{typ, lexeme, literal, s.line})
}
