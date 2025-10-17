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

// PexprType represents the type of a parsing expression.
type PexprType uint32

const (
	PexprTypeNonterm    PexprType = iota // Reference to a Rule
	PexprTypeTerm                         // Terminal token (INTEGER, IDENT, etc.)
	PexprTypeKeyword                      // Keyword in quotes ("if", "else", etc.)
	PexprTypeEmpty                        // Empty expression (epsilon)
	PexprTypeSequence                     // Sequence: e1 e2 e3
	PexprTypeChoice                       // Choice: e1 | e2 | e3
	PexprTypeZeroOrMore                   // Zero or more: e*
	PexprTypeOneOrMore                    // One or more: e+
	PexprTypeOptional                     // Optional: e?
	PexprTypeAnd                          // And-predicate: &e (lookahead)
	PexprTypeNot                          // Not-predicate: !e (negation)
)

// Pexpr represents a Parsing Expression in a PEG grammar.
type Pexpr struct {
	Type              PexprType
	Location          Location
	Sym               *Sym       // For keywords and nonterminals
	TokenType         TokenType  // For Term pexprs (INTEGER, IDENT, etc.)
	HasParens         bool       // Whether this was originally in parentheses
	CanBeEmpty        bool       // Whether this expression can match empty input
	Weak              bool       // If true, don't include in parse tree
	Keyword           *Keyword   // For Keyword pexprs
	NontermRule       *Rule      // For Nonterm pexprs (filled in by bindNonterms)

	// TailLinked Pexpr:"Parent" Pexpr:"Child" cascade
	firstChildPexpr *Pexpr
	lastChildPexpr  *Pexpr

	// DoublyLinked in parent-child relationship
	parentPexpr *Pexpr
	nextPexpr   *Pexpr
	prevPexpr   *Pexpr

	// TailLinked Keyword Pexpr cascade
	nextKeywordPexpr *Pexpr

	// TailLinked Rule:"Nonterm" Pexpr:"Nonterm" cascade
	nextNontermPexpr *Pexpr

	// OneToOne Rule Pexpr cascade (only for root pexprs of rules)
	rule *Rule
}

// NewPexpr creates a new parsing expression of the given type.
func NewPexpr(pexprType PexprType, location Location) *Pexpr {
	return &Pexpr{
		Type:           pexprType,
		Location:       location,
		CanBeEmpty:     false,
		Weak:           false,
		firstChildPexpr: nil,
		lastChildPexpr: nil,
		parentPexpr:    nil,
	}
}

// ============================================================================
// TailLinked Pexpr:"Parent" Pexpr:"Child" cascade
// ============================================================================

// AppendChildPexpr adds a child parsing expression at the end.
func (p *Pexpr) AppendChildPexpr(child *Pexpr) {
	if child == nil {
		return
	}
	// Verify child isn't already in a relation
	if child.parentPexpr != nil {
		panic(fmt.Sprintf("AppendChildPexpr: child already has parent"))
	}

	if p.lastChildPexpr == nil {
		p.firstChildPexpr = child
	} else {
		p.lastChildPexpr.nextPexpr = child
	}
	p.lastChildPexpr = child
	child.parentPexpr = p
	child.nextPexpr = nil
	child.prevPexpr = p.lastChildPexpr
	if len(p.ChildPexprs()) > 1 {
		// Update prevPexpr for doubly-linked semantics
		child.prevPexpr = p.ChildPexprs()[len(p.ChildPexprs())-2]
	}
}

// InsertChildPexpr inserts a child at the beginning.
func (p *Pexpr) InsertChildPexpr(child *Pexpr) {
	if child == nil {
		return
	}
	// Verify child isn't already in a relation
	if child.parentPexpr != nil {
		panic(fmt.Sprintf("InsertChildPexpr: child already has parent"))
	}

	if p.firstChildPexpr == nil {
		p.lastChildPexpr = child
	} else {
		p.firstChildPexpr.prevPexpr = child
		child.nextPexpr = p.firstChildPexpr
	}
	p.firstChildPexpr = child
	child.parentPexpr = p
	child.prevPexpr = nil
}

// RemoveChildPexpr removes a specific child.
func (p *Pexpr) RemoveChildPexpr(child *Pexpr) {
	if child == nil || child.parentPexpr != p {
		return
	}

	if child.prevPexpr != nil {
		child.prevPexpr.nextPexpr = child.nextPexpr
	} else {
		p.firstChildPexpr = child.nextPexpr
	}

	if child.nextPexpr != nil {
		child.nextPexpr.prevPexpr = child.prevPexpr
	} else {
		p.lastChildPexpr = child.prevPexpr
	}

	child.parentPexpr = nil
	child.nextPexpr = nil
	child.prevPexpr = nil
}

// FirstChildPexpr returns the first child, or nil.
func (p *Pexpr) FirstChildPexpr() *Pexpr {
	return p.firstChildPexpr
}

// ChildPexprs returns a slice of all child pexprs for iteration.
func (p *Pexpr) ChildPexprs() []*Pexpr {
	var children []*Pexpr
	for child := p.firstChildPexpr; child != nil; child = child.nextPexpr {
		children = append(children, child)
	}
	return children
}

