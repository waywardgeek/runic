# Contributing to Runic

Thank you for your interest in contributing to Runic! This document provides guidelines for contributions.

## Ways to Contribute

### 1. New Language Implementations

We welcome implementations in new programming languages! See [docs/porting-guide.md](docs/porting-guide.md) for detailed instructions.

**Process:**
1. Open an issue announcing your intent to implement language X
2. Follow the porting guide structure (Phases 1-3)
3. Ensure all conformance tests pass
4. Submit a pull request

**Requirements:**
- Must pass all conformance tests in `tests/conformance/`
- Must include unit tests
- Must include README with usage examples
- Should follow language idioms while maintaining semantic compatibility

### 2. Example Grammars

Contributions of `.syn` grammar files are welcome:

**Examples:**
- Programming language grammars
- Data format parsers (XML, YAML, etc.)
- Domain-specific languages
- Mathematical notation

**Requirements:**
- Place in `examples/grammars/`
- Include example input files in `examples/inputs/`
- Add documentation comments in the grammar
- Test with at least one implementation

### 3. Documentation

Help improve documentation:
- Fix typos and clarify explanations
- Add tutorials and examples
- Translate documentation to other languages
- Add diagrams and visual aids

### 4. Bug Reports

If you find a bug:
1. Check if it's already reported in issues
2. Create a new issue with:
   - Description of the problem
   - Steps to reproduce
   - Expected vs actual behavior
   - Implementation/version affected
   - Minimal grammar/input that demonstrates the bug

### 5. Bug Fixes

To fix a bug:
1. Reference the issue number in your PR
2. Add a test that fails without the fix
3. Ensure all existing tests still pass
4. Update documentation if needed

## Code Style Guidelines

### General Principles

- **Consistency**: Follow the reference implementation (Go) semantics
- **Clarity**: Code should be self-documenting with comments for complex logic
- **Testing**: All code should be tested
- **Documentation**: Public APIs should be documented

### Language-Specific Style

- **Go**: Follow `gofmt` and Go conventions
- **Python**: Follow PEP 8
- **Rust**: Follow `rustfmt` and Rust conventions
- **C**: Follow K&R style with 4-space indentation
- **JavaScript**: Follow StandardJS or Prettier

## Testing Requirements

### Unit Tests

- Test each phase independently
- Test edge cases and error conditions
- Test memory management (where applicable)

### Integration Tests

- Test complete parse workflows
- Test with realistic grammars
- Test error reporting

### Conformance Tests

**All implementations must pass these:**
- `tests/conformance/basic_parsing.txt`
- `tests/conformance/left_recursion.txt`
- `tests/conformance/ast_simplification.txt`
- `tests/conformance/edge_cases.txt`

## Pull Request Process

1. **Fork** the repository
2. **Create a branch** for your feature/fix: `git checkout -b my-feature`
3. **Make changes** and commit with clear messages
4. **Test thoroughly** - all tests must pass
5. **Update documentation** if needed
6. **Submit PR** with:
   - Clear description of changes
   - Reference to related issues
   - Test results
   - Any breaking changes noted

### PR Review Process

- Maintainers will review within 1 week
- Address review comments promptly
- Once approved, a maintainer will merge

## Semantic Versioning

Runic follows semantic versioning (MAJOR.MINOR.PATCH):
- **MAJOR**: Incompatible grammar changes
- **MINOR**: New features, backward compatible
- **PATCH**: Bug fixes

## Compatibility Promise

**Grammar Compatibility**: Once a grammar feature is released, it won't be removed or changed incompatibly without a major version bump.

**Implementation Compatibility**: All implementations should produce identical parse trees for valid input and reject invalid input consistently.

## Architecture Decisions

For significant changes to architecture or algorithms:
1. Open an issue for discussion before implementing
2. Explain the problem and proposed solution
3. Consider impact on all language implementations
4. Update architecture documentation

## Code of Conduct

### Our Standards

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Assume good intentions

### Unacceptable Behavior

- Harassment or discrimination
- Personal attacks
- Trolling or insulting comments
- Publishing others' private information

### Enforcement

Violations may result in:
1. Warning
2. Temporary ban
3. Permanent ban

Report issues to the maintainers privately.

## Development Setup

### Go Implementation

```bash
cd implementations/go
go test ./...
```

### Running Conformance Tests

```bash
cd implementations/go
go test -run Conformance
```

### Building Documentation

```bash
# Use any markdown viewer or GitHub preview
```

## Questions?

- **General questions**: Open a discussion on GitHub
- **Bug reports**: Open an issue
- **Security issues**: Email maintainers directly (see SECURITY.md)
- **Architecture questions**: Reference docs/architecture.md

## License

By contributing, you agree that your contributions will be licensed under the Apache License 2.0.

## Recognition

Contributors will be recognized in:
- The README.md contributors section
- The release notes for their contributions
- The AUTHORS file

Thank you for helping make Runic better!
