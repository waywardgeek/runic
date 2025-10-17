# Porting Guide: Implementing Runic in Your Language

This guide explains how to implement Runic in a new programming language while maintaining consistency across implementations.

## Architecture Overview

Runic has three phases that must be implemented:

### Phase 1: Data Structures (Foundation)

Core types that form the basis of the parser:

1. **Peg** - Main container holding rules and parser state
2. **Rule** - Represents a grammar rule with its expression
3. **Pexpr** - Parsing expression (11 types: nonterm, term, keyword, sequence, choice, etc.)
4. **Token** - Lexical token from input
5. **Lexer** - Tokenizes input text
6. **ParseResult** - Memoization entry for packrat parsing
7. **Node** - AST node after parsing
8. **Keytab** - Keyword/symbol table
9. **Location** - Source position tracking
10. **Filepath** - File representation with content

**Relations to implement:**
- Rule ↔ Pexpr (one-to-one)
- Pexpr ↔ Pexpr (parent-child tree)
- Rule ↔ Rule (nonterminal references)
- ParseResult ↔ ParseResult (parent-child, memoization hash)
- Node ↔ Node (AST tree)
- Lexer ↔ Token (one-to-many)
- Lexer ↔ ParseResult (one-to-many)

### Phase 2: Grammar Parser

Parses `.syn` files to build the grammar:

**Functions to implement:**
- `parseRules()` - Main entry point
- `parseRule()` - Parse one rule
- `parsePexpr()` - Parse expression tree
- `parseChoicePexpr()` - Handle `|` operator
- `parseSequencePexpr()` - Handle sequences
- `parsePrefixPexpr()` - Handle `&` and `!` operators
- `parsePostfixPexpr()` - Handle `?`, `*`, `+` operators
- `parseBasicPexpr()` - Handle terminals and nonterminals
- `parseParenPexpr()` - Handle `(` ... `)`

