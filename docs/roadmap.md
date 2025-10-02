# Roadmap

## Phase 1: Core Parser - COMPLETE

- Define AST node types, including positional data (line, column).
- Implement the lexer to tokenize the input source code.
- Implement the parser to construct an AST from the token stream.
- **Goal**: A function that can turn source text into a complete AST.

## Phase 2: Data Mapping - COMPLETE

- Implement `Unmarshal` logic to walk the AST and populate Go objects using `reflect`.
- Implement `Marshal` logic to build an AST from Go objects.
- Implement the `Encoder` to serialize an AST back to a formatted MAML string.
- Add detailed error messages for the mapping layer (e.g., type mismatches).

## Phase 3: Advanced Features and Polish - COMPLETE

- Integrate support for the custom marshaling interfaces.
- Add more functional options for fine-grained encoder/decoder control.
- Write comprehensive documentation and examples for the public API.
