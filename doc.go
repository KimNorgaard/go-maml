/*
Package maml provides a robust and idiomatic Go interface for parsing and encoding
the MAML (Minimal Abstract Markup Language). The library's API is designed to be
familiar to Go developers, closely mirroring the standard `encoding/json` package.

The package offers two primary workflows depending on the use case:

1. Data-Oriented Decoding and Encoding

For the common task of converting MAML data into Go structs (and vice versa),
the Marshal and Unmarshal functions provide a simple and direct API. This path
is optimized for data extraction and does not preserve comments or formatting.

Example of unmarshaling into a struct:

	var data = []byte(`{ name: "MAML", version: 1.0 }`)

	type Config struct {
		Name    string  `maml:"name"`
		Version float64 `maml:"version"`
	}

	var cfg Config
	if err := maml.Unmarshal(data, &cfg); err != nil {
		// handle error
	}
	// cfg is now populated with {Name: "MAML", Version: 1.0}

2. Full-Fidelity Document Manipulation

For advanced use cases like building linters, formatters, or configuration
editors, the library provides a way to work with a full-fidelity Abstract
Syntax Tree (AST). The Parse function is the entry point for this workflow,
preserving all comments, spacing, and structural information from the source.

The returned AST can be programmatically modified and then marshaled back
to a string, keeping the original comments intact.

Example of a comment-preserving round-trip:

	// Source MAML with a comment
	var input = []byte("# The project name\n{ name: \"MAML\" }")

	// Parse into a full-fidelity AST
	doc, err := maml.Parse(input)
	if err != nil {
		// handle error
	}

	// The 'doc' can now be inspected or modified.

	// Marshal the AST back to bytes, preserving the comment.
	// Use functional options like Indent for formatting.
	output, err := maml.Marshal(doc, maml.Indent(2))
	if err != nil {
		// handle error
	}
	// output will contain the original comment and structure.

Customization is available via struct field tags (e.g., `maml:"key,omitempty"`)
and by implementing the maml.Marshaler and maml.Unmarshaler interfaces.
*/
package maml