// ============================================================================
// Methods for first set computation
// ============================================================================

// FindFirstSet computes the first set of tokens that could start this expression.
// It updates firstKeywords and firstTokens arrays.
func (p *Pexpr) FindFirstSet(firstKeywords []bool, firstTokens []bool) {
	switch p.Type {
	case PexprTypeNonterm:
		// The first set of a nonterminal is the first set of its rule
		rule := p.NontermRule
		if rule != nil && !rule.FirstSetFound {
			rule.FindFirstSet()
		}
		if rule != nil {
			// Merge the rule's first sets
			for i, v := range rule.FirstKeywords {
				if i < len(firstKeywords) {
					firstKeywords[i] = firstKeywords[i] || v
				}
			}
			for i, v := range rule.FirstTokens {
				if i < len(firstTokens) {
					firstTokens[i] = firstTokens[i] || v
				}
			}
			p.CanBeEmpty = rule.CanBeEmpty
		}

	case PexprTypeTerm:
		// A term contributes its token type to the first set
		if uint32(p.TokenType) < uint32(len(firstTokens)) {
			firstTokens[uint32(p.TokenType)] = true
		}

	case PexprTypeKeyword:
		// A keyword contributes its keyword number to the first set
		if p.Keyword != nil && p.Keyword.Num < uint32(len(firstKeywords)) {
			firstKeywords[p.Keyword.Num] = true
		}

	case PexprTypeEmpty, PexprTypeAnd, PexprTypeNot:
		// These can all match empty input
		p.CanBeEmpty = true

	case PexprTypeSequence:
		// For sequence, compute first set of each element until we find one that can't be empty
		for _, child := range p.ChildPexprs() {
			child.FindFirstSet(firstKeywords, firstTokens)
			if !child.CanBeEmpty {
				return
			}
		}
		p.CanBeEmpty = true

	case PexprTypeChoice:
		// For choice, compute first set of all alternatives
		for _, child := range p.ChildPexprs() {
			child.FindFirstSet(firstKeywords, firstTokens)
			if child.CanBeEmpty {
				p.CanBeEmpty = true
			}
		}

	case PexprTypeZeroOrMore, PexprTypeOptional:
		// These can always match empty
		p.CanBeEmpty = true
		if p.firstChildPexpr != nil {
			p.firstChildPexpr.FindFirstSet(firstKeywords, firstTokens)
		}

	case PexprTypeOneOrMore:
		// OneOrMore: can be empty only if child can be empty
		if p.firstChildPexpr != nil {
			child := p.firstChildPexpr
			child.FindFirstSet(firstKeywords, firstTokens)
			p.CanBeEmpty = child.CanBeEmpty
		}
	}
}

// ============================================================================
// String representation
// ============================================================================

// RawToString returns the string representation without parentheses.
func (p *Pexpr) RawToString() string {
	switch p.Type {
	case PexprTypeNonterm:
		if p.Sym != nil {
			return p.Sym.Name
		}
		return "?"

	case PexprTypeTerm:
		if p.Sym != nil {
			return p.Sym.Name
		}
		return fmt.Sprintf("TokenType(%d)", p.TokenType)

	case PexprTypeEmpty:
		return "EMPTY"

	case PexprTypeKeyword:
		if p.Sym != nil {
			return fmt.Sprintf(`"%s"`, p.Sym.Name)
		}
		return `"?"`

	case PexprTypeSequence:
		s := ""
		firstTime := true
		for _, child := range p.ChildPexprs() {
			// Skip EOF tokens in sequence strings
			if child.Type == PexprTypeTerm && child.TokenType == TokenTypeEof {
				continue
			}
			if !firstTime {
				s += " "
			}
			firstTime = false
			s += child.ToString()
		}
		return s

	case PexprTypeChoice:
		s := ""
		firstTime := true
		for _, child := range p.ChildPexprs() {
			if !firstTime {
				s += " | "
			}
			firstTime = false
			s += child.ToString()
		}
		return s

	case PexprTypeZeroOrMore:
		if p.firstChildPexpr != nil {
			return p.firstChildPexpr.ToString() + "*"
		}
		return "*"

	case PexprTypeOneOrMore:
		if p.firstChildPexpr != nil {
			return p.firstChildPexpr.ToString() + "+"
		}
		return "+"

	case PexprTypeOptional:
		if p.firstChildPexpr != nil {
			return p.firstChildPexpr.ToString() + "?"
		}
		return "?"

	case PexprTypeAnd:
		if p.firstChildPexpr != nil {
			return "&" + p.firstChildPexpr.ToString()
		}
		return "&"

	case PexprTypeNot:
		if p.firstChildPexpr != nil {
			return "!" + p.firstChildPexpr.ToString()
		}
		return "!"

	default:
		return fmt.Sprintf("UnknownType(%d)", p.Type)
	}
}

// ToString returns the string representation of this expression, including parentheses if needed.
func (p *Pexpr) ToString() string {
	s := p.RawToString()
	if !p.HasParens {
		return s
	}
	return "(" + s + ")"
}

// Dump outputs debugging information about this expression.
func (p *Pexpr) Dump() {
	fmt.Println(p.ToString())
}
