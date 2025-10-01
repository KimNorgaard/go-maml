package lexer

import (
	"bytes"
	"fmt"
	"unicode/utf8"

	"github.com/KimNorgaard/go-maml/token"
)

// Lexer holds the state for tokenizing MAML source.
type Lexer struct {
	input        []byte
	position     int
	readPosition int
	ch           rune
	line         int
	column       int
}

// New creates and returns a new Lexer.
func New(input []byte) *Lexer {
	l := &Lexer{input: input, line: 1, column: 1}
	l.readChar()
	return l
}

// NextToken scans the input and returns the next token.
func (l *Lexer) NextToken() token.Token {
	l.skipWhitespace()
	tok := token.Token{Line: l.line, Column: l.column}
	switch l.ch {
	case '{', '}', '[', ']', ',', ':':
		tok.Type = token.TokenType(l.ch)
		tok.Literal = string(l.ch)
	case '\r':
		if l.peekChar() == '\n' {
			// This is a CRLF newline. Consume the \r.
			// The \n will be the current char for the next token's advance().
			l.advance()
			tok.Type = token.NEWLINE
			tok.Literal = "\r\n"
		} else {
			// Standalone CR is an illegal character per the spec.
			tok.Type = token.ILLEGAL
			tok.Literal = "\r"
		}
	case '\n':
		tok.Type = token.NEWLINE
		tok.Literal = "\n"
	case '#':
		lit, ok := l.readComment()
		if !ok {
			tok.Type = token.ILLEGAL
		} else {
			tok.Type = token.COMMENT
		}
		tok.Literal = lit
		return tok
	case '"':
		lit, ok := l.readString()
		if !ok {
			tok.Type = token.ILLEGAL
		} else {
			tok.Type = token.STRING
		}
		tok.Literal = lit
		return tok
	case 0:
		tok.Type = token.EOF
		tok.Literal = ""
		return tok
	case -1:
		tok.Type = token.ILLEGAL
		tok.Literal = "invalid utf-8"
	default:
		if isDigit(l.ch) || (l.ch == '-' && (isDigit(l.peekChar()) || l.peekChar() == '.')) {
			literal := l.readPotentialNumberOrIdentifier()
			if typ, ok := ParseAsNumber(literal); ok {
				tok.Type = typ
			} else {
				tok.Type = token.IDENT
			}
			tok.Literal = literal
			return tok
		}
		if isIdentifierChar(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			return tok
		}
		tok.Type = token.ILLEGAL
		tok.Literal = string(l.ch)
	}
	l.advance()
	return tok
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
		l.position = l.readPosition // Important for correct slicing at EOF
	} else {
		r, size := utf8.DecodeRune(l.input[l.readPosition:])
		if r == utf8.RuneError {
			l.ch = -1
		} else {
			l.ch = r
		}
		l.position = l.readPosition
		l.readPosition += size
	}
}

func (l *Lexer) advance() {
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
	l.readChar()
	l.column++
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' {
		l.advance()
	}
}

func (l *Lexer) readComment() (string, bool) {
	l.advance() // consume '#'
	for l.ch == ' ' || l.ch == '\t' {
		l.advance() // consume leading whitespace
	}
	startPos := l.position
	for l.ch != '\n' && l.ch != 0 {
		if isForbiddenControlChar(l.ch) {
			return fmt.Sprintf("forbidden control character U+%04X in comment", l.ch), false
		}
		l.advance()
	}
	return string(l.input[startPos:l.position]), true
}

func (l *Lexer) readIdentifier() string {
	startPos := l.position
	for isIdentifierChar(l.ch) {
		l.advance()
	}
	return string(l.input[startPos:l.position])
}

func (l *Lexer) readPotentialNumberOrIdentifier() string {
	startPos := l.position
	for isIdentifierChar(l.ch) || l.ch == '.' || l.ch == 'e' || l.ch == 'E' || l.ch == '+' {
		l.advance()
	}
	return string(l.input[startPos:l.position])
}

