package maml

import (
	"bytes"
)

// Marshaler is the interface implemented by types that
// can marshal themselves into valid MAML.
type Marshaler interface {
	MarshalMAML() ([]byte, error)
}

// Marshal returns the MAML encoding of in.
func Marshal(in any, opts ...Option) (out []byte, err error) {
	var buf bytes.Buffer
	e := NewEncoder(&buf, opts...)
	if err := e.Encode(in); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal parses the MAML-encoded data and stores the result
// in the value pointed to by out.
func Unmarshal(in []byte, out any, opts ...Option) error {
	dec := NewDecoder(bytes.NewReader(in), opts...)
	return dec.Decode(out)
}
