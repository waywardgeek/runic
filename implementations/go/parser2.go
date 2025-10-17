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

// ============================================================================
// MAIN ENTRY POINT: Parse grammar rules from .syn file
// ============================================================================

// ParseRules parses all rules from the syntax file.
// This is Phase 2 implementation of the recursive descent parser for .syn files.
func (p *Peg) ParseRules() error {
	if p.lexer == nil {
		return fmt.Errorf("ParseRules: no lexer available")
	}

	p.lexer.EnableWeakStrings(true)

	for !p.lexer.Eof() {
		err := p.parseRule()
		if err != nil {
			// Check if error is due to EOF - if so, we're done
			if p.lexer.Eof() {
				break
			}
			return err
		}
	}

	// Assign keyword numbers
	p.numKeywords = p.Keytab.SetKeywordNums()

	// Bind nonterminals to rules
	if !p.bindNonterms() {
		return fmt.Errorf("ParseRules: failed to bind nonterminals")
	}

	// Check for unused rules
	if !p.checkForUnusedRules() {
		return fmt.Errorf("ParseRules: unused rules detected")
	}

	// Find first sets for all rules (includes left-recursion detection)
	p.findFirstSets()

	return nil
}

// ============================================================================
// parseRule - Parse a single rule: name := pexpr ;
// ============================================================================

func (p *Peg) parseRule() error {
	// Parse identifier (rule name)
	identToken, err := p.parseIdent()
	if err != nil {
		return err
	}

	// Parse ':' or ':='
	token, err := p.parseToken()
	if err != nil {
		return err
	}

	if token.Type != TokenTypeKeyword {
		return fmt.Errorf("parseRule: expected ':' or ':=', got %v at line %d", token.Type, token.Location.Line)
	}

	keyword := token.Keyword
	if keyword != p.kwColon && keyword != p.kwColonEquals {
		return fmt.Errorf("parseRule: expected ':' or ':=', got %s at line %d", keyword.Sym.Name, token.Location.Line)
	}

	isWeak := keyword == p.kwColon

	// Parse parsing expression
	pexpr, err := p.parsePexpr()
	if err != nil {
		return err
	}

	// Verify we're at end of rule
	if !p.endOfRule() {
		return fmt.Errorf("parseRule: unexpected token at end of rule")
	}

	// Create the rule and add it
	sym := identToken.Value.Val.(*Sym)
	rule := NewRule(p, sym, pexpr, identToken.Location)
	rule.Weak = isWeak

	// Add to Peg (both hashed and ordered)
	p.InsertRule(rule)
	p.AppendOrderedRule(rule)

	return nil
}

// ============================================================================
// parsePexpr - Top-level expression dispatcher
// ============================================================================

func (p *Peg) parsePexpr() (*Pexpr, error) {
	return p.parseChoicePexpr()
}

// ============================================================================
// parseChoicePexpr - Parse choice: e1 | e2 | e3
// ============================================================================

func (p *Peg) parseChoicePexpr() (*Pexpr, error) {
	var choicePexpr *Pexpr

	for {
		pexpr, err := p.parseSequencePexpr()
		if err != nil {
			return nil, err
		}

		nextToken, err := p.peekToken(1)
		if err != nil {
			return nil, err
		}

		// Check if next is pipe (|)
		if nextToken.Type != TokenTypeKeyword || nextToken.Keyword != p.kwPipe {
			// Not a choice, return single expression
			if choicePexpr == nil {
				return pexpr, nil
			}
			choicePexpr.AppendChildPexpr(pexpr)
			return choicePexpr, nil
		}

		// Create choice if first time
		if choicePexpr == nil {
			choicePexpr = NewPexpr(PexprTypeChoice, pexpr.Location)
		}
		choicePexpr.AppendChildPexpr(pexpr)

		// Consume the pipe
		if _, err := p.parseToken(); err != nil {
			return nil, err
		}
	}
}

// ============================================================================
// parseSequencePexpr - Parse sequence: e1 e2 e3
// ============================================================================