**Critical details:**
- 2-token lookahead buffer (`savedToken1`, `savedToken2`)
- Skip newlines in grammar (they're just whitespace)
- `endOfRule()` checks for `:` or `:=` at lookahead(2)
- `endOfSequence()` checks for `|`, `)`, or end-of-rule

### Phase 3: PEG Engine

Uses the grammar to parse input files:

**Functions to implement:**
- `parse()` - Main entry point
- `tokenizeInput()` - Read all tokens upfront
- `addEOFToFirstRule()` - Ensure first rule ends with EOF
- `parseUsingRule()` - Parse with memoization and left-recursion
- `parseUsingPexpr()` - Dispatch to specific pexpr handler
- `parseUsingSequencePexpr()` - Match sequence
- `parseUsingChoicePexpr()` - Try alternatives
- `parseUsingZeroOrMorePexpr()` - Match `*`
- `parseUsingOneOrMorePexpr()` - Match `+`
- `parseUsingOptionalPexpr()` - Match `?`
- `parseUsingAndPexpr()` - Lookahead without consuming
- `parseUsingNotPexpr()` - Negative lookahead
- `buildParseTree()` - Construct AST from ParseResults

**Critical algorithms:**

1. **Left-Recursion (Warth et al. 2008)**:
```
function parseUsingRule(rule, pos):
    # Check memo
    if memoized(rule, pos):
        return cached_result
    
    # Seed with failure
    result = seed_failure
    
    # Grow until no progress
    loop:
        result = try_parse(rule, pos)
        if not improved:
            break
        cache(rule, pos, result)
    
    return result
```

2. **Memoization**:
   - Cache ParseResults by (rule, position)
   - Track recursion with `pending` and `foundRecursion` flags
   - Clear cache between parse calls

3. **AST Simplification**:
   - Remove nodes if (rule is null/weak) AND (token is null/weak)
   - Merge single children unless both parent and child are strong
   - Preserve string literals and strong tokens

## Implementation Steps

### Step 1: Data Structures (Week 1-2)

1. Implement all core types
2. Implement relations (linked lists, hashes)
3. Write tests for each type
4. Verify memory management

**Tests to pass:**
- Create and destroy objects
- Build relation graphs
- Hash lookup performance

### Step 2: Lexer (Week 2-3)

1. Implement character reading (UTF-8 support)
2. Implement token types
3. Implement keyword table
4. Handle escape sequences
5. Handle comments

**Tests to pass:**
- Tokenize integers, floats, strings, identifiers
- Handle escape sequences
- Skip comments
- Track line/column positions

### Step 3: Grammar Parser (Week 3-4)

1. Implement 2-token lookahead
2. Implement `parseRule()` and helpers
3. Handle all Pexpr types
4. Build keyword table
5. Bind nonterminal references

**Tests to pass:**
- Parse simple rules
- Parse sequences and choices
- Parse operators (?, *, +, &, !)
- Parse full rune.syn (157 rules)
- Round-trip: parse → print → parse

### Step 4: PEG Engine (Week 4-6)

1. Implement tokenization
2. Implement basic parsing (no left-recursion)
3. Add memoization
4. Add left-recursion support
5. Implement AST building
6. Implement AST simplification

**Tests to pass:**
- Parse simple expressions
- Handle operator precedence
- Parse with left-recursion
- Build correct AST
- Simplify AST correctly

## Testing Strategy

### Unit Tests

Each phase should have unit tests:
- Data structure creation/manipulation
- Lexer token generation
- Grammar parsing
- PEG engine parsing

### Integration Tests

Test complete workflows:
- Parse grammar → Parse input → Build AST
- Parse complex grammars (json.syn, calculator.syn, rune.syn)

### Conformance Tests

All implementations must pass common tests in `tests/conformance/`:
- `test_basic_parsing.txt` - Simple sequences and choices
- `test_left_recursion.txt` - Left-recursive operator precedence
- `test_ast_simplification.txt` - Weak rule removal
- `test_edge_cases.txt` - Error handling, empty input, etc.

## Common Pitfalls

### 1. Lookahead Buffer

**Problem**: Token buffer not maintained correctly after peeking.

**Solution**: 
- `peekToken(1)` fills `savedToken1` if empty
- `peekToken(2)` fills `savedToken2` if empty
- `parseToken()` returns `savedToken1`, shifts `savedToken2` → `savedToken1`
- Never check `lexer.EOF()` when buffer might have tokens

### 2. endOfRule() Logic

**Problem**: Confusing physical EOF with logical EOF.

**Solution**: `endOfRule()` should check if `peekToken(2)` is `:`, `:=`, or EOF token type (not lexer position).

### 3. AST Simplification AND Logic

**Problem**: Removing strong tokens (like string literals).

**Solution**: Remove only if (rule is null/weak) **AND** (token is null/weak). Use AND, not OR.

### 4. Left-Recursion Memoization

**Problem**: Infinite recursion or incorrect results.

**Solution**: 
- Seed with failure, grow result
- Track `pending` to detect recursion
- Clear cache between parse calls

### 5. Token Duplication

**Problem**: Tokens appearing multiple times in lexer.

**Solution**: Only append token to lexer.Tokens when creating with NewToken(), not when reading from buffer.

## Performance Considerations

1. **Memoization**: O(n) parsing with proper caching
2. **Tokenization**: Read all tokens upfront (simpler than streaming)
3. **AST Building**: One pass, no reparsing
4. **Memory**: Clear caches between parses

## Language-Specific Notes

### Go
- Use pointers for relations
- Use `nil` for null references
- Implement error handling with `error` return type

### Python
- Use `None` for null references
- Consider using dataclasses or attrs
- Weak references for circular structures

### Rust
- Use `Option<Box<T>>` for nullable references
- Use `Rc<RefCell<T>>` for shared mutable state
- Implement Drop for cleanup

### C
- Manual memory management required
- Use linked lists for relations
- Hash tables for memoization

### JavaScript/TypeScript
- Use `null` for references
- TypeScript provides type safety
- Consider using classes with private fields

## Reference Implementation

The Go implementation in `implementations/go/` is the reference. When in doubt:
1. Check the Go code
2. Compare with Rune bootstrap parser in `bootstrap/parse/pegparser.rn`
3. Run conformance tests

## Getting Help

- Check existing implementations for examples
- Read the architecture docs
- Compare test results
- File issues on GitHub

## Submission Checklist

Before submitting a new language implementation:

- [ ] All unit tests pass
- [ ] All conformance tests pass
- [ ] Parses rune.syn correctly
- [ ] Handles helloworld.rn correctly
- [ ] AST simplification works
- [ ] Left-recursion supported
- [ ] Documentation complete
- [ ] README.md in implementation directory
- [ ] Examples directory with sample usage

## Estimated Effort

- **Simple implementation**: 2-3 weeks full-time
- **Production-ready**: 4-6 weeks with full testing
- **With optimizations**: 8-10 weeks

Start with the Go implementation as a reference and translate carefully, testing each phase before moving to the next.
