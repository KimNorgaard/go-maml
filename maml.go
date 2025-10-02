package maml

import (
	"bytes"
	"fmt"

	"github.com/KimNorgaard/go-maml/internal/lexer"
	"github.com/KimNorgaard/go-maml/internal/parser"
)

// Marshaler is the interface implemented by types that
// can marshal themselves into valid MAML.
type Marshaler interface {
	MarshalMAML() ([]byte, error)
}

// Marshal returns the MAML encoding of v.
func Marshal(v any, opts ...Option) ([]byte, error) {
	var buf bytes.Buffer
	e := NewEncoder(&buf, opts...)
	if err := e.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal parses the MAML-encoded data and stores the result
// in the value pointed to by v.
func Unmarshal(data []byte, v any, opts ...Option) error {
	o := options{
		maxDepth: defaultMaxDepth,
	}

	for _, opt := range opts {
		if err := opt(&o); err != nil {
			return err
		}
	}

	l := lexer.New(data)
	p := parser.New(l)
	doc := p.Parse()

	if len(p.Errors()) > 0 {
		var errStr string
		for i, msg := range p.Errors() {
			if i > 0 {
				errStr += "\n"
			}
			errStr += msg
		}
		return fmt.Errorf("maml: parsing error: %s", errStr)
	}

	return mapDocument(doc, v, &o)
}
