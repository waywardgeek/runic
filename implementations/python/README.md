# Runic Python Implementation

🚧 **Status**: Planned - Not yet implemented

Python implementation of the Runic PEG parser with left-recursion support.

## Planned Features

- ✅ Full PEG grammar support
- ✅ Left-recursion handling
- ✅ Packrat parsing (memoization)
- ✅ AST simplification
- ✅ UTF-8 support
- ✅ Type hints for better IDE support

## Installation (Future)

```bash
pip install runic
```

## Usage (Planned)

```python
from runic import Peg

# Load grammar
peg = Peg.from_file("mygrammar.syn")

# Parse input
node = peg.parse("input.txt")

# Simplify AST
node.simplify()

# Print result
print(node.to_string())
```

## Development

### Setup

```bash
python -m venv venv
source venv/bin/activate
pip install -r requirements.txt
```

### Testing

```bash
pytest tests/
```

## Contributing

Want to implement the Python version? See [../../docs/porting-guide.md](../../docs/porting-guide.md) for instructions.

Steps:
1. Port Phase 1 (data structures)
2. Port Phase 2 (grammar parser)
3. Port Phase 3 (PEG engine)
4. Ensure conformance tests pass

## Reference

The Go implementation in `../go/` is the reference implementation. Follow its structure and semantics.
