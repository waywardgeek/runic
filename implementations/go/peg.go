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

import "fmt"

// Peg is the main PEG parser class.
type Peg struct {
	// Keyword tables
	PegKeytab *Keytab // Keywords for parsing .syn files
	Keytab    *Keytab // Keywords for parsing input files

	// Current lexer
	lexer *Lexer

	// Hashed Peg Rule cascade ("sym") - rules by symbol name
	ruleTable       []*Rule
	numRules        uint32
	nextHashedRule  *Rule

	// TailLinked Peg:"Ordered" Rule:"Ordered" - all rules in order
	firstOrderedRule *Rule
	lastOrderedRule  *Rule

	// Parser state
	maxTokenPos   uint32
	savedToken1   *Token
	savedToken2   *Token
	numKeywords   uint32
	initialized   bool
	simplifyNodes bool // Whether to simplify the node tree after parsing

	// Builtin keywords for PEG syntax
	kwColon       *Keyword
	kwColonEquals *Keyword
	kwPipe        *Keyword
	kwOpenParen   *Keyword
	kwCloseParen  *Keyword
	kwStar        *Keyword
	kwPlus        *Keyword
	kwQuestion    *Keyword
	kwAnd         *Keyword
	kwNot         *Keyword
	kwNewline     *Keyword
	kwEmpty       *Keyword
	kwEof         *Keyword
	kwIdent       *Keyword
	kwInteger     *Keyword
	kwFloat       *Keyword
	kwString      *Keyword
	kwRandInt     *Keyword
	kwIntType     *Keyword
	kwUintType    *Keyword
}

// NewPeg creates a new Peg parser for the given syntax file.
func NewPeg(syntaxFileName string) (*Peg, error) {
	peg := &Peg{
		PegKeytab:     NewKeytab(),
		Keytab:        NewKeytab(),
		numKeywords:   0,
		initialized:   false,
		maxTokenPos:   0,
		ruleTable:     make([]*Rule, 0),
		numRules:      0,
		simplifyNodes: true, // Default to simplifying nodes
	}

	// Build the PEG keyword table
	peg.buildPegKeywordTable()

	// Create lexer for the syntax file
	filepath := NewFilepath(syntaxFileName, nil, false)
	lexer, err := NewLexer(filepath, peg.PegKeytab, true)
	if err != nil {
		return nil, fmt.Errorf("Failed to create lexer: %v", err)
	}
	peg.lexer = lexer
	peg.lexer.peg = peg
	peg.lexer.EnableWeakStrings(true)

	// Parse the rules from the syntax file
	if err := peg.ParseRules(); err != nil {
		return nil, fmt.Errorf("Failed to parse rules: %v", err)
	}

	return peg, nil
}

// ============================================================================
// Hashed Peg Rule cascade ("sym")
// ============================================================================

// FindRule looks up a Rule by symbol name.
func (p *Peg) FindRule(sym *Sym) *Rule {
	if len(p.ruleTable) == 0 || sym == nil {
		return nil
	}

	hash := hashSym(sym) & (uint32(len(p.ruleTable)) - 1)
	for entry := p.ruleTable[hash]; entry != nil; entry = entry.nextHashedPegRule {
		if entry.Sym == sym {
			return entry
		}
	}
	return nil
}

// InsertRule adds a Rule to the hash table.
func (p *Peg) InsertRule(rule *Rule) {
	if rule == nil {
		return
	}

	// Resize if needed
	if p.numRules == uint32(len(p.ruleTable)) {
		if len(p.ruleTable) == 0 {
			p.ruleTable = make([]*Rule, 32)
		} else {
			p.resizeRuleTable()
		}
	}

	hash := hashSym(rule.Sym) & (uint32(len(p.ruleTable)) - 1)
	rule.nextHashedPegRule = p.ruleTable[hash]
	p.ruleTable[hash] = rule
	p.numRules++
}

// RemoveRule removes a Rule from the hash table.
func (p *Peg) RemoveRule(rule *Rule) {
	if rule == nil || len(p.ruleTable) == 0 {
		return
	}

	hash := hashSym(rule.Sym) & (uint32(len(p.ruleTable)) - 1)
	var prev *Rule
	for entry := p.ruleTable[hash]; entry != nil; entry = entry.nextHashedPegRule {
		if entry == rule {
			if prev == nil {
				p.ruleTable[hash] = rule.nextHashedPegRule
			} else {
				prev.nextHashedPegRule = rule.nextHashedPegRule
			}
			rule.nextHashedPegRule = nil
			p.numRules--
			return
		}
		prev = entry
	}
}