func (p *Peg) parseSequencePexpr() (*Pexpr, error) {
	pexpr, err := p.parsePrefixPexpr()
	if err != nil {
		return nil, err
	}

	// Check for end of sequence
	if p.endOfRule() || p.endOfSequence() {
		return pexpr, nil
	}

	// Multiple items - create sequence
	sequencePexpr := NewPexpr(PexprTypeSequence, pexpr.Location)
	sequencePexpr.AppendChildPexpr(pexpr)

	for !p.endOfSequence() {
		pexpr, err := p.parsePrefixPexpr()
		if err != nil {
			return nil, err
		}
		sequencePexpr.AppendChildPexpr(pexpr)
	}

	return sequencePexpr, nil
}

// endOfSequence checks if we've reached the end of a sequence.
func (p *Peg) endOfSequence() bool {
	if p.endOfRule() {
		return true
	}

	token, err := p.peekToken(1)
	if err != nil {
		return false  // Changed from true - unmatchable token should not end sequence
	}

	switch token.Type {
	case TokenTypeKeyword:
		keyword := token.Keyword
		// End of sequence at | (pipe) or ) (close paren)
		return keyword == p.kwPipe || keyword == p.kwCloseParen
	case TokenTypeIdent, TokenTypeString, TokenTypeWeakString:
		return false
	case TokenTypeEof:
		return true
	}
	// Implicitly return false for any unhandled token types (like INTEGER, FLOAT, etc.)
	// This matches the Rune code which has no default case
	return false
}

// ============================================================================
// parsePrefixPexpr - Parse prefix operators: & (and) and ! (not)
// ============================================================================

func (p *Peg) parsePrefixPexpr() (*Pexpr, error) {
	token, err := p.peekToken(1)
	if err != nil {
		return nil, err
	}

	if token.Type == TokenTypeKeyword {
		keyword := token.Keyword
		if keyword == p.kwAnd || keyword == p.kwNot {
			// Consume the operator
			if _, err := p.parseToken(); err != nil {
				return nil, err
			}

			// Parse the operand
			pexpr, err := p.parsePostfixPexpr()
			if err != nil {
				return nil, err
			}

			// Create unary expression
			if keyword == p.kwAnd {
				return p.unaryPexpr(PexprTypeAnd, pexpr, token.Location), nil
			}
			return p.unaryPexpr(PexprTypeNot, pexpr, token.Location), nil
		}
	}

	return p.parsePostfixPexpr()
}

// ============================================================================
// parsePostfixPexpr - Parse postfix operators: * + ?
// ============================================================================

func (p *Peg) parsePostfixPexpr() (*Pexpr, error) {
	pexpr, err := p.parseBasicPexpr()
	if err != nil {
		return nil, err
	}

	if p.endOfRule() {
		return pexpr, nil
	}

	token, err := p.peekToken(1)
	if err != nil {
		return nil, err
	}

	if token.Type == TokenTypeKeyword {
		keyword := token.Keyword
		if keyword == p.kwQuestion {
			if _, err := p.parseToken(); err != nil {
				return nil, err
			}
			return p.unaryPexpr(PexprTypeOptional, pexpr, token.Location), nil
		} else if keyword == p.kwStar {
			if _, err := p.parseToken(); err != nil {
				return nil, err
			}
			return p.unaryPexpr(PexprTypeZeroOrMore, pexpr, token.Location), nil
		} else if keyword == p.kwPlus {
			if _, err := p.parseToken(); err != nil {
				return nil, err
			}
			return p.unaryPexpr(PexprTypeOneOrMore, pexpr, token.Location), nil
		}
	}

	return pexpr, nil
}

// ============================================================================
// parseBasicPexpr - Parse basic items: identifiers, keywords, groups
// ============================================================================

