# Runic Grammar Specification

## Overview

Runic uses a `.syn` file format to define Parsing Expression Grammars (PEGs). The syntax is designed to be simple and readable while supporting advanced features like left-recursion and weak rules.

## Basic Syntax

### Rules

```
ruleName := expression
```

Rules define non-terminals in the grammar. The left-hand side is the rule name, and the right-hand side is a parsing expression.

### Weak Rules

```
weakRule : expression
```

Using `:` instead of `:=` creates a weak rule. Weak rules are removed during AST simplification, making the parse tree cleaner.

### Comments

```
# This is a comment
```

Comments start with `#` and continue to the end of the line.

## Parsing Expressions

### Sequence

```
sequence := "hello" "world" name
```

Matches each expression in order. All must succeed for the sequence to succeed.

### Choice (Ordered)

```
choice := "a" | "b" | "c"
```

Tries each alternative in order. Returns the first successful match. This is **ordered choice**, not the longest match.

### Optional

```
optional := expr?
```

Matches zero or one occurrence of the expression.

### Zero or More

```
zeroOrMore := expr*
```

Matches zero or more occurrences of the expression.

### One or More

```
oneOrMore := expr+
```

Matches one or more occurrences of the expression.

### And Predicate (Lookahead)

```
and := &expr
```

Succeeds if `expr` matches, but doesn't consume any input. Used for lookahead.

### Not Predicate (Negative Lookahead)

```
not := !expr
```

Succeeds if `expr` does NOT match, and doesn't consume any input.

### Parentheses

```
grouped := ("a" | "b") "c"
```

Parentheses group expressions to control precedence.

## Terminals

### String Literals

```
keyword := "if"
weakKeyword := 'else'
```

Double quotes create strong keywords (preserved in AST). Single quotes create weak keywords (removed during simplification).

### Built-in Token Types

Runic provides several built-in token types:

- `INTEGER` - Integer literals (e.g., `42`, `0x1A`, `123u32`)
- `FLOAT` - Floating-point literals (e.g., `3.14`, `2.5e10`, `1.0f32`)
- `STRING` - String literals (e.g., `"hello"`)
- `IDENT` - Identifiers (e.g., `myVariable`)
- `EOF` - End of file
- `INTTYPE` - Integer type specifiers (e.g., `i32`, `u64`)
- `UINTTYPE` - Unsigned integer type specifiers
- `RANDUINT` - Random integer width specifiers

### Empty

```
empty := EMPTY
```

The `EMPTY` terminal matches nothing (epsilon production).

## Left-Recursion

Runic supports **direct left-recursion** within a single rule:

```
# Left-recursive addition (left-associative)
expr := expr "+" term
      | term
```

This is implemented using the seed algorithm from Warth et al. (2008). The parser:
1. Seeds with an empty match
2. Grows the match as far as possible
3. Uses memoization to cache results

**Limitations:**
- Only direct left-recursion (within the same rule)
- Indirect left-recursion (through multiple rules) is not supported
- Hidden left-recursion (through nullable rules) is not supported

## Operator Precedence

Use left-recursion to define operator precedence:

```
expr := addExpr

addExpr := addExpr "+" mulExpr   # Lower precedence
         | addExpr "-" mulExpr
         | mulExpr

mulExpr := mulExpr "*" primary    # Higher precedence
         | mulExpr "/" primary
         | primary

primary := INTEGER | "(" expr ")"
```

The parser evaluates higher-precedence rules first because they appear lower in the grammar.

## AST Simplification

### Weak Rules

Weak rules (using `:` instead of `:=`) are removed during simplification:

```
# This rule will be removed, its children promoted
statement : exprStatement | returnStatement
```

### Weak Keywords

Keywords in single quotes are removed during simplification:

```
ifStatement := 'if' expr 'then' block 'end'
```

The keywords `'if'`, `'then'`, and `'end'` will be removed from the AST.

### Strong Objects

Some tokens are always preserved:
- String literals in double quotes
- Integer and float literals
- Identifiers
- Tokens from strong rules

## Best Practices

1. **Use weak rules for grouping** - Rules that just group alternatives should be weak
2. **Use weak keywords for punctuation** - Operators and keywords that don't carry semantic meaning
3. **Use strong rules for semantic constructs** - Statements, expressions, declarations
4. **Left-recursion for operators** - Use left-recursive rules for left-associative operators
5. **Order matters in choice** - Put more specific alternatives before general ones

## Example: Expression Grammar

```
# Complete expression grammar with precedence

expr := selectExpr

# Ternary operator (lowest precedence)
selectExpr := orExpr "?" expr ":" expr
            | orExpr

# Logical OR
orExpr := orExpr "||" andExpr
        | andExpr

# Logical AND  
andExpr := andExpr "&&" relExpr
         | relExpr

# Relational operators
relExpr := relExpr ("<" | ">" | "<=" | ">=" | "==" | "!=") addExpr
         | addExpr

# Addition/Subtraction
addExpr := addExpr ("+" | "-") mulExpr
         | mulExpr

# Multiplication/Division
mulExpr := mulExpr ("*" | "/" | "%") unaryExpr
         | unaryExpr

# Unary operators
unaryExpr := ("-" | "!" | "~") unaryExpr
           | primary

# Primary expressions
primary := INTEGER
         | FLOAT
         | STRING
         | IDENT
         | "(" expr ")"
```

## Grammar Testing

All implementations should pass the same conformance tests. See `tests/conformance/` for standard test cases that verify:

- Basic parsing
- Left-recursion handling
- AST simplification
- Error reporting
- Token recognition

## References

- **PEG Paper**: Bryan Ford, "Parsing Expression Grammars: A Recognition-Based Syntactic Foundation" (2004)
- **Left-Recursion**: Alessandro Warth et al., "Packrat Parsers Can Support Left Recursion" (2008)
- **Rune Language**: https://github.com/google/rune