// resizeRuleTable doubles the rule hash table and rehashes.
func (p *Peg) resizeRuleTable() {
	if len(p.ruleTable) == 0 {
		return
	}

	oldTable := p.ruleTable
	newSize := len(oldTable) * 2
	p.ruleTable = make([]*Rule, newSize)

	// Rehash all entries
	for _, entry := range oldTable {
		for entry != nil {
			nextEntry := entry.nextHashedPegRule
			newHash := hashSym(entry.Sym) & (uint32(newSize) - 1)
			entry.nextHashedPegRule = p.ruleTable[newHash]
			p.ruleTable[newHash] = entry
			entry = nextEntry
		}
	}
}

// ============================================================================
// TailLinked Peg:"Ordered" Rule:"Ordered"
// ============================================================================

// AppendOrderedRule adds a Rule to the ordered list.
func (p *Peg) AppendOrderedRule(rule *Rule) {
	if rule == nil {
		return
	}

	if p.lastOrderedRule == nil {
		p.firstOrderedRule = rule
	} else {
		p.lastOrderedRule.nextOrderedRule = rule
		rule.prevOrderedRule = p.lastOrderedRule
	}
	p.lastOrderedRule = rule
	rule.peg = p
	rule.nextOrderedRule = nil
}

// OrderedRules returns a slice of all rules in order.
func (p *Peg) OrderedRules() []*Rule {
	var rules []*Rule
	for rule := p.firstOrderedRule; rule != nil; rule = rule.nextOrderedRule {
		rules = append(rules, rule)
	}
	return rules
}

// ============================================================================
// OneToOne Peg Lexer cascade
// ============================================================================

// InsertLexer sets the current lexer.
func (p *Peg) InsertLexer(lexer *Lexer) {
	if lexer == nil {
		return
	}
	p.lexer = lexer
	lexer.peg = p
}

// ============================================================================
// Keyword table building
// ============================================================================

// buildPegKeywordTable initializes PEG syntax keywords.
func (p *Peg) buildPegKeywordTable() {
	p.kwColon = NewKeyword(p.PegKeytab, ":")
	p.kwColonEquals = NewKeyword(p.PegKeytab, ":=")
	p.kwPipe = NewKeyword(p.PegKeytab, "|")
	p.kwOpenParen = NewKeyword(p.PegKeytab, "(")
	p.kwCloseParen = NewKeyword(p.PegKeytab, ")")
	p.kwStar = NewKeyword(p.PegKeytab, "*")
	p.kwPlus = NewKeyword(p.PegKeytab, "+")
	p.kwQuestion = NewKeyword(p.PegKeytab, "?")
	p.kwAnd = NewKeyword(p.PegKeytab, "&")
	p.kwNot = NewKeyword(p.PegKeytab, "!")
	p.kwNewline = NewKeyword(p.PegKeytab, "\n")
	p.kwEmpty = NewKeyword(p.PegKeytab, "EMPTY")
	p.kwEof = NewKeyword(p.PegKeytab, "EOF")
	p.kwIdent = NewKeyword(p.PegKeytab, "IDENT")
	p.kwInteger = NewKeyword(p.PegKeytab, "INTEGER")
	p.kwFloat = NewKeyword(p.PegKeytab, "FLOAT")
	p.kwString = NewKeyword(p.PegKeytab, "STRING")
	p.kwRandInt = NewKeyword(p.PegKeytab, "RANDUINT")
	p.kwIntType = NewKeyword(p.PegKeytab, "INTTYPE")
	p.kwUintType = NewKeyword(p.PegKeytab, "UINTTYPE")
}

// ============================================================================
// Parsing: Phase 2 in parser2.go, Phase 3 in parser3.go
// ============================================================================

// SetSimplifyNodes controls whether the node tree should be simplified after parsing.
func (p *Peg) SetSimplifyNodes(simplify bool) {
	p.simplifyNodes = simplify
}

// SimplifyNodes returns whether node simplification is enabled.
func (p *Peg) SimplifyNodes() bool {
	return p.simplifyNodes
}

// ============================================================================
// Helper functions
// ============================================================================

// hashSym computes a hash for a Sym.
func hashSym(sym *Sym) uint32 {
	if sym == nil {
		return 0
	}
	// Simple hash of symbol name
	hash := uint32(0)
	for _, ch := range sym.Name {
		hash = hash*31 + uint32(ch)
	}
	return hash
}

// ToString returns a string representation of all rules.
func (p *Peg) ToString() string {
	s := ""
	for _, rule := range p.OrderedRules() {
		s += rule.ToString()
		s += "\n"
	}
	return s
}

// Dump outputs all rules.
func (p *Peg) Dump() {
	fmt.Println(p.ToString())
}
