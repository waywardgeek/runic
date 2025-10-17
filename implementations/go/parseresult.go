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

// Match represents the result of a parsing attempt.
type Match struct {
	Success bool
	Pos     uint32
}

// NewMatch creates a new Match result.
func NewMatch(success bool, pos uint32) Match {
	return Match{
		Success: success,
		Pos:     pos,
	}
}

// ParseResult represents the result of parsing a rule at a specific position.
type ParseResult struct {
	Rule              *Rule
	Pos               uint32 // Position where this parse attempt started
	Result            Match  // The result of parsing
	FoundRecursion    bool   // Whether left-recursion was detected
	Pending           bool   // Whether this is in-progress (for left-recursion detection)

	// OneToOne ParseResult Node cascade
	node *Node

	// DoublyLinked Rule ParseResult cascade
	ruleParent              *Rule
	prevRuleParseResult     *ParseResult
	nextRuleParseResult     *ParseResult

	// Hashed Rule:"Hashed" ParseResult:"Hashed" cascade ("pos")
	nextHashedRuleParseResult *ParseResult

	// DoublyLinked Lexer ParseResult cascade
	lexer                     *Lexer
	prevLexerParseResult      *ParseResult
	nextLexerParseResult      *ParseResult

	// DoublyLinked ParseResult:"Parent" ParseResult:"Child"
	parentParseResult         *ParseResult
	firstChildParseResult     *ParseResult
	lastChildParseResult      *ParseResult
	prevChildParseResult      *ParseResult
	nextChildParseResult      *ParseResult

	// For collecting tokens/parse tree building
	lastChildParseResultSnapshot *ParseResult
}

// NewParseResult creates a new ParseResult.
func NewParseResult(parentParseResult *ParseResult, rule *Rule, pos uint32, result Match) *ParseResult {
	pr := &ParseResult{
		Rule:              rule,
		Pos:               pos,
		Result:            result,
		FoundRecursion:    false,
		Pending:           false,
		parentParseResult: parentParseResult,
		node:              nil,
	}

	// Add to rule's hashed table and doubly-linked list
	rule.InsertHashedParseResult(pr)
	rule.AppendParseResult(pr)

	// Add to parent if provided
	if parentParseResult != nil {
		parentParseResult.AppendChildParseResult(pr)
	}

	// Add to lexer so we can access parse results later
	if rule.peg != nil && rule.peg.lexer != nil {
		lexer := rule.peg.lexer
		lexer.AppendParseResult(pr)
	}

	// Record what the last child was before this append
	pr.lastChildParseResultSnapshot = nil
	if parentParseResult != nil {
		if parentParseResult.lastChildParseResult != pr {
			pr.lastChildParseResultSnapshot = parentParseResult.lastChildParseResult
		}
	}

	return pr
}

// ============================================================================
// DoublyLinked Rule ParseResult cascade
// ============================================================================

// RuleParent returns the parent Rule.
func (pr *ParseResult) RuleParent() *Rule {
	return pr.ruleParent
}

// ============================================================================
// DoublyLinked Lexer ParseResult cascade
// ============================================================================

// SetLexer sets the Lexer parent for this ParseResult.
func (pr *ParseResult) SetLexer(lexer *Lexer) {
	if lexer == nil {
		return
	}
	pr.lexer = lexer
	lexer.AppendParseResult(pr)
}

// Lexer returns the parent Lexer.
func (pr *ParseResult) Lexer() *Lexer {
	return pr.lexer
}

// ============================================================================
// DoublyLinked ParseResult:"Parent" ParseResult:"Child" cascade
// ============================================================================

// AppendChildParseResult adds a child ParseResult (DoublyLinked).
func (pr *ParseResult) AppendChildParseResult(child *ParseResult) {
	if child == nil {
		return
	}

	if pr.lastChildParseResult == nil {
		pr.firstChildParseResult = child
	} else {
		pr.lastChildParseResult.nextChildParseResult = child
		child.prevChildParseResult = pr.lastChildParseResult
	}
	pr.lastChildParseResult = child
	child.parentParseResult = pr
	child.nextChildParseResult = nil
}

