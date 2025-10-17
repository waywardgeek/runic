# Runic Repository Structure

Created: October 16, 2024

This document describes the organization of the Runic multi-language PEG parser repository.

## Directory Structure

```
runic/
├── README.md                     # Main project overview
├── LICENSE                       # Apache 2.0 license
├── CONTRIBUTING.md              # Contribution guidelines
├── .gitignore                   # Git ignore patterns
│
├── docs/                        # Documentation
│   ├── specification.md         # Grammar syntax and semantics
│   ├── porting-guide.md         # How to implement in new languages
│   ├── architecture.md          # [TODO] System architecture
│   └── left-recursion.md        # [TODO] Algorithm explanation
│
├── examples/                    # Example grammars and inputs
│   ├── grammars/
│   │   ├── json.syn            # JSON parser
│   │   ├── calculator.syn      # Arithmetic with left-recursion
│   │   └── rune.syn            # Full Rune language (157 rules)
│   └── inputs/
│       └── helloworld.rn       # Sample Rune program
│
├── tests/                       # Common test files
│   └── conformance/             # [TODO] Cross-implementation tests
│       ├── basic_parsing.txt
│       ├── left_recursion.txt
│       ├── ast_simplification.txt
│       └── edge_cases.txt
│
└── implementations/             # Language implementations
    ├── go/                      # ✅ Complete (reference implementation)
    │   ├── README.md
    │   ├── *.go                 # Source files
    │   ├── *_test.go            # Test files
    │   └── cmd/                 # Command-line tools
    │
    ├── python/                  # 🚧 Planned
    │   └── README.md
    │
    ├── rust/                    # 🚧 Planned
    │
    ├── c/                       # 🚧 Planned
    │
    └── js/                      # 🚧 Planned
```

## Implementation Status

### Go (Complete) ✅
- **Lines of code**: ~5,000
- **Tests**: 44 passing
- **Features**: Full PEG, left-recursion, AST simplification
- **Location**: `implementations/go/`
- **Status**: Reference implementation, all features complete

### Python (Planned) 🚧
- **Status**: Not yet started
- **Effort**: 2-3 weeks estimated
- **Priority**: High (popular language)

### Rust (Planned) 🚧
- **Status**: Not yet started
- **Effort**: 2-3 weeks estimated
- **Priority**: Medium (systems programming)

### C (Planned) 🚧
- **Status**: Not yet started
- **Effort**: 3-4 weeks estimated (manual memory management)
- **Priority**: Medium (bootstrap/embedded)

### JavaScript/TypeScript (Planned) 🚧
- **Status**: Not yet started
- **Effort**: 2-3 weeks estimated
- **Priority**: High (web development)

## Key Design Principles

1. **Language Parity**: All implementations must have identical semantics
2. **No Special Status**: Go is just the first, not privileged
3. **Conformance Tests**: All implementations must pass common tests
4. **Independent Building**: Each implementation builds independently
5. **Consistent Structure**: Follow the three-phase architecture

## Three-Phase Architecture

All implementations follow the same structure:

### Phase 1: Data Structures
Core types that form the foundation:
- Peg (main container)
- Rule (grammar rules)
- Pexpr (parsing expressions)
- Token (lexical tokens)
- Lexer (tokenizer)
- ParseResult (memoization)
- Node (AST)

### Phase 2: Grammar Parser
Parses `.syn` files to build grammar structures:
- 2-token lookahead
- Handles all PEG operators
- Builds rule tree

### Phase 3: PEG Engine
Uses grammar to parse input:
- Left-recursion support (Warth et al. 2008)
- Packrat parsing (memoization)
- AST building and simplification

## Documentation Status

- ✅ Main README
- ✅ Contributing guide
- ✅ Grammar specification
- ✅ Porting guide
- ✅ Go implementation README
- 🚧 Architecture overview (TODO)
- 🚧 Left-recursion algorithm (TODO)
- 🚧 API reference (TODO)

## Example Grammars

Included examples demonstrate different features:

1. **json.syn**: Simple format parser
2. **calculator.syn**: Operator precedence with left-recursion
3. **rune.syn**: Full production language (157 rules)

## Testing Strategy

### Per-Implementation Tests
- Unit tests for each component
- Integration tests for complete workflows
- Language-specific test frameworks

### Conformance Tests (TODO)
Common test files in `tests/conformance/` that all implementations must pass:
- Basic parsing (sequences, choices)
- Left-recursion (operator precedence)
- AST simplification (weak rules)
- Edge cases (errors, EOF, empty)

## Getting Started

### For Users
```bash
cd implementations/go
go get .
go test ./...
```

### For Contributors
See [CONTRIBUTING.md](CONTRIBUTING.md) for:
- How to implement in a new language
- How to add example grammars
- How to improve documentation
- Code review process

## Next Steps

### Immediate (Week 1-2)
- [ ] Create architecture.md documentation
- [ ] Create left-recursion.md documentation
- [ ] Add conformance test files
- [ ] Add CI/CD setup (GitHub Actions)

### Short-term (Month 1-2)
- [ ] Python implementation
- [ ] More example grammars
- [ ] Performance benchmarks
- [ ] API documentation

### Long-term (Month 3-6)
- [ ] Rust implementation
- [ ] JavaScript/TypeScript implementation
- [ ] C implementation
- [ ] Website with interactive demo
- [ ] Language grammar repository

## Origin and Credits

Runic is derived from the PEG parser in the [Rune programming language](https://github.com/google/rune) by Bill Cox at Google. The Rune bootstrap parser (written in Rune) demonstrated the viability of this approach.

**Key innovations from Rune:**
- Left-recursion support in PEG
- Weak rules for AST simplification
- Clean grammar syntax

**Runic contributions:**
- Multi-language implementations
- Comprehensive documentation
- Example library
- Conformance testing

## License

Apache License 2.0 - See [LICENSE](LICENSE) file.

Original Rune code: Copyright 2021 Google LLC
Runic adaptations: Copyright 2024 Contributors
