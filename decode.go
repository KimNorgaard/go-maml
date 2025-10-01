package maml

import (
	"io"
)

// DecodeOption is a functional option for configuring a Decoder.
type DecodeOption func(*Decoder) error

// Decoder reads and decodes MAML values from an input stream.
type Decoder struct {
	r io.Reader
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader, opts ...DecodeOption) *Decoder {
	d := &Decoder{r: r}
	return d
}

// Decode reads the next MAML-encoded value from its input
// and stores it in the value pointed to by v.
func (d *Decoder) Decode(v any) error {
	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}
	// Note: This is a non-streaming implementation. A true streaming decoder
	// would feed the reader directly to the lexer/parser.
	return Unmarshal(data, v)
}