func (p *Peg) parseBasicPexpr() (*Pexpr, error) {
	token, err := p.parseToken()
	if err != nil {
		return nil, err
	}

	switch token.Type {
	case TokenTypeIdent:
		// Nonterminal reference
		pexpr := NewPexpr(PexprTypeNonterm, token.Location)
		if val, ok := token.Value.Val.(*Sym); ok {
			pexpr.Sym = val
		}
		return pexpr, nil

	case TokenTypeString, TokenTypeWeakString:
		// Keyword in quotes
		pexpr := NewPexpr(PexprTypeKeyword, token.Location)
		if str, ok := token.Value.Val.(string); ok {
			pexpr.Sym = NewSym(str)
			pexpr.Weak = token.Type == TokenTypeWeakString

			// Register keyword in keytab and link to pexpr
			keyword := p.Keytab.New(str)
			keyword.AppendPexpr(pexpr)
			pexpr.Keyword = keyword
		}
		return pexpr, nil

	case TokenTypeKeyword:
		keyword := token.Keyword

		// Check for special keywords
		if keyword == p.kwEmpty {
			return NewPexpr(PexprTypeEmpty, token.Location), nil
		}

		if keyword == p.kwOpenParen {
			return p.parseParenPexpr()
		}

		// Terminal token type (INTEGER, IDENT, FLOAT, etc.)
		pexpr := NewPexpr(PexprTypeTerm, token.Location)
		tokenType, err := p.keywordToTokenType(keyword, token.Location)
		if err != nil {
			return nil, err
		}
		pexpr.TokenType = tokenType
		pexpr.Sym = keyword.Sym
		return pexpr, nil

	default:
		return nil, fmt.Errorf("parseBasicPexpr: unexpected token type %v at line %d", token.Type, token.Location.Line)
	}
}

// ============================================================================
// parseParenPexpr - Parse parenthesized expression
// ============================================================================

func (p *Peg) parseParenPexpr() (*Pexpr, error) {
	pexpr, err := p.parsePexpr()
	if err != nil {
		return nil, err
	}

	token, err := p.parseToken()
	if err != nil {
		return nil, err
	}

	if token.Type != TokenTypeKeyword || token.Keyword != p.kwCloseParen {
		return nil, fmt.Errorf("parseParenPexpr: expected ')', got %s at line %d", token.GetName(), token.Location.Line)
	}

	pexpr.HasParens = true
	return pexpr, nil
}

// ============================================================================
// Token reading with lookahead
// ============================================================================

// parseToken reads and returns the next token.
func (p *Peg) parseToken() (*Token, error) {
	// Check lookahead buffer first
	if p.savedToken1 != nil {
		token := p.savedToken1
		p.savedToken1 = p.savedToken2
		p.savedToken2 = nil
		return token, nil
	}

	return p.rawParseToken()
}

// rawParseToken reads from lexer, skipping newlines.
func (p *Peg) rawParseToken() (*Token, error) {
	for {
		token, err := p.lexer.ParseToken()
		if err != nil {
			return nil, err
		}

		// Skip newline tokens
		if token.Type == TokenTypeKeyword && token.Keyword == p.kwNewline {
			continue
		}

		return token, nil
	}
}

// peekToken looks ahead 1 or 2 tokens without consuming them.
func (p *Peg) peekToken(depth int) (*Token, error) {
	if depth < 1 || depth > 2 {
		return nil, fmt.Errorf("peekToken: depth must be 1 or 2")
	}

	if depth >= 1 && p.savedToken1 == nil {
		token, err := p.rawParseToken()
		if err != nil {
			return nil, err
		}
		p.savedToken1 = token
	}

	if depth >= 2 && p.savedToken2 == nil {
		token, err := p.rawParseToken()
		if err != nil {
			return nil, err
		}
		p.savedToken2 = token
	}

	if depth == 1 {
		return p.savedToken1, nil
	}
	return p.savedToken2, nil
}

// ============================================================================
// Helper methods
// ============================================================================

// parseIdent reads and returns an identifier token.
func (p *Peg) parseIdent() (*Token, error) {
	token, err := p.parseToken()
	if err != nil {
		return nil, err
	}

	if token.Type != TokenTypeIdent {
		return nil, fmt.Errorf("parseIdent: expected identifier, got %v at line %d", token.Type, token.Location.Line)
	}

	return token, nil
}

// unaryPexpr creates a unary expression node.
func (p *Peg) unaryPexpr(pexprType PexprType, child *Pexpr, location Location) *Pexpr {
	parent := NewPexpr(pexprType, location)
	parent.InsertChildPexpr(child)
	return parent
}

