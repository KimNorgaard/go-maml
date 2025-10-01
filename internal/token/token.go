package token

// TokenType is the type of a token.
type TokenType string

// Token represents a lexical token.
type Token struct {
	Type    TokenType
	Literal string
	Line    int
	Column  int
}

const (
	// Special tokens
	ILLEGAL TokenType = "ILLEGAL" // An unknown or invalid token
	EOF     TokenType = "EOF"     // End of file

	// Literals
	IDENT  TokenType = "IDENT"  // a, key, name
	INT    TokenType = "INT"    // 12345
	FLOAT  TokenType = "FLOAT"  // 123.45
	STRING TokenType = "STRING" // "hello world"

	// Delimiters
	LBRACE TokenType = "{"
	RBRACE TokenType = "}"
	LBRACK TokenType = "["
	RBRACK TokenType = "]"
	COMMA  TokenType = ","
	COLON  TokenType = ":"

	// Keywords
	TRUE  TokenType = "TRUE"
	FALSE TokenType = "FALSE"
	NULL  TokenType = "NULL"

	// Comments and Whitespace
	COMMENT TokenType = "COMMENT" // # a comment
	NEWLINE TokenType = "NEWLINE" // \n
)

var keywords = map[string]TokenType{
	"true":  TRUE,
	"false": FALSE,
	"null":  NULL,
}

// LookupIdent checks the keywords table for an identifier.
// If the identifier is a keyword, it returns the keyword's token type.
// Otherwise, it returns IDENT.
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
