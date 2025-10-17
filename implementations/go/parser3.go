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
)

// ============================================================================
// Phase 3: PEG Engine - Parse input files using the grammar
// ============================================================================

// Parse parses an input file using the PEG grammar rules.
// fileSpec can be a string (filename) or a *Filepath.
// allowUnderscores determines if identifiers can contain underscores.
func (p *Peg) Parse(fileSpec interface{}, allowUnderscores bool) (*Node, error) {
	// Initialize on first parse
	if !p.initialized {
		p.addEOFToFirstRule()
		p.initialized = true
	}

	// Clear lookahead buffer
	p.savedToken1 = nil
	p.savedToken2 = nil

	// Create filepath from input
	var filepath *Filepath
	switch v := fileSpec.(type) {
	case string:
		filepath = NewFilepath(v, nil, false)
	case *Filepath:
		filepath = v
	default:
		return nil, fmt.Errorf("Parse: fileSpec must be string or *Filepath")
	}

	// Determine if we need to read the file
	needRead := filepath.Text == ""

	// Create new lexer for input file
	lexer, err := NewLexer(filepath, p.Keytab, needRead)
	if err != nil {
		return nil, err
	}
	lexer.AllowIdentUnderscores = allowUnderscores

	// Replace lexer if we had one
	if p.lexer != nil {
		// TODO: cleanup old lexer if needed
	}
	p.lexer = lexer

	// Tokenize entire input upfront
	p.tokenizeInput()

	// Clear memoization caches from previous parses
	for _, rule := range p.OrderedRules() {
		rule.ClearHashedParseResults()
		rule.ClearParseResults()
	}

	// Start parsing from first rule
	rule := p.firstOrderedRule
	if rule == nil {
		return nil, fmt.Errorf("Parse: no rules defined")
	}

	result := p.parseUsingRule(nil, rule, 0)
	if !result.Success {
		// Find where we got stuck
		pos := p.maxTokenPos
		if int(pos) >= len(p.lexer.Tokens) {
			pos = uint32(len(p.lexer.Tokens) - 1)
		}
		token := p.lexer.Tokens[pos]
		return nil, fmt.Errorf("Syntax error at line %d", token.Location.Line)
	}

	// Build parse tree from first ParseResult
	if len(p.lexer.ParseResults) == 0 {
		return nil, fmt.Errorf("Parse: no parse results generated")
	}
	parseResult := p.lexer.ParseResults[0]
	node := parseResult.BuildParseTree(p.simplifyNodes)

	return node, nil
}

// tokenizeInput reads all tokens from the lexer into an array.
func (p *Peg) tokenizeInput() {
	// Clear any existing tokens
	p.lexer.Tokens = make([]*Token, 0)
	
	for {
		token, err := p.lexer.ParseToken()
		if err != nil {
			// On error, add an EOF token and stop
			token = p.lexer.EofToken()
			// Note: NewToken already calls lexer.AppendToken, so we don't need to call it again
			break
		}
		// Note: NewToken already appends the token to lexer.Tokens, so we don't call AppendToken here
		token.Pexpr = nil
		if token.IsEof() {
			break
		}
	}
}

// addEOFToFirstRule appends an EOF terminal to the first (goal) rule.
// This ensures the parser matches the entire input.
func (p *Peg) addEOFToFirstRule() {
	goal := p.firstOrderedRule
	if goal == nil {
		return
	}

	pexpr := goal.pexpr
	if pexpr == nil {
		return
	}

	// If the goal rule isn't a sequence, make it one
	if pexpr.Type != PexprTypeSequence {
		goal.RemovePexpr(pexpr)
		seqPexpr := NewPexpr(PexprTypeSequence, pexpr.Location)
		seqPexpr.InsertChildPexpr(pexpr)
		goal.InsertPexpr(seqPexpr)
		pexpr = seqPexpr
	}

	// Add EOF terminal to end of sequence
	eofPexpr := NewPexpr(PexprTypeTerm, pexpr.Location)
	eofPexpr.TokenType = TokenTypeEof
	eofPexpr.Sym = p.kwEof.Sym
	pexpr.AppendChildPexpr(eofPexpr)
}

// ============================================================================
// parseUsingRule - Parse using a specific rule with memoization
// ============================================================================

