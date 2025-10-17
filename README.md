# Runic: A Multi-Language PEG Parser with Left-Recursion Support

Runic is a Parsing Expression Grammar (PEG) parser that supports left-recursion and packrat parsing (memoization). It's designed to be ported across multiple programming languages while maintaining consistent behavior.

## Features

- ✅ **Left-Recursion Support**: Handles direct left-recursion using the seed algorithm from [Warth et al. (2008)](http://www.tinlizzie.org/~awarth/papers/pepm08.pdf)
- ✅ **Packrat Parsing**: Memoization for O(n) parsing performance
- ✅ **AST Simplification**: Configurable node tree simplification with weak rules
- ✅ **Clean Syntax**: Simple `.syn` grammar file format
- ✅ **Multi-Language**: Same semantics across all language implementations

## Architecture

Runic consists of three phases:

1. **Phase 1: Data Structures** - Core types (Peg, Rule, Pexpr, Node, Token, etc.)
2. **Phase 2: Grammar Parser** - Parses `.syn` grammar files into rule structures
3. **Phase 3: PEG Engine** - Uses grammar to parse input files with memoization

## Language Implementations

| Language | Status | Directory | Notes |
|----------|--------|-----------|-------|
| Go | ✅ Complete | `implementations/go/` | Reference implementation (44 tests) |
| Python | 🚧 Planned | `implementations/python/` | Coming soon |
| Rust | 🚧 Planned | `implementations/rust/` | Coming soon |
| C | 🚧 Planned | `implementations/c/` | Coming soon |
| JavaScript/TypeScript | 🚧 Planned | `implementations/js/` | Coming soon |

Each implementation:
- Has its own test suite
- Must pass common conformance tests in `tests/conformance/`
- Follows the same parsing semantics
- Maintains feature parity

## Quick Start

### Go Implementation

```bash
cd implementations/go
go test ./...
```

### Parsing a File

```go
package main

import (
    "fmt"
    "github.com/yourusername/runic/implementations/go/parser"
)

func main() {
    // Load grammar
    peg, err := parser.NewPeg("mygrammar.syn")
    if err != nil {
        panic(err)
    }
    
    // Parse input
    node, err := peg.Parse("input.txt", false)
    if err != nil {
        panic(err)
    }
    
    // Print AST
    fmt.Println(node.ToString())
}
```

## Grammar Syntax

Runic uses a simple `.syn` file format:

```
# Comments start with #
ruleName := expression
weakRule : expression    # : makes a weak rule (removed during simplification)

# Operators
sequence    := "hello" "world"
choice      := "a" | "b" | "c"
optional    := expr?
zeroOrMore  := expr*
oneOrMore   := expr+
and         := &expr     # Lookahead (doesn't consume)
not         := !expr     # Negative lookahead

# Terminals
keyword     := "if"      # Literal string
token       := INTEGER   # Built-in token types
empty       := EMPTY     # Matches nothing (epsilon)
```

## Example Grammars

See `examples/grammars/` for:
- `json.syn` - JSON parser
- `calculator.syn` - Arithmetic expression calculator
- `rune.syn` - Rune language grammar (157 rules)

## Documentation

- [Grammar Specification](docs/specification.md)
- [Architecture Overview](docs/architecture.md)
- [Left-Recursion Algorithm](docs/left-recursion.md)
- [Porting Guide](docs/porting-guide.md)

## Origin

Runic is based on the PEG parser from the [Rune programming language](https://github.com/google/rune) by Bill Cox at Google. The Rune bootstrap parser (written in Rune) demonstrated the viability of this approach, and Runic brings it to multiple languages.

## License

Apache License 2.0 (see LICENSE file)

## Contributing

Contributions are welcome! Especially:
- New language implementations
- Additional example grammars
- Documentation improvements
- Test coverage

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## Credits

- **Original Design**: Bill Cox (Rune PEG parser)
- **Left-Recursion Algorithm**: Alessandro Warth, James R. Douglass, Todd Millstein
- **Go Implementation**: CodeRhapsody AI
