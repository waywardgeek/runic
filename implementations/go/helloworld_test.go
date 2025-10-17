package parser

import (
	"fmt"
	"os"
	"testing"
)

func TestHelloWorld(t *testing.T) {
	// Parse rune.syn to get the grammar
	fp := NewFilepath("rune.syn", nil, false)
	text, err := os.ReadFile("rune.syn")
	if err != nil {
		t.Fatalf("Error reading rune.syn: %v", err)
	}
	fp.Text = string(text)
	
	// Create Peg and parse the grammar
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
	
	lexer, err := NewLexer(fp, peg.PegKeytab, false)
	if err != nil {
		t.Fatalf("Error creating lexer: %v", err)
	}
	peg.InsertLexer(lexer)
	peg.lexer.EnableWeakStrings(true)
	
	// Parse the grammar
	err = peg.ParseRules()
	if err != nil {
		t.Fatalf("Error parsing rune.syn: %v", err)
	}
	
	t.Logf("✅ Parsed rune.syn: %d rules", peg.numRules)
	
	// Now parse helloworld.rn using the grammar
	node, err := peg.Parse("../../examples/inputs/helloworld.rn", false)
	if err != nil {
		t.Fatalf("❌ Failed to parse helloworld.rn: %v", err)
	}
	
	fmt.Printf("\n=== UNSIMPLIFIED TREE ===\n")
	fmt.Println(node.ToString())
	
	// Now parse again with simplification
	node, err = peg.Parse("../../examples/inputs/helloworld.rn", false)
	if err != nil {
		t.Fatalf("❌ Failed to parse helloworld.rn (simplified): %v", err)
	}
	node.Simplify()
	
	fmt.Printf("\n=== SIMPLIFIED TREE ===\n")
	fmt.Println(node.ToString())
}