// parseUsingRule attempts to parse input at position pos using the given rule.
// Implements packrat parsing with memoization and handles left-recursion.
func (p *Peg) parseUsingRule(parentParseResult *ParseResult, rule *Rule, pos uint32) Match {
	// Check memoization table
	parseResult := rule.FindHashedParseResult(pos)
	if parseResult != nil {
		// Found cached result
		if parseResult.Pending {
			// Detected left-recursion
			parseResult.FoundRecursion = true
		} else if parseResult.Result.Success && parentParseResult != nil && parseResult.parentParseResult == nil {
			// Re-attach successful result to new parent
			parentParseResult.AppendChildParseResult(parseResult)
		}
		return parseResult.Result
	}

	// Check first-set optimization
	if int(pos) < len(p.lexer.Tokens) {
		token := p.lexer.Tokens[pos]
		if token.Type == TokenTypeKeyword {
			if int(token.Keyword.Num) < len(rule.FirstKeywords) && !rule.FirstKeywords[token.Keyword.Num] {
				// Token not in first set
				return Match{Success: rule.CanBeEmpty, Pos: pos}
			}
		} else {
			if int(token.Type) < len(rule.FirstTokens) && !rule.FirstTokens[int(token.Type)] {
				// Token type not in first set
				return Match{Success: rule.CanBeEmpty, Pos: pos}
			}
		}
	}

	// Use the "seed" approach for left-recursion handling
	// Initialize with failure result
	pres := NewParseResult(parentParseResult, rule, pos, Match{Success: false, Pos: pos})
	// Note: NewParseResult already adds to rule's hash table and lexer

	lastResult := Match{Success: false, Pos: pos}

	// Try parsing repeatedly until no more progress
	for {
		pres.Pending = true
		result := p.parseUsingPexpr(pres, rule.pexpr, pos)
		pres.Pending = false

		madeProgress := false
		if result.Success && result.Pos > lastResult.Pos {
			madeProgress = true
			lastResult = result
			pres.Result = lastResult

			if pres.FoundRecursion {
				// Push recursive result
				pres = p.pushRecursiveParseResult(pres, rule)
			}
		}

		if !madeProgress || !pres.FoundRecursion {
			break
		}
	}

	return lastResult
}

// pushRecursiveParseResult creates a new ParseResult to hold recursive match info.
func (p *Peg) pushRecursiveParseResult(pres *ParseResult, rule *Rule) *ParseResult {
	rule.RemoveHashedParseResult(pres)
	parent := pres.parentParseResult
	if parent != nil {
		parent.RemoveChildParseResult(pres)
	}

	// Create new ParseResult with same result
	// Note: NewParseResult adds to rule's hash table and lexer automatically
	result := pres.Result
	newPres := NewParseResult(parent, rule, pres.Pos, result)
	newPres.FoundRecursion = pres.FoundRecursion
	newPres.Pending = pres.Pending
	newPres.AppendChildParseResult(pres)

	return newPres
}

// ============================================================================
// parseUsingPexpr - Wrapper that tracks maxTokenPos and prunes failures
// ============================================================================

// parseUsingPexpr parses using a pexpr, tracking progress and pruning failures.
func (p *Peg) parseUsingPexpr(parseResult *ParseResult, pexpr *Pexpr, pos uint32) Match {
	lastChild := parseResult.lastChildParseResult
	result := p.parseUsingPexprImpl(parseResult, pexpr, pos)

	if result.Success && result.Pos > p.maxTokenPos {
		p.maxTokenPos = result.Pos
	}

	if !result.Success {
		// Prune any successful ParseResults that we built before failing
		for parseResult.lastChildParseResult != lastChild {
			child := parseResult.lastChildParseResult
			if child == nil {
				break
			}
			parseResult.RemoveChildParseResult(child)
		}
	}

	return result
}

// ============================================================================
// parseUsingPexprImpl - Dispatch by pexpr type
// ============================================================================

// parseUsingPexprImpl implements the actual matching logic for each pexpr type.
func (p *Peg) parseUsingPexprImpl(parseResult *ParseResult, pexpr *Pexpr, pos uint32) Match {
	if int(pos) >= len(p.lexer.Tokens) {
		return Match{Success: false, Pos: pos}
	}

	token := p.lexer.Tokens[pos]

	switch pexpr.Type {
	case PexprTypeNonterm:
		// Recurse into nonterminal rule
		if pexpr.NontermRule == nil {
			return Match{Success: false, Pos: pos}
		}
		return p.parseUsingRule(parseResult, pexpr.NontermRule, pos)

	case PexprTypeTerm:
		// Match terminal token type
		if token.Type != pexpr.TokenType {
			return Match{Success: false, Pos: pos}
		}
		token.Pexpr = pexpr
		return Match{Success: true, Pos: pos + 1}

	case PexprTypeKeyword:
		// Match specific keyword
		if token.Type != TokenTypeKeyword || token.Keyword != pexpr.Keyword {
			return Match{Success: false, Pos: pos}
		}
		token.Pexpr = pexpr
		return Match{Success: true, Pos: pos + 1}

	case PexprTypeEmpty:
		// Empty always succeeds
		return Match{Success: true, Pos: pos}

	case PexprTypeSequence:
		return p.parseUsingSequencePexpr(parseResult, pexpr, pos)

	case PexprTypeChoice:
		return p.parseUsingChoicePexpr(parseResult, pexpr, pos)

	case PexprTypeZeroOrMore:
		return p.parseUsingZeroOrMorePexpr(parseResult, pexpr, pos)

	case PexprTypeOneOrMore:
		return p.parseUsingOneOrMorePexpr(parseResult, pexpr, pos)

	case PexprTypeOptional:
		return p.parseUsingOptionalPexpr(parseResult, pexpr, pos)

	case PexprTypeAnd:
		return p.parseUsingAndPexpr(parseResult, pexpr, pos)

	case PexprTypeNot:
		return p.parseUsingNotPexpr(parseResult, pexpr, pos)

	default:
		return Match{Success: false, Pos: pos}
	}
}

