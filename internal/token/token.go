package token

// Type is the type of a token.
type Type string

// Token represents a lexical token.
type Token struct {
	Type    Type
	Literal string
	Line    int
	Column  int
}

const (
	// Special tokens
	ILLEGAL Type = "ILLEGAL" // An unknown or invalid token
	EOF     Type = "EOF"     // End of file

	// Literals
	IDENT  Type = "IDENT"  // a, key, name
	INT    Type = "INT"    // 12345
	FLOAT  Type = "FLOAT"  // 123.45
	STRING Type = "STRING" // "hello world"

	// Delimiters
	LBRACE Type = "{"
	RBRACE Type = "}"
	LBRACK Type = "["
	RBRACK Type = "]"
	COMMA  Type = ","
	COLON  Type = ":"

	// Keywords
	TRUE  Type = "TRUE"
	FALSE Type = "FALSE"
	NULL  Type = "NULL"

	// Comments and Whitespace
	COMMENT Type = "COMMENT" // # a comment
	NEWLINE Type = "NEWLINE" // \n
)

var keywords = map[string]Type{
	"true":  TRUE,
	"false": FALSE,
	"null":  NULL,
}

// LookupIdent checks the keywords table for an identifier.
// If the identifier is a keyword, it returns the keyword's token type.
// Otherwise, it returns IDENT.
func LookupIdent(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
