# Runic Go Implementation

Go implementation of the Runic PEG parser with left-recursion support.

## Features

- ✅ Full PEG grammar support
- ✅ Left-recursion handling
- ✅ Packrat parsing (memoization)
- ✅ AST simplification
- ✅ UTF-8 support
- ✅ 44 unit tests passing

## Installation

```bash
go get github.com/yourusername/runic/implementations/go
```

## Usage

### Parsing with a Grammar

```go
package main

import (
    "fmt"
    "log"
    parser "github.com/yourusername/runic/implementations/go"
)

func main() {
    // Load and parse grammar
    peg := &parser.Peg{
        PegKeytab:   parser.NewKeytab(),
        Keytab:      parser.NewKeytab(),
        // ... initialization
    }
    peg.BuildPegKeywordTable()
    
    // Load grammar file
    grammarFile := parser.NewFilepath("mygrammar.syn", nil, true)
    lexer, _ := parser.NewLexer(grammarFile, peg.PegKeytab, false)
    peg.InsertLexer(lexer)
    
    if err := peg.ParseRules(); err != nil {
        log.Fatal(err)
    }
    
    // Parse input file
    node, err := peg.Parse("input.txt", false)
    if err != nil {
        log.Fatal(err)
    }
    
    // Print AST
    fmt.Println(node.ToString())
}
```

### Example: Calculator

```go
// Load calculator grammar
peg, _ := parser.NewPegFromFile("calculator.syn")

// Parse expression
node, _ := peg.Parse("2 + 3 * 4", false)

// Simplified AST shows operator precedence
node.Simplify()
fmt.Println(node.ToString())
// Output: addExpr(2 mulExpr(3 4))
```

## Testing

```bash
# Run all tests
go test ./...

# Run specific test
go test -run TestParseRuneSyn

# Verbose output
go test -v

# With coverage
go test -cover
```

## Performance

- **Tokenization**: O(n) - single pass through input
- **Parsing**: O(n) with memoization - packrat parsing
- **Memory**: O(n) for tokens and parse results

Typical performance on modern hardware:
- **Small grammars** (<20 rules): <1ms
- **Medium grammars** (~50 rules): ~5ms  
- **Large grammars** (rune.syn, 157 rules): ~10ms
- **Large inputs** (>10KB): ~50-100ms

## Architecture

See [../../docs/architecture.md](../../docs/architecture.md) for overall design.

### Phase 1: Data Structures
- `peg.go` - Main Peg container
- `rule.go` - Grammar rules
- `pexpr.go` - Parsing expressions
- `token.go` - Lexical tokens
- `lexer.go` - Tokenizer
- `parseresult.go` - Memoization entries
- `node.go` - AST nodes
- `keytab.go` - Symbol table
- `location.go` - Source positions

### Phase 2: Grammar Parser
- `parser2.go` - Parses .syn grammar files
- Builds Peg, Rule, and Pexpr structures
- 2-token lookahead
- Handles all PEG operators

### Phase 3: PEG Engine
- `parser3.go` - Uses grammar to parse input
- Left-recursion support (Warth et al. 2008)
- Packrat parsing with memoization
- AST building and simplification

## API Reference

### Core Types

```go
type Peg struct {
    // Main parser container
}

type Rule struct {
    Sym  *Sym
    Pexpr *Pexpr
    Weak bool
}

type Pexpr struct {
    Type PexprType  // Sequence, Choice, etc.
    // ... children
}

type Node struct {
    ParseResult *ParseResult
    Token       *Token
    // ... children
}
```

### Key Functions

```go
// Parse grammar file
func (p *Peg) ParseRules() error

// Parse input using grammar
func (p *Peg) Parse(fileSpec interface{}, allowUnderscores bool) (*Node, error)

// Simplify AST
func (n *Node) Simplify()

// Convert AST to string
func (n *Node) ToString() string
```

## Examples

See [../../examples/](../../examples/) for:
- `grammars/json.syn` - JSON parser
- `grammars/calculator.syn` - Expression evaluator
- `grammars/rune.syn` - Full Rune language

## Known Limitations

- Only direct left-recursion (not indirect through multiple rules)
- No hidden left-recursion through nullable rules
- Error messages could be more detailed

## Contributing

See [../../CONTRIBUTING.md](../../CONTRIBUTING.md) for guidelines.

## License

Apache License 2.0 - See [../../LICENSE](../../LICENSE)
