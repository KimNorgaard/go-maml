package maml

import (
	"bytes"
)

// Marshaler is the interface implemented by types that can marshal themselves
// into valid MAML.
type Marshaler interface {
	// MarshalMAML returns the MAML encoding of the value.
	MarshalMAML() ([]byte, error)
}

// Unmarshaler is the interface implemented by types that can unmarshal
// a MAML description of themselves. The input can be assumed to be a
// valid MAML value. UnmarshalMAML must copy the MAML data if it wishes
// to retain the data after returning.
type Unmarshaler interface {
	// UnmarshalMAML unmarshals the MAML-encoded data and stores the result
	// in the value pointed to by the receiver.
	UnmarshalMAML([]byte) error
}

// Marshal returns the MAML encoding of v.
//
// Marshal functions similarly to encoding/json.Marshal, traversing the value v
// recursively. If an encountered value implements the Marshaler interface,
// Marshal calls its MarshalMAML method to produce MAML.
//
// The mapping between Go values and MAML values is analogous to encoding/json:
//
// Boolean values encode as MAML booleans.
//
// Floating point, integer, and uint values encode as MAML numbers.
//
// String values encode as MAML strings.
//
// Slices and arrays encode as MAML arrays.
//
// Struct values encode as MAML objects. Exported fields are used as object keys.
// The `maml` struct tag can be used to customize key names and behavior,
// e.g., `maml:"my_key,omitempty"`.
//
// Maps encode as MAML objects. The map's key type must be a string.
//
// Pointers are dereferenced and their values are encoded. A nil pointer
// encodes as the MAML null value.
func Marshal(in any, opts ...Option) (out []byte, err error) {
	var buf bytes.Buffer
	e := NewEncoder(&buf, opts...)
	if err := e.Encode(in); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Unmarshal parses the MAML-encoded data and stores the result in the value
// pointed to by v. If v is nil or not a pointer, Unmarshal returns an error.
//
// Unmarshal uses a similar mapping from MAML to Go values as encoding/json.Unmarshal,
// and it will use the inverse of the rules described in Marshal. It supports
// `maml` struct tags for custom field mapping and honors the Unmarshaler and
// encoding.TextUnmarshaler interfaces.
//
// If the MAML data contains syntax errors, Unmarshal will return a ParseErrors
// value containing detailed information about each error.
func Unmarshal(in []byte, out any, opts ...Option) error {
	dec := NewDecoder(bytes.NewReader(in), opts...)
	return dec.Decode(out)
}
