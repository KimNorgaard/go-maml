package maml

import (
	"fmt"
	"strconv"

	"github.com/KimNorgaard/go-maml/ast"
	"github.com/KimNorgaard/go-maml/token"
)

// Parser transforms a stream of tokens into an AST.
type Parser struct {
	l      *Lexer
	errors []string

	curToken  token.Token
	peekToken token.Token
}

// NewParser creates a new Parser.
func NewParser(l *Lexer) *Parser {
	p := &Parser{l: l}
	// Read two tokens, so curToken and peekToken are both set
	p.nextToken()
	p.nextToken()
	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

func (p *Parser) Errors() []string {
	return p.errors
}

// ParseDocument parses the MAML document and returns the root AST node.
func (p *Parser) ParseDocument() *ast.Document {
	document := &ast.Document{}
	document.Statements = []ast.Statement{}

	for !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			document.Statements = append(document.Statements, stmt)
		}
	}
	return document
}

func (p *Parser) parseStatement() ast.Statement {
	// Skip trivia like comments and newlines that are not syntactically significant
	p.skipTrivia()

	if p.curTokenIs(token.EOF) {
		return nil
	}

	stmt := p.parseExpressionStatement()
	p.nextToken() // Always advance after a statement
	return stmt
}

func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	stmt := &ast.ExpressionStatement{Token: p.curToken}
	stmt.Expression = p.parseExpression()
	return stmt
}

func (p *Parser) parseExpression() ast.Expression {
	switch p.curToken.Type {
	case token.IDENT:
		return p.parseIdentifier()
	case token.INT:
		return p.parseIntegerLiteral()
	case token.FLOAT:
		return p.parseFloatLiteral()
	case token.STRING:
		return p.parseStringLiteral()
	case token.TRUE, token.FALSE:
		return p.parseBooleanLiteral()
	case token.NULL:
		return p.parseNullLiteral()
	case token.LBRACK:
		return p.parseArrayLiteral()
	case token.LBRACE:
		return p.parseObjectLiteral()
	default:
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseIntegerLiteral() ast.Expression {
	lit := &ast.IntegerLiteral{Token: p.curToken}
	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseFloatLiteral() ast.Expression {
	lit := &ast.FloatLiteral{Token: p.curToken}
	value, err := strconv.ParseFloat(p.curToken.Literal, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as float", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}
	lit.Value = value
	return lit
}

func (p *Parser) parseBooleanLiteral() ast.Expression {
	return &ast.BooleanLiteral{Token: p.curToken, Value: p.curToken.Type == token.TRUE}
}

func (p *Parser) parseStringLiteral() ast.Expression {
	return &ast.StringLiteral{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) parseNullLiteral() ast.Expression {
	return &ast.NullLiteral{Token: p.curToken}
}

func (p *Parser) parseArrayLiteral() ast.Expression {
	array := &ast.ArrayLiteral{Token: p.curToken} // current token is '['
	array.Elements = []ast.Expression{}

	for {
		p.nextToken()
		p.skipTrivia()
		if p.curTokenIs(token.RBRACK) || p.curTokenIs(token.EOF) {
			break
		}
		array.Elements = append(array.Elements, p.parseExpression())
	}
	return array
}

func (p *Parser) parseObjectLiteral() ast.Expression {
	obj := &ast.ObjectLiteral{Token: p.curToken} // current token is '{'
	obj.Pairs = []*ast.PairExpression{}

	for {
		p.nextToken()
		p.skipTrivia()
		if p.curTokenIs(token.RBRACE) || p.curTokenIs(token.EOF) {
			break
		}

		key := p.parseExpression()
		p.nextToken()

		if !p.curTokenIs(token.COLON) {
			p.peekError(token.COLON)
			return nil
		}
		p.nextToken() // consume colon
		p.skipTrivia()

		value := p.parseExpression()

		obj.Pairs = append(obj.Pairs, &ast.PairExpression{Key: key, Value: value})
	}
	return obj
}

func (p *Parser) skipTrivia() {
	for p.curToken.Type == token.NEWLINE || p.curToken.Type == token.COMMA || p.curToken.Type == token.COMMENT {
		p.nextToken()
	}
}

func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	}
	p.peekError(t)
	return false
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead", t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}
