package lexer

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/KimNorgaard/go-maml/internal/token"
)

// Lexer holds the state for tokenizing MAML source.
type Lexer struct {
	r      *bufio.Reader
	buf    bytes.Buffer
	ch     rune
	line   int
	column int
}

// New creates and returns a new Lexer.
func New(r io.Reader) *Lexer {
	l := &Lexer{
		r:      bufio.NewReader(r),
		line:   1,
		column: 1,
	}
	l.readRune()
	return l
}

// NextToken scans the input and returns the next token.
func (l *Lexer) NextToken() token.Token { //nolint:gocognit
	l.skipWhitespace()
	tok := token.Token{Line: l.line, Column: l.column}
	switch l.ch {
	case '{', '}', '[', ']', ',', ':':
		tok.Type = token.Type(l.ch)
		tok.Literal = string(l.ch)
	case '\r':
		if l.peekRune() == '\n' {
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
	case -1: // Corresponds to io.EOF
		tok.Type = token.EOF
		tok.Literal = ""
		return tok
	default:
		if isDigit(l.ch) || (l.ch == '-' && (isDigit(l.peekRune()) || l.peekRune() == '.')) {
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
		if l.ch == utf8.RuneError {
			tok.Literal = "invalid utf-8"
		} else {
			tok.Literal = string(l.ch)
		}
	}
	l.advance()
	return tok
}

func (l *Lexer) readRune() {
	r, _, err := l.r.ReadRune()
	if err != nil {
		l.ch = -1
		return
	}
	l.ch = r
}

func (l *Lexer) advance() {
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
	l.readRune()
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
	l.buf.Reset()
	for l.ch != '\n' && l.ch != -1 {
		if isForbiddenControlChar(l.ch) {
			return fmt.Sprintf("forbidden control character U+%04X in comment", l.ch), false
		}
		l.buf.WriteRune(l.ch)
		l.advance()
	}
	return l.buf.String(), true
}

func (l *Lexer) readIdentifier() string {
	l.buf.Reset()
	for isIdentifierChar(l.ch) {
		l.buf.WriteRune(l.ch)
		l.advance()
	}
	return l.buf.String()
}

func (l *Lexer) readPotentialNumberOrIdentifier() string {
	l.buf.Reset()
	for isIdentifierChar(l.ch) || l.ch == '.' || l.ch == 'e' || l.ch == 'E' || l.ch == '+' {
		l.buf.WriteRune(l.ch)
		l.advance()
	}
	return l.buf.String()
}

func (l *Lexer) readString() (string, bool) {
	if l.peekRune() == '"' && l.peekNextRune() == '"' {
		return l.readMultilineString()
	}
	return l.readSingleLineString()
}

func (l *Lexer) readEscapeSequence() (rune, bool, string) {
	l.advance() // consume backslash
	switch l.ch {
	case 'b', 'f', 'n', 'r', 't', '"', '\\', '/':
		return unescape(l.ch), true, ""
	case 'u':
		val, ok := l.readHex(4)
		if !ok {
			return 0, false, "invalid unicode escape"
		}
		if val >= 0xD800 && val <= 0xDFFF {
			return 0, false, "invalid unicode scalar value (surrogate pair)"
		}
		return val, true, ""
	default:
		return 0, false, fmt.Sprintf("invalid escape sequence \\%c", l.ch)
	}
}

func (l *Lexer) readSingleLineString() (string, bool) {
	l.advance() // consume opening quote
	l.buf.Reset()
	for {
		if l.ch == '"' {
			l.advance() // consume closing quote
			return l.buf.String(), true
		}
		if l.ch == '\n' || l.ch == -1 {
			return "unterminated string", false
		}

		if l.ch == '\\' {
			r, ok, errMsg := l.readEscapeSequence()
			if !ok {
				return errMsg, false
			}
			l.buf.WriteRune(r)
		} else {
			if l.ch == utf8.RuneError {
				return "invalid utf-8 sequence in string", false
			}
			if isForbiddenControlChar(l.ch) {
				return fmt.Sprintf("forbidden control character U+%04X in string", l.ch), false
			}
			l.buf.WriteRune(l.ch)
		}
		l.advance()
	}
}

func (l *Lexer) readMultilineString() (string, bool) {
	l.advance() // consume first quote
	l.advance() // consume second quote
	l.advance() // consume third quote
	if l.ch == '\n' {
		l.advance()
	}
	l.buf.Reset()
	for {
		if l.ch == -1 {
			return "unterminated multiline string", false
		}
		if l.ch == '"' && l.peekRune() == '"' && l.peekNextRune() == '"' {
			l.advance()
			l.advance()
			l.advance()
			return l.buf.String(), true
		}
		if l.ch == utf8.RuneError {
			return "invalid utf-8", false
		}
		if l.ch != '\n' && isForbiddenControlChar(l.ch) {
			return fmt.Sprintf("forbidden control character U+%04X in multiline string", l.ch), false
		}
		l.buf.WriteRune(l.ch)
		l.advance()
	}
}

func (l *Lexer) readHex(n int) (rune, bool) {
	var val rune
	for range n {
		l.advance()
		var d rune
		switch {
		case '0' <= l.ch && l.ch <= '9':
			d = l.ch - '0'
		case 'a' <= l.ch && l.ch <= 'f':
			d = l.ch - 'a' + 10
		case 'A' <= l.ch && l.ch <= 'F':
			d = l.ch - 'A' + 10
		default:
			return 0, false
		}
		val = val*16 + d
	}
	return val, true
}

func (l *Lexer) peekRune() rune {
	// Prioritize the returned slice, as Peek can return both bytes and an error
	bytes, _ := l.r.Peek(utf8.UTFMax)
	if len(bytes) == 0 {
		return 0
	}
	r, _ := utf8.DecodeRune(bytes)
	return r
}

func (l *Lexer) peekNextRune() rune {
	// Prioritize the returned slice, as Peek can return both bytes and an error
	bytes, _ := l.r.Peek(utf8.UTFMax * 2)
	if len(bytes) == 0 {
		return 0
	}

	_, firstRuneSize := utf8.DecodeRune(bytes)
	if len(bytes) <= firstRuneSize { // Not enough bytes for a second rune.
		return 0
	}

	r, _ := utf8.DecodeRune(bytes[firstRuneSize:])
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

func consumeDigits(s string, i int) int {
	for i < len(s) && isDigit(rune(s[i])) {
		i++
	}
	return i
}

func parseIntegerPart(s string, i int) (newIndex int, ok bool) {
	integerStart := i
	i = consumeDigits(s, i)
	if i == integerStart {
		return i, false // No digits found.
	}
	integerPart := s[integerStart:i]
	if len(integerPart) > 1 && integerPart[0] == '0' {
		return i, false // Leading zeros are not allowed.
	}
	return i, true
}

func parseFractionalPart(s string, i int) (newIndex int, ok bool, isFloat bool) {
	if i >= len(s) || s[i] != '.' {
		return i, true, false
	}
	i++ // Consume '.'.
	fractionStart := i
	i = consumeDigits(s, i)
	if i == fractionStart {
		return i, false, true // No digits after '.'.
	}
	return i, true, true
}

func parseExponentPart(s string, i int) (newIndex int, ok bool, isFloat bool) {
	if i >= len(s) || (s[i] != 'e' && s[i] != 'E') {
		return i, true, false
	}
	i++ // Consume 'e' or 'E'.
	if i < len(s) && (s[i] == '+' || s[i] == '-') {
		i++
	}
	exponentStart := i
	i = consumeDigits(s, i)
	if i == exponentStart {
		return i, false, true // No digits in exponent.
	}
	return i, true, true
}

func ParseAsNumber(s string) (token.Type, bool) {
	if len(s) == 0 {
		return token.ILLEGAL, false
	}
	i, isFloat := 0, false

	// Optional sign.
	if s[i] == '-' {
		if len(s) == 1 {
			return token.ILLEGAL, false
		}
		i++
	}

	// Integer part.
	var ok bool
	i, ok = parseIntegerPart(s, i)
	if !ok {
		return token.ILLEGAL, false
	}

	// Fractional part.
	var fracIsFloat bool
	i, ok, fracIsFloat = parseFractionalPart(s, i)
	if !ok {
		return token.ILLEGAL, false
	}
	if fracIsFloat {
		isFloat = true
	}

	// Exponent part.
	var expIsFloat bool
	i, ok, expIsFloat = parseExponentPart(s, i)
	if !ok {
		return token.ILLEGAL, false
	}
	if expIsFloat {
		isFloat = true
	}

	// Must consume the whole string.
	if i != len(s) {
		return token.ILLEGAL, false
	}

	if isFloat {
		return token.FLOAT, true
	}
	return token.INT, true
}
