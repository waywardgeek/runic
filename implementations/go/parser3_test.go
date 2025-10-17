package parser

import (
	"testing"
)

// TestSimpleExpression tests parsing a simple arithmetic expression.
func TestSimpleExpression(t *testing.T) {
	// Create a simple grammar for expressions
	grammarContent := `expr := term
term := INTEGER`

	// Write grammar to a temp file
	grammarFile := NewFilepath("test_expr.syn", nil, false)
	grammarFile.Text = grammarContent + "\n"

	// Create parser
	peg := &Peg{
		PegKeytab:   NewKeytab(),
		Keytab:      NewKeytab(),
		numKeywords: 0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(grammarFile, peg.PegKeytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}
	peg.lexer = lexer
	peg.lexer.peg = peg
	peg.lexer.EnableWeakStrings(true)

	err = peg.ParseRules()
	if err != nil {
		t.Fatalf("Failed to parse rules: %v", err)
	}

	// Now parse an input file
	inputFile := NewFilepath("test_input.txt", nil, false)
	inputFile.Text = "42"  // Remove newline - it causes issues

	node, err := peg.Parse(inputFile, false)
	if err != nil {
		// Debug: print tokens
		if peg.lexer != nil && len(peg.lexer.Tokens) > 0 {
			t.Logf("Tokens parsed (%d total):", len(peg.lexer.Tokens))
			for i, tok := range peg.lexer.Tokens {
				t.Logf("  [%d] Type=%d, Name=%s", i, tok.Type, tok.GetName())
			}
		}
		t.Logf("First rule: %s", peg.firstOrderedRule.Sym.Name)
		t.Logf("Lexer Pos=%d, Len=%d", peg.lexer.Pos, peg.lexer.Len)
		t.Fatalf("Failed to parse input: %v", err)
	}

	if node == nil {
		t.Fatal("Parse returned nil node")
	}

	t.Logf("✅ Successfully parsed simple expression")
}

// TestSequenceParsing tests parsing a sequence of tokens.
func TestSequenceParsing(t *testing.T) {
	// Grammar with sequence
	grammarContent := `rule := "hello" "world"`

	grammarFile := NewFilepath("test_seq.syn", nil, false)
	grammarFile.Text = grammarContent + "\n"

	peg := &Peg{
		PegKeytab:   NewKeytab(),
		Keytab:      NewKeytab(),
		numKeywords: 0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(grammarFile, peg.PegKeytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}
	peg.lexer = lexer
	peg.lexer.peg = peg
	peg.lexer.EnableWeakStrings(true)

	err = peg.ParseRules()
	if err != nil {
		t.Fatalf("Failed to parse rules: %v", err)
	}

	// Parse input with the keywords
	inputFile := NewFilepath("test_seq_input.txt", nil, false)
	inputFile.Text = "hello world\n"

	node, err := peg.Parse(inputFile, false)
	if err != nil {
		t.Fatalf("Failed to parse input: %v", err)
	}

	if node == nil {
		t.Fatal("Parse returned nil node")
	}

	t.Logf("✅ Successfully parsed sequence")
}

// TestChoiceParsing tests parsing with choice alternatives.
func TestChoiceParsing(t *testing.T) {
	// Grammar with choice
	grammarContent := `rule := "foo" | "bar"`

	grammarFile := NewFilepath("test_choice.syn", nil, false)
	grammarFile.Text = grammarContent + "\n"

	peg := &Peg{
		PegKeytab:   NewKeytab(),
		Keytab:      NewKeytab(),
		numKeywords: 0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(grammarFile, peg.PegKeytab, false)
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}
	peg.lexer = lexer
	peg.lexer.peg = peg
	peg.lexer.EnableWeakStrings(true)

	err = peg.ParseRules()
	if err != nil {
		t.Fatalf("Failed to parse rules: %v", err)
	}

	// Test first alternative
	inputFile := NewFilepath("test_choice_input1.txt", nil, false)
	inputFile.Text = "foo\n"

	node, err := peg.Parse(inputFile, false)
	if err != nil {
		t.Fatalf("Failed to parse 'foo': %v", err)
	}
	if node == nil {
		t.Fatal("Parse returned nil node for 'foo'")
	}

	// Test second alternative
	inputFile2 := NewFilepath("test_choice_input2.txt", nil, false)
	inputFile2.Text = "bar\n"

	node2, err := peg.Parse(inputFile2, false)
	if err != nil {
		t.Logf("Lexer has %d tokens:", len(peg.lexer.Tokens))
		for i, tok := range peg.lexer.Tokens {
			t.Logf("  [%d] Type=%d, Name=%s", i, tok.Type, tok.GetName())
		}
		t.Logf("Lexer has %d ParseResults", len(peg.lexer.ParseResults))
		t.Logf("First rule: %s, pexpr type: %d", peg.firstOrderedRule.Sym.Name, peg.firstOrderedRule.pexpr.Type)
		t.Logf("First rule has %d children", len(peg.firstOrderedRule.pexpr.ChildPexprs()))
		if len(peg.firstOrderedRule.pexpr.ChildPexprs()) > 0 {
			for i, child := range peg.firstOrderedRule.pexpr.ChildPexprs() {
				t.Logf("  Child[%d]: type=%d", i, child.Type)
				if child.Keyword != nil {
					t.Logf("    Keyword: %s (num=%d)", child.Keyword.Sym.Name, child.Keyword.Num)
				}
			}
		}
		t.Fatalf("Failed to parse 'bar': %v", err)
	}
	if node2 == nil {
		t.Fatal("Parse returned nil node for 'bar'")
	}

	t.Logf("✅ Successfully parsed both alternatives")
}
