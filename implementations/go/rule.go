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

// Rule represents a single grammar rule in a PEG grammar.
type Rule struct {
	Sym      *Sym   // Symbol name of the rule
	Location Location
	Weak     bool   // If true, this is a weak rule (collapsed in parse tree)

	// OneToOne Rule Pexpr cascade
	pexpr *Pexpr

	// TailLinked Peg:"Ordered" Rule:"Ordered"
	peg              *Peg
	nextOrderedRule  *Rule
	prevOrderedRule  *Rule

	// Hashed Peg Rule cascade ("sym") - child side
	nextHashedPegRule *Rule

	// TailLinked Rule:"Nonterm" Pexpr:"Nonterm" cascade
	// (Pexprs that reference this rule as a nonterminal)
	firstNontermPexpr *Pexpr
	lastNontermPexpr  *Pexpr

	// DoublyLinked Rule ParseResult cascade
	firstParseResult *ParseResult
	lastParseResult  *ParseResult

	// Hashed Rule:"Hashed" ParseResult:"Hashed" cascade ("pos")
	hashedParseResultTable []*ParseResult
	numHashedParseResults  uint32

	// First set computation
	FirstKeywords   []bool
	FirstTokens     []bool
	FirstSetFound   bool
	findingFirstSet bool // For loop detection
	CanBeEmpty      bool
}

// NewRule creates a new grammar rule.
func NewRule(peg *Peg, sym *Sym, pexpr *Pexpr, location Location) *Rule {
	r := &Rule{
		Sym:                    sym,
		Location:               location,
		Weak:                   false,
		pexpr:                  pexpr,
		peg:                    peg,
		FirstKeywords:          make([]bool, 0),
		FirstTokens:            make([]bool, 256), // Approximate for token types
		FirstSetFound:          false,
		findingFirstSet:        false,
		CanBeEmpty:             false,
		hashedParseResultTable: make([]*ParseResult, 0),
		numHashedParseResults:  0,
	}

	// If pexpr is provided, set the OneToOne relationship
	if pexpr != nil {
		pexpr.rule = r
	}

	return r
}

// ============================================================================
// OneToOne Rule Pexpr cascade
// ============================================================================

// Pexpr returns the root parsing expression for this rule.
func (r *Rule) Pexpr() *Pexpr {
	return r.pexpr
}

// InsertPexpr sets the parsing expression for this rule (OneToOne).
func (r *Rule) InsertPexpr(pexpr *Pexpr) {
	if pexpr == nil {
		return
	}
	if r.pexpr != nil {
		// Remove old pexpr
		r.pexpr.rule = nil
	}
	r.pexpr = pexpr
	pexpr.rule = r
}

// RemovePexpr removes the parsing expression from this rule.
func (r *Rule) RemovePexpr(pexpr *Pexpr) {
	if pexpr == nil || pexpr.rule != r {
		return
	}
	r.pexpr = nil
	pexpr.rule = nil
}

// ============================================================================
// TailLinked Rule:"Nonterm" Pexpr:"Nonterm" cascade
// ============================================================================

// AppendNontermPexpr adds a Pexpr that references this rule as a nonterminal.
func (r *Rule) AppendNontermPexpr(pexpr *Pexpr) {
	if pexpr == nil {
		return
	}

	if r.lastNontermPexpr == nil {
		r.firstNontermPexpr = pexpr
	} else {
		r.lastNontermPexpr.nextNontermPexpr = pexpr
	}
	r.lastNontermPexpr = pexpr
	pexpr.nextNontermPexpr = nil
}

// FirstNontermPexpr returns the first Pexpr that references this rule.
func (r *Rule) FirstNontermPexpr() *Pexpr {
	return r.firstNontermPexpr
}

// NontermPexprs returns a slice of all nonterminal Pexprs.
func (r *Rule) NontermPexprs() []*Pexpr {
	var pexprs []*Pexpr
	for p := r.firstNontermPexpr; p != nil; p = p.nextNontermPexpr {
		pexprs = append(pexprs, p)
	}
	return pexprs
}

// ============================================================================
// DoublyLinked Rule ParseResult cascade
// ============================================================================

// AppendParseResult adds a ParseResult for this rule (DoublyLinked).
func (r *Rule) AppendParseResult(pr *ParseResult) {
	if pr == nil {
		return
	}

	if r.lastParseResult == nil {
		r.firstParseResult = pr
	} else {
		r.lastParseResult.nextRuleParseResult = pr
		pr.prevRuleParseResult = r.lastParseResult
	}
	r.lastParseResult = pr
	pr.ruleParent = r
}

// RemoveParseResult removes a ParseResult from this rule.
func (r *Rule) RemoveParseResult(pr *ParseResult) {
	if pr == nil || pr.ruleParent != r {
		return
	}

	if pr.prevRuleParseResult != nil {
		pr.prevRuleParseResult.nextRuleParseResult = pr.nextRuleParseResult
	} else {
		r.firstParseResult = pr.nextRuleParseResult
	}

	if pr.nextRuleParseResult != nil {
		pr.nextRuleParseResult.prevRuleParseResult = pr.prevRuleParseResult
	} else {
		r.lastParseResult = pr.prevRuleParseResult
	}

	pr.prevRuleParseResult = nil
	pr.nextRuleParseResult = nil
	pr.ruleParent = nil
}

