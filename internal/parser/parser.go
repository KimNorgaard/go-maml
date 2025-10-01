package parser

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/lexer"
	"github.com/KimNorgaard/go-maml/internal/token"
)

type prefixParseFn func() ast.Expression

// Parser holds the state of the parser.
type Parser struct {
	l      *lexer.Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.TokenType]prefixParseFn
}

// New creates a new parser.
func New(l *lexer.Lexer) *Parser {
	p := &Parser{
		l:      l,
		errors: []string{},
	}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.FLOAT, p.parseFloatLiteral)
	p.registerPrefix(token.STRING, p.parseStringLiteral)
	p.registerPrefix(token.TRUE, p.parseBooleanLiteral)
	p.registerPrefix(token.FALSE, p.parseBooleanLiteral)
	p.registerPrefix(token.NULL, p.parseNullLiteral)
	p.registerPrefix(token.LBRACK, p.parseArrayLiteral)
	p.registerPrefix(token.LBRACE, p.parseObjectLiteral)
	p.registerPrefix(token.ILLEGAL, p.parseIllegal)

	// Read two tokens, so curToken and peekToken are both set.
	p.nextToken()
	p.nextToken()

	return p
}

// Errors returns a slice of error messages encountered during parsing.
func (p *Parser) Errors() []string {
	return p.errors
}

// Parse parses the MAML document and returns the root AST node.
func (p *Parser) Parse() *ast.Document {
	document := &ast.Document{}
	document.Statements = []ast.Statement{}

	p.skip(token.NEWLINE)

	if p.curTokenIs(token.EOF) {
		return document
	}

	stmt := p.parseStatement()
	if stmt != nil {
		document.Statements = append(document.Statements, stmt)
	}

	p.skip(token.NEWLINE)

	if !p.curTokenIs(token.EOF) {
		p.errors = append(p.errors, fmt.Sprintf("unexpected token after main value: %s ('%s')", p.curToken.Type, p.curToken.Literal))
	}

	return document
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
	for p.curTokenIs(token.COMMENT) {
		p.nextToken()
	}
}

func (p *Parser) parseStatement() ast.Statement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression()
	return stmt
}

func (p *Parser) parseExpression() ast.Expression {
	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	return prefix()
}

// The contract for all parse functions is that they are entered with p.curToken
// being the first token of the construct, and they must return with p.curToken
// pointing to the token *after* the construct.

func (p *Parser) parseIdentifier() ast.Expression {
	// If the lexer gives us an IDENT that starts with a digit or a '-',
	// it must be a malformed number, because a valid number would have
	// been tokenized as INT or FLOAT. This applies to identifiers used as values.
	lit := p.curToken.Literal
	if len(lit) > 0 {
		firstChar := lit[0]
		if (firstChar >= '0' && firstChar <= '9') || firstChar == '-' {
			p.errors = append(p.errors, fmt.Sprintf("invalid number format: %s", lit))
			p.nextToken()
			return nil
		}
	}

	expr := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken()
	return expr
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	value, err := strconv.ParseInt(p.curToken.Literal, 10, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as integer: %s", p.curToken.Literal, err))
		p.nextToken()
		return nil
	}
	lit.Value = value
	p.nextToken()
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		p.errors = append(p.errors, fmt.Sprintf("could not parse %q as float: %s", p.curToken.Literal, err))
		p.nextToken()
		return nil
	}
	lit.Value = value
	p.nextToken()
	return lit
}

func (p *Parser) parseStringLiteral() ast.Expression {
	expr := &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
	p.nextToken()
	return expr
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	expr := &ast.BooleanLiteral{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
	p.nextToken()
	return expr
}

func (p *Parser) parseNullLiteral() ast.Expression {
	expr := &ast.NullLiteral{Token: p.curToken}
	p.nextToken()
	return expr
}

func (p *Parser) parseIllegal() ast.Expression {
	p.errors = append(p.errors, fmt.Sprintf("illegal token encountered: %s", p.curToken.Literal))
	p.nextToken()
	return nil
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	p.nextToken() // Consume '['

	array.Elements = p.parseExpressionList(token.RBRACK)

	if !p.curTokenIs(token.RBRACK) {
		p.errors = append(p.errors, fmt.Sprintf("unterminated array literal, expected ']' got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // Consume ']'
	return array
}

func (p *Parser) parseExpressionList(end token.TokenType) []ast.Expression {
	list := []ast.Expression{}
	p.skip(token.NEWLINE)
	if p.curTokenIs(end) {
		return list
	}

	list = append(list, p.parseExpression())

	for {
		p.skip(token.NEWLINE, token.COMMA)
		if p.curTokenIs(end) || p.curTokenIs(token.EOF) {
			break
		}
		list = append(list, p.parseExpression())
	}
	return list
}

func (p *Parser) parseObjectLiteral() ast.Expression {
	obj := &ast.ObjectLiteral{Token: p.curToken, Pairs: []*ast.KeyValueExpression{}}
	keys := make(map[string]bool)
	p.nextToken() // Consume '{'

	p.skip(token.NEWLINE)
	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		pair := p.parseKeyValuePair()
		if pair != nil {
			var keyStr string
			switch k := pair.Key.(type) {
			case *ast.Identifier:
				keyStr = k.Value
			case *ast.StringLiteral:
				keyStr = k.Value
			}

			if keys[keyStr] {
				p.errors = append(p.errors, fmt.Sprintf("duplicate key in object: %s", keyStr))
			}
			keys[keyStr] = true
			obj.Pairs = append(obj.Pairs, pair)
		} else {
			// Error already reported. Recover to the next separator or end of object.
			for !p.curTokenIs(token.NEWLINE) && !p.curTokenIs(token.COMMA) && !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
				p.nextToken()
			}
		}

		p.skip(token.NEWLINE, token.COMMA)
	}

	if !p.curTokenIs(token.RBRACE) {
		p.errors = append(p.errors, fmt.Sprintf("unterminated object literal, expected '}' got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // Consume '}'
	return obj
}

func (p *Parser) parseKeyValuePair() *ast.KeyValueExpression {
	key := p.parseObjectKey()
	if key == nil {
		return nil
	}

	if !p.curTokenIs(token.COLON) {
		p.errors = append(p.errors, fmt.Sprintf("expected ':' after key, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // Consume ':'
	p.skip(token.NEWLINE)

	value := p.parseExpression()
	if value == nil {
		return nil
	}

	return &ast.KeyValueExpression{Key: key, Value: value}
}

func (p *Parser) parseObjectKey() ast.Expression {
	var key ast.Expression
	switch p.curToken.Type {
	case token.STRING:
		key = &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken()
	case token.IDENT, token.INT:
		// Per spec, numeric keys are treated as identifiers. No special validation needed here.
		key = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		p.nextToken()
	default:
		p.errors = append(p.errors, fmt.Sprintf("invalid token for object key: %s ('%s')", p.curToken.Type, p.curToken.Literal))
		p.nextToken()
		return nil
	}
	return key
}

func (p *Parser) skip(types ...token.TokenType) {
	for {
		if found := slices.ContainsFunc(types, func(t token.TokenType) bool {
			return p.curTokenIs(t)
		}); !found {
			break
		}
		p.nextToken()
	}
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s ('%s') found", t, p.curToken.Literal)
	p.errors = append(p.errors, msg)
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}
