package maml

import (
	"bytes"
	"fmt"

	"github.com/KimNorgaard/go-maml/lexer"
	"github.com/KimNorgaard/go-maml/mapper"
	"github.com/KimNorgaard/go-maml/parser"
)

// Marshal returns the MAML encoding of v.
func Marshal(v any, opts ...EncodeOption) ([]byte, error) {
	var buf bytes.Buffer
	e := NewEncoder(&buf, opts...)
	if err := e.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal parses the MAML-encoded data and stores the result
// in the value pointed to by v.
func Unmarshal(data []byte, v any, opts ...DecodeOption) error {
	// Note: We would create a temporary decoder and apply opts here.
	// dec := &Decoder{}
	// for _, opt := range opts { opt(dec) }

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

	return mapper.Map(doc, v)
}