// keywordToTokenType maps PEG keywords to TokenTypes.
func (p *Peg) keywordToTokenType(keyword *Keyword, location Location) (TokenType, error) {
	switch keyword {
	case p.kwEof:
		return TokenTypeEof, nil
	case p.kwIdent:
		return TokenTypeIdent, nil
	case p.kwInteger:
		return TokenTypeInteger, nil
	case p.kwFloat:
		return TokenTypeFloat, nil
	case p.kwString:
		return TokenTypeString, nil
	case p.kwRandInt:
		return TokenTypeRandUint, nil
	case p.kwIntType:
		return TokenTypeIntType, nil
	case p.kwUintType:
		return TokenTypeUintType, nil
	default:
		return TokenTypeKeyword, fmt.Errorf("keywordToTokenType: unknown keyword %s", keyword.Sym.Name)
	}
}

// endOfRule checks if we're at the end of a rule definition.
// End of rule is marked by seeing ':' or ':=' at lookahead(2), or being at logical EOF.
func (p *Peg) endOfRule() bool {
	// Check logical EOF: lexer at EOF AND no buffered tokens
	if p.lexer.Eof() && p.savedToken1 == nil && p.savedToken2 == nil {
		return true
	}

	token, err := p.peekToken(2)
	if err != nil {
		// Error peeking - treat as end of rule
		return true
	}

	// If peek(2) is EOF, we might be at end of rule
	// But only if peek(1) is ALSO EOF or a rule-ending token
	// Otherwise there's still content to parse in this rule
	if token.Type == TokenTypeEof {
		// Check what peek(1) is
		token1, _ := p.peekToken(1)
		if token1 == nil || token1.Type == TokenTypeEof {
			return true
		}
		// There's a valid token at peek(1), so not end of rule yet
		return false
	}

	// ':' or ':=' at lookahead(2) means the next rule is starting
	if token.Type != TokenTypeKeyword {
		return false
	}

	return token.Keyword == p.kwColon || token.Keyword == p.kwColonEquals
}

// ============================================================================
// Bind nonterminals to their rules
// ============================================================================

// bindNonterms links all nonterminal references in expressions to their Rule objects.
func (p *Peg) bindNonterms() bool {
	passed := true

	for _, rule := range p.OrderedRules() {
		if rule.pexpr != nil {
			if !p.bindPexprNonterms(rule.pexpr) {
				passed = false
			}
		}
	}

	return passed
}

// bindPexprNonterms recursively binds nonterminals in a Pexpr tree.
func (p *Peg) bindPexprNonterms(pexpr *Pexpr) bool {
	if pexpr == nil {
		return true
	}

	passed := true

	// If this is a nonterminal reference, bind it to its rule
	if pexpr.Type == PexprTypeNonterm {
		rule := p.FindRule(pexpr.Sym)
		if rule == nil {
			fmt.Printf("Error: undefined rule '%s' at line %d\n", pexpr.Sym.Name, pexpr.Location.Line)
			passed = false
		} else {
			pexpr.NontermRule = rule
			rule.AppendNontermPexpr(pexpr)
		}
	}

	// Recursively bind children
	for _, child := range pexpr.ChildPexprs() {
		if !p.bindPexprNonterms(child) {
			passed = false
		}
	}

	return passed
}

// ============================================================================
// Find first sets for all rules
// ============================================================================

// findFirstSets computes the first token sets for all rules.
// This detects left-recursion.
func (p *Peg) findFirstSets() {
	for _, rule := range p.OrderedRules() {
		if !rule.FirstSetFound {
			rule.FindFirstSet()
		}
	}
}

// ============================================================================
// Check for unused rules
// ============================================================================

// checkForUnusedRules reports rules that are never referenced.
func (p *Peg) checkForUnusedRules() bool {
	passed := true
	firstTime := true

	for _, rule := range p.OrderedRules() {
		if !firstTime {
			// Check if rule is referenced as a nonterminal
			if rule.firstNontermPexpr == nil {
				fmt.Printf("Warning: unused rule '%s' at line %d\n", rule.Sym.Name, rule.Location.Line)
				// Don't fail on unused rules - just warn
			}
		}
		firstTime = false
	}

	return passed
}
