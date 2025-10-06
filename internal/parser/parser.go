package parser

import (
	"fmt"
	"slices"
	"strconv"

	"github.com/KimNorgaard/go-maml/errors"
	"github.com/KimNorgaard/go-maml/internal/ast"
	"github.com/KimNorgaard/go-maml/internal/lexer"
	"github.com/KimNorgaard/go-maml/internal/token"
)

type prefixParseFn func() ast.Expression

// Option is a function that configures a Parser.
type Option func(*Parser)

// WithParseComments enables comment parsing.
func WithParseComments() Option {
	return func(p *Parser) {
		p.parseComments = true
	}
}

// Parser holds the state of the parser.
type Parser struct {
	l      *lexer.Lexer
	errors errors.ParseErrors

	curToken  token.Token
	peekToken token.Token

	prefixParseFns map[token.Type]prefixParseFn

	parseComments bool
}

// New creates a new parser.
func New(l *lexer.Lexer, opts ...Option) *Parser {
	p := &Parser{
		l:      l,
		errors: errors.ParseErrors{},
	}

	for _, opt := range opts {
		opt(p)
	}

	p.prefixParseFns = make(map[token.Type]prefixParseFn)
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
func (p *Parser) Errors() errors.ParseErrors {
	return p.errors
}

// Parse parses the MAML document and returns the root AST node.
func (p *Parser) Parse() *ast.Document {
	document := &ast.Document{}
	document.Statements = []ast.Statement{}

	p.skip(token.NEWLINE)

	// When parsing with comments, they can appear before the main value.
	if p.parseComments {
		document.HeadComments = p.consumeComments()
		p.skip(token.NEWLINE)
	}

	if p.curTokenIs(token.EOF) {
		return document
	}

	stmt := p.parseStatement()
	if stmt != nil {
		document.Statements = append(document.Statements, stmt)
	}

	p.skip(token.NEWLINE)

	if !p.curTokenIs(token.EOF) {
		p.appendError(fmt.Sprintf("unexpected token after main value: %s ('%s')", p.curToken.Type, p.curToken.Literal))
	}

	return document
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
	if !p.parseComments {
		for p.curTokenIs(token.COMMENT) {
			p.nextToken()
		}
	}
}

// consumeComments consumes a block of comments, including the newlines between them.
func (p *Parser) consumeComments() []*ast.Comment {
	comments := []*ast.Comment{}
COMMENTS:
	for {
		switch {
		case p.curTokenIs(token.COMMENT):
			comment := &ast.Comment{Token: p.curToken, Value: p.curToken.Literal}
			comments = append(comments, comment)
			p.nextToken() // consume comment token
		case p.curTokenIs(token.NEWLINE) && p.peekTokenIs(token.COMMENT):
			// If the newline is followed by another comment, consume the newline and continue the loop.
			p.nextToken()
		default:
			break COMMENTS // Not a comment or a newline followed by a comment, so the block is done.
		}
	}
	return comments
}

// consumeNewlines consumes one or more newline tokens and returns the count.
func (p *Parser) consumeNewlines() int {
	count := 0
	for p.curTokenIs(token.NEWLINE) {
		count++
		p.nextToken()
	}
	return count
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
			p.appendError(fmt.Sprintf("invalid number format: %s", lit))
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
		p.appendError(fmt.Sprintf("could not parse %q as integer: %s", p.curToken.Literal, err))
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
		p.appendError(fmt.Sprintf("could not parse %q as float: %s", p.curToken.Literal, err))
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
	p.appendError(fmt.Sprintf("illegal token encountered: %s", p.curToken.Literal))
	p.nextToken()
	return nil
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken}
	p.nextToken() // Consume '['

	array.Elements = p.parseExpressionList(token.RBRACK)

	if !p.curTokenIs(token.RBRACK) {
		p.appendError(fmt.Sprintf("unterminated array literal, expected ']' got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // Consume ']'
	return array
}

func (p *Parser) parseExpressionList(end token.Type) []ast.Expression {
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

func (p *Parser) parseObjectLiteral() ast.Expression { //nolint:gocognit
	obj := &ast.ObjectLiteral{Token: p.curToken, Pairs: []*ast.KeyValueExpression{}}
	keys := make(map[string]bool)
	p.nextToken() // Consume '{'

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		newlines := p.consumeNewlines()
		if p.curTokenIs(token.COMMA) {
			p.nextToken()
			newlines += p.consumeNewlines()
		}

		if p.curTokenIs(token.RBRACE) {
			break
		}

		var headComments []*ast.Comment
		if p.parseComments {
			// A key-value pair can be preceded by multiple comment blocks,
			// separated by newlines. We need to consume all of them.
			for p.curTokenIs(token.COMMENT) {
				headComments = append(headComments, p.consumeComments()...)
				p.skip(token.NEWLINE)
			}
		}

		if p.curTokenIs(token.RBRACE) {
			break
		}

		pair := p.parseKeyValuePair(headComments, newlines)
		if pair != nil {
			var keyStr string
			switch k := pair.Key.(type) {
			case *ast.Identifier:
				keyStr = k.Value
			case *ast.StringLiteral:
				keyStr = k.Value
			}

			if keys[keyStr] {
				p.appendError(fmt.Sprintf("duplicate key in object: %s", keyStr))
			}
			keys[keyStr] = true
			obj.Pairs = append(obj.Pairs, pair)
			// After parsing a pair, check for foot comments that might follow.
			if p.parseComments {
				// After parsing a pair, check for foot comments that might follow,
				// which may be preceded by an optional comma.
				if p.curTokenIs(token.COMMA) && p.peekTokenIs(token.NEWLINE) {
					p.nextToken() // consume comma
				}
				// A foot comment must be on a new line.
				if p.curTokenIs(token.NEWLINE) && p.peekTokenIs(token.COMMENT) {
					p.nextToken() // consume newline
					pair.FootComments = p.consumeComments()
				}
			}
		} else {
			p.nextToken()
		}
	}

	if !p.curTokenIs(token.RBRACE) {
		p.appendError(fmt.Sprintf("unterminated object literal, expected '}' got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // Consume '}'
	return obj
}

func (p *Parser) parseKeyValuePair(headComments []*ast.Comment, newlinesBefore int) *ast.KeyValueExpression {
	key := p.parseObjectKey()
	if key == nil {
		return nil
	}

	if !p.curTokenIs(token.COLON) {
		p.appendError(fmt.Sprintf("expected ':' after key, got %s", p.curToken.Type))
		return nil
	}
	p.nextToken() // Consume ':'
	p.skip(token.NEWLINE)

	value := p.parseExpression()
	if value == nil {
		return nil
	}

	kvp := &ast.KeyValueExpression{Key: key, Value: value, HeadComments: headComments, NewlinesBefore: newlinesBefore}

	if p.parseComments {
		// A line comment must not be separated by a newline from the value.
		// It can appear before or after an optional comma.
		if p.curTokenIs(token.COMMENT) {
			// Case: `key: value # comment`
			kvp.LineComment = &ast.Comment{Token: p.curToken, Value: p.curToken.Literal}
			p.nextToken() // consume comment
		} else if p.curTokenIs(token.COMMA) && p.peekTokenIs(token.COMMENT) {
			// Case: `key: value, # comment`. Here we consume the comma as well
			// as it is part of the "line" that the comment is on.
			p.nextToken() // consume comma
			kvp.LineComment = &ast.Comment{Token: p.curToken, Value: p.curToken.Literal}
			p.nextToken() // consume comment
		}
	}

	return kvp
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
		p.appendError(fmt.Sprintf("invalid token for object key: %s ('%s')", p.curToken.Type, p.curToken.Literal))
		p.nextToken()
		return nil
	}
	return key
}

func (p *Parser) skip(types ...token.Type) {
	for {
		if found := slices.ContainsFunc(types, func(t token.Type) bool {
			return p.curTokenIs(t)
		}); !found {
			break
		}
		p.nextToken()
	}
}

func (p *Parser) registerPrefix(tokenType token.Type, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) noPrefixParseFnError(t token.Type) {
	msg := fmt.Sprintf("no prefix parse function for %s ('%s') found", t, p.curToken.Literal)
	p.appendError(msg)
}

func (p *Parser) appendError(msg string) {
	p.errors = append(p.errors, errors.ParseError{Message: msg, Line: p.curToken.Line, Column: p.curToken.Column})
}

func (p *Parser) curTokenIs(t token.Type) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.Type) bool {
	return p.peekToken.Type == t
}