// RemoveChildParseResult removes a child ParseResult.
func (pr *ParseResult) RemoveChildParseResult(child *ParseResult) {
	if child == nil || child.parentParseResult != pr {
		return
	}

	if child.prevChildParseResult != nil {
		child.prevChildParseResult.nextChildParseResult = child.nextChildParseResult
	} else {
		pr.firstChildParseResult = child.nextChildParseResult
	}

	if child.nextChildParseResult != nil {
		child.nextChildParseResult.prevChildParseResult = child.prevChildParseResult
	} else {
		pr.lastChildParseResult = child.prevChildParseResult
	}

	child.prevChildParseResult = nil
	child.nextChildParseResult = nil
	child.parentParseResult = nil
}

// LastChildParseResult returns the last child ParseResult.
func (pr *ParseResult) LastChildParseResult() *ParseResult {
	return pr.lastChildParseResult
}

// ChildParseResults returns a slice of all child ParseResults.
func (pr *ParseResult) ChildParseResults() []*ParseResult {
	var children []*ParseResult
	for child := pr.firstChildParseResult; child != nil; child = child.nextChildParseResult {
		children = append(children, child)
	}
	return children
}

// SafeChildParseResults returns a slice of all child ParseResults (safe during modification).
func (pr *ParseResult) SafeChildParseResults() []*ParseResult {
	var children []*ParseResult
	child := pr.firstChildParseResult
	for child != nil {
		next := child.nextChildParseResult
		children = append(children, child)
		child = next
	}
	return children
}

// ============================================================================
// OneToOne ParseResult Node cascade
// ============================================================================

// InsertNode sets the Node for this ParseResult (OneToOne cascade).
func (pr *ParseResult) InsertNode(node *Node) {
	if node == nil {
		return
	}
	pr.node = node
	node.ParseResult = pr
}

// Node returns the associated Node.
func (pr *ParseResult) Node() *Node {
	return pr.node
}

// ============================================================================
// Helper methods for parse tree building
// ============================================================================

// BuildParseTree constructs an AST Node from this ParseResult.
func (pr *ParseResult) BuildParseTree(simplify bool) *Node {
	var parentNode *Node
	if pr.parentParseResult != nil {
		parentNode = pr.parentParseResult.Node()
	}

	// Create the Node for this ParseResult
	node := NewNode(parentNode, pr, pr.Pos, pr.Result.Pos)
	pr.InsertNode(node)

	// Add tokens from this parse result's range
	pos := pr.Pos
	for _, child := range pr.ChildParseResults() {
		// Add any tokens between current pos and child's start
		pr.addNodeTokens(node, pos, child.Pos)
		child.BuildParseTree(simplify)
		pos = child.Result.Pos
	}
	// Add remaining tokens
	pr.addNodeTokens(node, pos, pr.Result.Pos)

	// Simplify the node tree if requested
	if simplify {
		node.Simplify()
	}

	return node
}

// addNodeTokens adds tokens in the given range to the node.
func (pr *ParseResult) addNodeTokens(node *Node, startPos uint32, endPos uint32) {
	if pr.lexer == nil {
		return
	}

	for pos := startPos; pos < endPos && pos < uint32(len(pr.lexer.Tokens)); pos++ {
		token := pr.lexer.Tokens[pos]
		if token.Pexpr != nil {
			pexpr := token.Pexpr.(*Pexpr)
			if !pexpr.Weak {
				NewNode(node, nil, pos, pos+1).SetToken(token)
			}
		}
	}
}

// ============================================================================
// String representation
// ============================================================================

// Dump outputs debugging information about this ParseResult.
func (pr *ParseResult) Dump() {
	pr.DumpIndented(0)
}

// DumpIndented outputs indented debugging information.
func (pr *ParseResult) DumpIndented(depth uint32) {
	indent := ""
	for i := uint32(0); i < depth*2; i++ {
		indent += " "
	}

	if pr.Rule != nil {
		fmt.Printf("%s%s <%p>\n", indent, pr.Rule.Sym.Name, pr)
	} else {
		fmt.Printf("%s<unknown> <%p>\n", indent, pr)
	}

	for _, child := range pr.ChildParseResults() {
		child.DumpIndented(depth + 1)
	}
}

// ToString returns a string representation of this ParseResult.
func (pr *ParseResult) ToString() string {
	if pr.Rule != nil {
		return pr.Rule.Sym.Name
	}
	return fmt.Sprintf("<ParseResult at %d>", pr.Pos)
}
