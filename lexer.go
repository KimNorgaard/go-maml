package maml

import "github.com/KimNorgaard/go-maml/token"

// Lexer transforms a MAML source string into a stream of tokens.
type Lexer struct {
	input        []byte
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           byte // current char under examination
	line         int  // current line number
	col          int  // current column number
}

// NewLexer creates a new Lexer.
func NewLexer(input []byte) *Lexer {
	l := &Lexer{input: input, line: 1, col: 0}
	l.readChar()
	return l
}

// readChar gives us the next character and advances our position in the input string.
func (l *Lexer) readChar() {
	if l.ch == '\n' {
		l.line++
		l.col = 0
	}

	if l.readPosition >= len(l.input) {
		l.ch = 0 // NUL character, signifies EOF
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
	l.col++
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespace()

	// Record the starting position of the token
	tok.Line = l.line
	tok.Column = l.col

	switch l.ch {
	case '{':
		tok.Type = token.LBRACE
		tok.Literal = string(l.ch)
	case '}':
		tok.Type = token.RBRACE
		tok.Literal = string(l.ch)
	case '[':
		tok.Type = token.LBRACK
		tok.Literal = string(l.ch)
	case ']':
		tok.Type = token.RBRACK
		tok.Literal = string(l.ch)
	case ',':
		tok.Type = token.COMMA
		tok.Literal = string(l.ch)
	case ':':
		tok.Type = token.COLON
		tok.Literal = string(l.ch)
	case '\n':
		tok.Type = token.NEWLINE
		tok.Literal = string(l.ch)
	case '#':
		tok.Type = token.COMMENT
		tok.Literal = l.readComment()
		return tok
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case 0:
		tok.Literal = ""
		tok.Type = token.EOF
	default:
		if isLetter(l.ch) {
			if l.ch == '-' && isDigit(l.peekChar()) {
				tok.Type, tok.Literal = l.readNumber()
			} else {
				tok.Literal = l.readIdentifier()
				tok.Type = token.LookupIdent(tok.Literal)
			}
			return tok
		} else if isDigit(l.ch) {
			tok.Type, tok.Literal = l.readNumber()
			return tok
		} else {
			tok.Type = token.ILLEGAL
			tok.Literal = string(l.ch)
		}
	}

	l.readChar()
	return tok
}

// peekChar looks at the next character without advancing the position.
func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
		l.readChar()
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return string(l.input[position:l.position])
}

func (l *Lexer) readNumber() (token.TokenType, string) {
	position := l.position
	tokType := token.INT
	if l.ch == '-' {
		l.readChar()
	}
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' {
		tokType = token.FLOAT
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return tokType, string(l.input[position:l.position])
}

func (l *Lexer) readComment() string {
	position := l.position
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
	return string(l.input[position:l.position])
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_' || ch == '-'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}
