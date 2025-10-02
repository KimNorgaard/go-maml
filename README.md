# Go MAML

[![PkgGoDev](https://pkg.go.dev/badge/github.com/KimNorgaard/go-maml)](https://pkg.go.dev/github.com/KimNorgaard/go-maml)
![CI](https://github.com/KimNorgaard/go-maml/workflows/CI/badge.svg)
[![codecov](https://codecov.io/gh/KimNorgaard/go-maml/branch/main/graph/badge.svg)](https://codecov.io/gh/KimNorgaard/go-maml)
[![Go Report Card](https://goreportcard.com/badge/github.com/KimNorgaard/go-maml)](https://goreportcard.com/report/github.com/KimNorgaard/go-maml)

Go-MAML is a Go library for parsing the [MAML (Minimal Abstract Markup
Language)](https://maml.dev) configuration language.

MAML is a human-readable configuration language that keeps JSON's simplicity and
adds features like comments, multiline strings, optional commas, and optional
key quotes.

## Example

Here is an example of MAML:

```maml
{
  project: "MAML"
  tags: [
    "minimal"
    "readable"
  ]

  # A simple nested object
  spec: {
    version: 1
    author: "Anton Medvedev"
  }
}
```

## Usage

The library's API is designed to be idiomatic Go, similar to the standard
`encoding/json` package.

```go
package main

import (
	"fmt"
	"log"

	"github.com/KimNorgaard/go-maml"
)

var data = []byte(`
{
  project: "MAML"
  tags: [ "minimal", "readable" ]
  spec: {
    version: 1
    author: "Anton Medvedev"
  }
}
`)

type Config struct {
	Project string   `maml:"project"`
	Tags    []string `maml:"tags"`
	Spec    struct {
		Version int    `maml:"version"`
		Author  string `maml:"author"`
	} `maml:"spec"`
}

func main() {
	var cfg Config
	if err := maml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("error: %v", err)
	}

	fmt.Printf("Project: %s\n", cfg.Project)
	fmt.Printf("Tags: %v\n", cfg.Tags)
	fmt.Printf("Spec Version: %d\n", cfg.Spec.Version)
}
```
### Embedded Struct Support

`go-maml` fully supports unmarshaling into anonymous embedded structs, following
the same precedence rules as Go's standard `encoding/json` package. This
includes support for both value and pointer embedded structs.

Given the following Go types:

```go
type Address struct {
    City   string `maml:"city"`
    Street string
}

type User struct {
    Name string
    *Address // Embedded pointer to struct
}
```

And the MAML input:

```maml
{
  Name: "Jane Doe"
  city: "New York"
  Street: "123 Main St"
}
```

The `Unmarshal` call will behave as follows:

```go
var mamlInput = []byte(`
{
  Name: "Jane Doe"
  city: "New York"
  Street: "123 Main St"
}
`)

var user User
err := maml.Unmarshal(mamlInput, &user)

// Result:
// user.Name == "Jane Doe"
// user.Address != nil
// user.Address.City == "New York"  (matched via tag)
// user.Address.Street == "123 Main St" (matched via case-insensitive name)
```

## Features

*   Familiar `Marshal`/`Unmarshal`/`NewEncoder`/`NewDecoder` interface.
*   Full support for `maml.Marshaler` and `maml.Unmarshaler` interfaces.
*   Struct tags for custom field mapping (`maml:"key,omitempty"`).
*   Support for anonymous embedded structs, following `encoding/json` precedence rules.
*   Provides structured parse errors with line and column numbers.
*   Configurable encoding options, such as indentation.

## Roadmap

This library implements v0.1 of the MAML specification. With respect to this
version of the spec, the library's feature set is considered complete. As the
MAML specification itself is currently in an unstable version (0.1), this
library should also be considered unstable.

## Contributing

Contributions are welcome!

## Acknowledgements

The design of the public API is inspired by the excellent
[goccy/go-yaml](https://github.com/goccy/go-yaml) library.

## License

This project is licensed under the MIT License.
