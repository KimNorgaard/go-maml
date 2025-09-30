package ast

import (
	"testing"

	"github.com/KimNorgaard/go-maml/token"
	"github.com/stretchr/testify/require"
)

func TestString(t *testing.T) {
	document := &Document{
		Statements: []Statement{
			&ExpressionStatement{
				Token: token.Token{Type: token.LBRACE, Literal: "{"},
				Expression: &ObjectLiteral{
					Token: token.Token{Type: token.LBRACE, Literal: "{"},
					Pairs: []*PairExpression{
						{
							Token: token.Token{Type: token.COLON, Literal: ":"},
							Key: &Identifier{
								Token: token.Token{Type: token.IDENT, Literal: "my-key"},
								Value: "my-key",
							},
							Value: &StringLiteral{
								Token: token.Token{Type: token.STRING, Literal: "my-value"},
								Value: "my-value",
							},
						},
					},
				},
			},
		},
	}

	expected := `{my-key:"my-value"}`
	require.Equal(t, expected, document.String())
}
