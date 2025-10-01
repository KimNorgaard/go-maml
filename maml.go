package maml

import (
	"bytes"
	"fmt"

	"github.com/KimNorgaard/go-maml/internal/lexer"
	"github.com/KimNorgaard/go-maml/internal/mapper"
	"github.com/KimNorgaard/go-maml/internal/parser"
)

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
		maxDepth: mapper.DefaultMaxDepth,
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

	return mapper.Map(doc, v, o.maxDepth)
}