// ============================================================================
// Pexpr type-specific parsing functions
// ============================================================================

// parseUsingSequencePexpr matches all children in sequence.
func (p *Peg) parseUsingSequencePexpr(parseResult *ParseResult, pexpr *Pexpr, pos uint32) Match {
	childPos := pos
	for _, child := range pexpr.ChildPexprs() {
		result := p.parseUsingPexpr(parseResult, child, childPos)
		if !result.Success {
			return Match{Success: false, Pos: pos}
		}
		childPos = result.Pos
		if int(childPos) >= len(p.lexer.Tokens) {
			return result
		}
	}
	return Match{Success: true, Pos: childPos}
}

// parseUsingChoicePexpr tries each alternative until one succeeds.
func (p *Peg) parseUsingChoicePexpr(parseResult *ParseResult, pexpr *Pexpr, pos uint32) Match {
	for _, child := range pexpr.ChildPexprs() {
		result := p.parseUsingPexpr(parseResult, child, pos)
		if result.Success {
			return result
		}
	}
	return Match{Success: false, Pos: pos}
}

// parseUsingZeroOrMorePexpr matches the child zero or more times.
func (p *Peg) parseUsingZeroOrMorePexpr(parseResult *ParseResult, pexpr *Pexpr, pos uint32) Match {
	child := pexpr.FirstChildPexpr()
	if child == nil {
		return Match{Success: true, Pos: pos}
	}

	lastResult := Match{Success: true, Pos: pos}
	for {
		result := p.parseUsingPexpr(parseResult, child, lastResult.Pos)
		if !result.Success {
			break
		}
		lastResult = result
	}
	return lastResult
}

// parseUsingOneOrMorePexpr matches the child one or more times.
func (p *Peg) parseUsingOneOrMorePexpr(parseResult *ParseResult, pexpr *Pexpr, pos uint32) Match {
	child := pexpr.FirstChildPexpr()
	if child == nil {
		return Match{Success: false, Pos: pos}
	}

	lastResult := Match{Success: false, Pos: pos}
	for {
		result := p.parseUsingPexpr(parseResult, child, lastResult.Pos)
		if !result.Success {
			break
		}
		lastResult = result
	}
	return lastResult
}

// parseUsingOptionalPexpr tries to match the child, succeeding either way.
func (p *Peg) parseUsingOptionalPexpr(parseResult *ParseResult, pexpr *Pexpr, pos uint32) Match {
	child := pexpr.FirstChildPexpr()
	if child == nil {
		return Match{Success: true, Pos: pos}
	}

	result := p.parseUsingPexpr(parseResult, child, pos)
	if result.Success {
		return result
	}
	return Match{Success: true, Pos: pos}
}

// parseUsingAndPexpr implements positive lookahead (match but don't consume).
func (p *Peg) parseUsingAndPexpr(parseResult *ParseResult, pexpr *Pexpr, pos uint32) Match {
	child := pexpr.FirstChildPexpr()
	if child == nil {
		return Match{Success: false, Pos: pos}
	}

	result := p.parseUsingPexpr(parseResult, child, pos)
	// Return success/failure but keep position at pos (don't consume)
	return Match{Success: result.Success, Pos: pos}
}

// parseUsingNotPexpr implements negative lookahead (match if child fails).
func (p *Peg) parseUsingNotPexpr(parseResult *ParseResult, pexpr *Pexpr, pos uint32) Match {
	child := pexpr.FirstChildPexpr()
	if child == nil {
		return Match{Success: true, Pos: pos}
	}

	result := p.parseUsingPexpr(parseResult, child, pos)
	// Invert success and keep position at pos (don't consume)
	return Match{Success: !result.Success, Pos: pos}
}