// ParseAsNumber validates if a literal string conforms to the MAML number ABNF.
// It is a pure function that does not modify the lexer state.
//
// Valid examples: "0", "-10", "1.23", "5e-10", "-0.5E+2"
// Invalid examples: "01", "--1", "1.2.3", "5e-", "e10"
func ParseAsNumber(s string) (token.TokenType, bool) {
	if len(s) == 0 {
		return token.ILLEGAL, false
	}
	i, isFloat := 0, false
	if i < len(s) && s[i] == '-' {
		if len(s) == 1 {
			return token.ILLEGAL, false
		}
		i++
	}
	integerStart := i
	for i < len(s) && isDigit(rune(s[i])) {
		i++
	}
	if i == integerStart {
		return token.ILLEGAL, false
	}
	integerPart := s[integerStart:i]
	if len(integerPart) > 1 && integerPart[0] == '0' {
		return token.ILLEGAL, false
	}
	if i < len(s) && s[i] == '.' {
		isFloat = true
		i++
		fractionStart := i
		for i < len(s) && isDigit(rune(s[i])) {
			i++
		}
		if i == fractionStart {
			return token.ILLEGAL, false
		}
	}
	if i < len(s) && (s[i] == 'e' || s[i] == 'E') {
		isFloat = true
		i++
		if i < len(s) && (s[i] == '+' || s[i] == '-') {
			i++
		}
		exponentStart := i
		for i < len(s) && isDigit(rune(s[i])) {
			i++
		}
		if i == exponentStart {
			return token.ILLEGAL, false
		}
	}
	if i != len(s) {
		return token.ILLEGAL, false
	}
	if isFloat {
		return token.FLOAT, true
	}
	return token.INT, true
}

func (l *Lexer) readString() (string, bool) {
	if l.peekChar() == '"' && l.peekNextChar() == '"' {
		return l.readMultilineString()
	}
	return l.readSingleLineString()
}

func (l *Lexer) readSingleLineString() (string, bool) {
	l.advance()
	var buf bytes.Buffer
	for {
		if l.ch == '"' {
			l.advance()
			return buf.String(), true
		}
		if l.ch == '\n' || l.ch == 0 {
			return "unterminated string", false
		}
		if l.ch == -1 {
			return "invalid utf-8 sequence in string", false
		}
		if l.ch == '\\' {
			l.advance()
			switch l.ch {
			case 'b', 'f', 'n', 'r', 't', '"', '\\', '/':
				buf.WriteRune(unescape(l.ch))
			case 'u':
				val, ok := l.readHex(4)
				if !ok {
					return "invalid unicode escape", false
				}
				if val >= 0xD800 && val <= 0xDFFF {
					return "invalid unicode scalar value (surrogate pair)", false
				}
				buf.WriteRune(val)
			default:
				return fmt.Sprintf("invalid escape sequence \\%c", l.ch), false
			}
		} else {
			if isForbiddenControlChar(l.ch) {
				return fmt.Sprintf("forbidden control character U+%04X in string", l.ch), false
			}
			buf.WriteRune(l.ch)
		}
		l.advance()
	}
}

func (l *Lexer) readMultilineString() (string, bool) {
	l.advance()
	l.advance()
	l.advance()
	if l.ch == '\n' {
		l.advance()
	}
	var buf bytes.Buffer
	for {
		if l.ch == 0 {
			return "unterminated multiline string", false
		}
		if l.ch == -1 {
			return "invalid utf-8 sequence in multiline string", false
		}
		if l.ch == '"' && l.peekChar() == '"' && l.peekNextChar() == '"' {
			l.advance()
			l.advance()
			l.advance()
			return buf.String(), true
		}
		if l.ch != '\n' && isForbiddenControlChar(l.ch) {
			return fmt.Sprintf("forbidden control character U+%04X in multiline string", l.ch), false
		}
		buf.WriteRune(l.ch)
		l.advance()
	}
}

func (l *Lexer) readHex(n int) (rune, bool) {
	var val rune
	for range n {
		l.advance()
		var d rune
		if '0' <= l.ch && l.ch <= '9' {
			d = l.ch - '0'
		} else if 'a' <= l.ch && l.ch <= 'f' {
			d = l.ch - 'a' + 10
		} else if 'A' <= l.ch && l.ch <= 'F' {
			d = l.ch - 'A' + 10
		} else {
			return 0, false
		}
		val = val*16 + d
	}
	return val, true
}

func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRune(l.input[l.readPosition:])
	return r
}

func (l *Lexer) peekNextChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	_, size := utf8.DecodeRune(l.input[l.readPosition:])
	nextPos := l.readPosition + size
	if nextPos >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRune(l.input[nextPos:])
	return r
}

func isForbiddenControlChar(ch rune) bool {
	return (ch >= 0x00 && ch <= 0x08) || (ch >= 0x0A && ch <= 0x1F) || ch == 0x7F
}

func isDigit(ch rune) bool {
	return '0' <= ch && ch <= '9'
}

func isIdentifierChar(ch rune) bool {
	return ('a' <= ch && ch <= 'z') || ('A' <= ch && ch <= 'Z') || isDigit(ch) || ch == '_' || ch == '-'
}

func unescape(ch rune) rune {
	switch ch {
	case 'b':
		return '\b'
	case 'f':
		return '\f'
	case 'n':
		return '\n'
	case 'r':
		return '\r'
	case 't':
		return '\t'
	case '"':
		return '"'
	case '\\':
		return '\\'
	case '/':
		return '/'
	}
	return 0
}