// ParseResults returns a slice of all ParseResults for iteration.
func (r *Rule) ParseResults() []*ParseResult {
	var results []*ParseResult
	for pr := r.firstParseResult; pr != nil; pr = pr.nextRuleParseResult {
		results = append(results, pr)
	}
	return results
}

// ============================================================================
// Hashed Rule:"Hashed" ParseResult:"Hashed" cascade ("pos")
// ============================================================================

// FindHashedParseResult looks up a ParseResult by position (hash key).
func (r *Rule) FindHashedParseResult(pos uint32) *ParseResult {
	if len(r.hashedParseResultTable) == 0 {
		return nil
	}

	hash := pos & (uint32(len(r.hashedParseResultTable)) - 1)
	for entry := r.hashedParseResultTable[hash]; entry != nil; entry = entry.nextHashedRuleParseResult {
		if entry.Pos == pos {
			return entry
		}
	}
	return nil
}

// InsertHashedParseResult adds a ParseResult to the hash table.
func (r *Rule) InsertHashedParseResult(pr *ParseResult) {
	if pr == nil {
		return
	}

	// Resize if needed
	if r.numHashedParseResults == uint32(len(r.hashedParseResultTable)) {
		if len(r.hashedParseResultTable) == 0 {
			r.hashedParseResultTable = make([]*ParseResult, 32)
		} else {
			r.resizeHashedParseResultTable()
		}
	}

	hash := pr.Pos & (uint32(len(r.hashedParseResultTable)) - 1)
	pr.nextHashedRuleParseResult = r.hashedParseResultTable[hash]
	r.hashedParseResultTable[hash] = pr
	r.numHashedParseResults++
}

// RemoveHashedParseResult removes a ParseResult from the hash table.
func (r *Rule) RemoveHashedParseResult(pr *ParseResult) {
	if pr == nil || len(r.hashedParseResultTable) == 0 {
		return
	}

	hash := pr.Pos & (uint32(len(r.hashedParseResultTable)) - 1)
	var prev *ParseResult
	for entry := r.hashedParseResultTable[hash]; entry != nil; entry = entry.nextHashedRuleParseResult {
		if entry == pr {
			if prev == nil {
				r.hashedParseResultTable[hash] = pr.nextHashedRuleParseResult
			} else {
				prev.nextHashedRuleParseResult = pr.nextHashedRuleParseResult
			}
			pr.nextHashedRuleParseResult = nil
			r.numHashedParseResults--
			return
		}
		prev = entry
	}
}

// resizeHashedParseResultTable doubles the hash table size and rehashes.
func (r *Rule) resizeHashedParseResultTable() {
	if len(r.hashedParseResultTable) == 0 {
		return
	}

	oldTable := r.hashedParseResultTable
	newSize := len(oldTable) * 2
	r.hashedParseResultTable = make([]*ParseResult, newSize)

	newMask := uint32(newSize - 1)

	// Rehash all entries
	for _, entry := range oldTable {
		for entry != nil {
			nextEntry := entry.nextHashedRuleParseResult
			newHash := entry.Pos & newMask
			entry.nextHashedRuleParseResult = r.hashedParseResultTable[newHash]
			r.hashedParseResultTable[newHash] = entry
			entry = nextEntry
		}
	}
}

// ============================================================================
// First set computation
// ============================================================================

// FindFirstSet computes the first set of tokens for this rule.
func (r *Rule) FindFirstSet() {
	if r.FirstSetFound {
		return
	}

	if r.findingFirstSet {
		// Loop detected - left recursion
		return
	}

	r.findingFirstSet = true

	// Ensure arrays are sized appropriately
	if r.peg != nil {
		if len(r.FirstKeywords) < int(r.peg.numKeywords) {
			r.FirstKeywords = make([]bool, r.peg.numKeywords)
		}
	}

	if r.pexpr != nil {
		r.pexpr.FindFirstSet(r.FirstKeywords, r.FirstTokens)
		r.CanBeEmpty = r.pexpr.CanBeEmpty
	}

	r.FirstSetFound = true
	r.findingFirstSet = false
}

// ============================================================================
// Clear memoization caches (for starting a new parse)
// ============================================================================

// ClearHashedParseResults removes all ParseResults from the hash table.
func (r *Rule) ClearHashedParseResults() {
	r.hashedParseResultTable = make([]*ParseResult, 0)
	r.numHashedParseResults = 0
}

// ClearParseResults removes all ParseResults from the doubly-linked list.
func (r *Rule) ClearParseResults() {
	r.firstParseResult = nil
	r.lastParseResult = nil
}

// ============================================================================
// String representation
// ============================================================================

// ToString returns the string representation of this rule.
func (r *Rule) ToString() string {
	if r.pexpr == nil {
		return r.Sym.Name
	}
	s := r.Sym.Name
	s += ": "
	s += r.pexpr.ToString()
	return s
}

// Dump outputs debugging information about this rule.
func (r *Rule) Dump() {
	fmt.Println(r.ToString())
}
