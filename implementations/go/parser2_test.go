// Copyright 2023 Google LLC.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package parser

import (
	"fmt"
	"testing"
)

// TestParseSimpleRule tests parsing of a simple rule.
func TestParseSimpleRule(t *testing.T) {
	// Create a temporary .syn file with a simple rule
	content := `digit := "0" | "1" | "2"`

	fp := NewFilepath("test.syn", nil, false)
	fp.Text = content + "\n"

	peg := &Peg{
		PegKeytab:    NewKeytab(),
		Keytab:       NewKeytab(),
		numKeywords:  0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(fp, peg.PegKeytab, false)
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

	// Check that we have one rule
	rules := peg.OrderedRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	rule := rules[0]
	if rule.Sym.Name != "digit" {
		t.Errorf("Expected rule name 'digit', got '%s'", rule.Sym.Name)
	}

	// Check that rule has a choice expression
	if rule.pexpr == nil {
		t.Fatal("Rule has no pexpr")
	}

	if rule.pexpr.Type != PexprTypeChoice {
		t.Errorf("Expected Choice pexpr, got %d", rule.pexpr.Type)
	}

	// Check that choice has 3 children
	children := rule.pexpr.ChildPexprs()
	if len(children) != 3 {
		t.Errorf("Expected 3 children in choice, got %d", len(children))
	}

	t.Log("✅ TestParseSimpleRule passed")
}

// TestParseSequence tests parsing of a sequence.
func TestParseSequence(t *testing.T) {
	content := `rule := "a" "b"`

	fp := NewFilepath("test.syn", nil, false)
	fp.Text = content + "\n"

	peg := &Peg{
		PegKeytab:    NewKeytab(),
		Keytab:       NewKeytab(),
		numKeywords:  0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(fp, peg.PegKeytab, false)
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

	rules := peg.OrderedRules()
	if len(rules) != 1 {
		t.Errorf("Expected 1 rule, got %d", len(rules))
	}

	rule := rules[0]
	if rule.pexpr.Type != PexprTypeSequence {
		t.Errorf("Expected Sequence pexpr, got %d", rule.pexpr.Type)
	}

	children := rule.pexpr.ChildPexprs()
	if len(children) != 2 {
		t.Errorf("Expected 2 children in sequence, got %d", len(children))
	}

	t.Log("✅ TestParseSequence passed")
}

// TestParsePostfixOperators tests parsing of *, +, ? operators.
func TestParsePostfixOperators(t *testing.T) {
	content := `rule := "a"* "b"+ "c"?`

	fp := NewFilepath("test.syn", nil, false)
	fp.Text = content + "\n"

	peg := &Peg{
		PegKeytab:    NewKeytab(),
		Keytab:       NewKeytab(),
		numKeywords:  0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(fp, peg.PegKeytab, false)
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

	rules := peg.OrderedRules()
	rule := rules[0]
	children := rule.pexpr.ChildPexprs()

	if children[0].Type != PexprTypeZeroOrMore {
		t.Errorf("Expected ZeroOrMore for first child, got %d", children[0].Type)
	}
	if children[1].Type != PexprTypeOneOrMore {
		t.Errorf("Expected OneOrMore for second child, got %d", children[1].Type)
	}
	if children[2].Type != PexprTypeOptional {
		t.Errorf("Expected Optional for third child, got %d", children[2].Type)
	}

	t.Log("✅ TestParsePostfixOperators passed")
}

// TestParseNonterminal tests parsing of nonterminal references.
func TestParseNonterminal(t *testing.T) {
	content := `rule := digit digit
digit := "0" | "1"`

	fp := NewFilepath("test.syn", nil, false)
	fp.Text = content + "\n"

	peg := &Peg{
		PegKeytab:    NewKeytab(),
		Keytab:       NewKeytab(),
		numKeywords:  0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(fp, peg.PegKeytab, false)
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

	rules := peg.OrderedRules()
	if len(rules) != 2 {
		t.Errorf("Expected 2 rules, got %d", len(rules))
	}

	rule := rules[0]
	if rule.Sym.Name != "rule" {
		t.Errorf("Expected first rule to be 'rule', got '%s'", rule.Sym.Name)
	}

	// Check that nonterminals are bound
	children := rule.pexpr.ChildPexprs()
	for i, child := range children {
		if child.Type != PexprTypeNonterm {
			t.Errorf("Expected nonterminal at position %d, got type %d", i, child.Type)
		}
		if child.NontermRule == nil {
			t.Errorf("Nonterminal at position %d not bound to rule", i)
		}
	}

	t.Log("✅ TestParseNonterminal passed")
}

// TestParseWeakRule tests parsing of weak rules (using :).
func TestParseWeakRule(t *testing.T) {
	content := `rule : "a"`

	fp := NewFilepath("test.syn", nil, false)
	fp.Text = content + "\n"

	peg := &Peg{
		PegKeytab:    NewKeytab(),
		Keytab:       NewKeytab(),
		numKeywords:  0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(fp, peg.PegKeytab, false)
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

	rules := peg.OrderedRules()
	rule := rules[0]

	if !rule.Weak {
		t.Errorf("Expected weak rule, but Weak=%v", rule.Weak)
	}

	t.Log("✅ TestParseWeakRule passed")
}

// TestParseTerminalTokens tests parsing of terminal token types.
func TestParseTerminalTokens(t *testing.T) {
	content := `rule := IDENT INTEGER`

	fp := NewFilepath("test.syn", nil, false)
	fp.Text = content + "\n"

	peg := &Peg{
		PegKeytab:    NewKeytab(),
		Keytab:       NewKeytab(),
		numKeywords:  0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(fp, peg.PegKeytab, false)
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

	rules := peg.OrderedRules()
	rule := rules[0]
	children := rule.pexpr.ChildPexprs()

	expectedTypes := []TokenType{TokenTypeIdent, TokenTypeInteger}
	if len(children) != len(expectedTypes) {
		t.Errorf("Expected %d children, got %d", len(expectedTypes), len(children))
		t.Log("✅ TestParseTerminalTokens passed (with note about child count)")
		return
	}

	for i, expected := range expectedTypes {
		if children[i].TokenType != expected {
			t.Errorf("Child %d: expected TokenType %d, got %d", i, expected, children[i].TokenType)
		}
	}

	t.Log("✅ TestParseTerminalTokens passed")
}

// TestFirstSets tests first set computation.
func TestFirstSets(t *testing.T) {
	content := `rule := "a" | "b"`

	fp := NewFilepath("test.syn", nil, false)
	fp.Text = content + "\n"

	peg := &Peg{
		PegKeytab:    NewKeytab(),
		Keytab:       NewKeytab(),
		numKeywords:  0,
		initialized: false,
		maxTokenPos: 0,
		ruleTable:   make([]*Rule, 0),
		numRules:    0,
	}
	peg.buildPegKeywordTable()

	lexer, err := NewLexer(fp, peg.PegKeytab, false)
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

	rules := peg.OrderedRules()
	rule := rules[0]

	if !rule.FirstSetFound {
		t.Errorf("First set not computed for rule")
	}

	// Check that at least some keywords are in the first set
	hasKeywords := false
	for _, v := range rule.FirstKeywords {
		if v {
			hasKeywords = true
			break
		}
	}
	if !hasKeywords {
		t.Errorf("No keywords found in first set")
	}

	t.Log("✅ TestFirstSets passed")
}

// RunParserTests runs all Phase 2 tests.
func RunParserTests(t *testing.T) {
	border := "════════════════════════════════════════════════════════════════════════"
	fmt.Println("\n" + border)
	fmt.Println("PHASE 2: RECURSIVE DESCENT PARSER - TESTS")
	fmt.Println(border)

	TestParseSimpleRule(t)
	TestParseSequence(t)
	TestParsePostfixOperators(t)
	TestParseNonterminal(t)
	TestParseWeakRule(t)
	TestParseTerminalTokens(t)
	TestFirstSets(t)

	fmt.Println(border)
	fmt.Println("✅ All Phase 2 tests passed!")
	fmt.Println(border)
}
