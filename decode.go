package maml

import (
	"io"
)

// Decoder reads and decodes MAML values from an input stream.
type Decoder struct {
	r    io.Reader
	opts []Option
}

// NewDecoder returns a new decoder that reads from r. It stores options
// to be applied later by the Decode method.
func NewDecoder(r io.Reader, opts ...Option) *Decoder {
	return &Decoder{r: r, opts: opts}
}

// Decode reads the next MAML-encoded value from its input
// and stores it in the value pointed to by v.
// Note: This is a non-streaming implementation. It reads the entire
// reader into memory first before parsing.
func (d *Decoder) Decode(v any) error {
	if d.r == nil {
		return nil
	}
	data, err := io.ReadAll(d.r)
	if err != nil {
		return err
	}

	return Unmarshal(data, v, d.opts...)
}
