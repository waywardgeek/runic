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

// Node represents an AST (Abstract Syntax Tree) node, simplified from ParseResult.
type Node struct {
	ParseResult  *ParseResult // Reference to the ParseResult that generated this node
	StartPos     uint32       // Token position where this node starts
	EndPos       uint32       // Token position where this node ends
	Token        *Token       // If this node represents a single token
	Location     Location

	// DoublyLinked Node:"Parent" Node:"Child" cascade
	parent           *Node
	firstChildNode   *Node
	lastChildNode    *Node
	prevChildNode    *Node
	nextChildNode    *Node
}

// NewNode creates a new AST node.
func NewNode(parent *Node, parseResult *ParseResult, startPos uint32, endPos uint32) *Node {
	node := &Node{
		ParseResult: parseResult,
		StartPos:    startPos,
		EndPos:      endPos,
		Token:       nil,
		parent:      parent,
	}

	// Compute location from token positions
	node.computeLocation()

	// Add to parent if provided
	if parent != nil {
		parent.AppendChildNode(node)
	}

	// Add to parseResult if provided
	if parseResult != nil {
		parseResult.InsertNode(node)
	}

	return node
}

// NewNodeFromToken creates a Node from a single token.
func NewNodeFromToken(parent *Node, token *Token) *Node {
	node := &Node{
		ParseResult: nil,
		StartPos:    0,
		EndPos:      0,
		Token:       token,
		parent:      parent,
		Location:    token.Location,
	}

	if parent != nil {
		parent.AppendChildNode(node)
	}

	return node
}

// ============================================================================
// DoublyLinked Node:"Parent" Node:"Child" cascade
// ============================================================================

// AppendChildNode adds a child node (DoublyLinked).
func (n *Node) AppendChildNode(child *Node) {
	if child == nil {
		return
	}

	if n.lastChildNode == nil {
		n.firstChildNode = child
	} else {
		n.lastChildNode.nextChildNode = child
		child.prevChildNode = n.lastChildNode
	}
	n.lastChildNode = child
	child.parent = n
	child.nextChildNode = nil
}

// RemoveChildNode removes a child node.
func (n *Node) RemoveChildNode(child *Node) {
	if child == nil || child.parent != n {
		return
	}

	if child.prevChildNode != nil {
		child.prevChildNode.nextChildNode = child.nextChildNode
	} else {
		n.firstChildNode = child.nextChildNode
	}

	if child.nextChildNode != nil {
		child.nextChildNode.prevChildNode = child.prevChildNode
	} else {
		n.lastChildNode = child.prevChildNode
	}

	child.prevChildNode = nil
	child.nextChildNode = nil
	child.parent = nil
}

// InsertChildNode inserts a child at the beginning.
func (n *Node) InsertChildNode(child *Node) {
	if child == nil {
		return
	}

	if n.firstChildNode == nil {
		n.lastChildNode = child
	} else {
		n.firstChildNode.prevChildNode = child
		child.nextChildNode = n.firstChildNode
	}
	n.firstChildNode = child
	child.parent = n
	child.prevChildNode = nil
}

// FirstChildNode returns the first child node.
func (n *Node) FirstChildNode() *Node {
	return n.firstChildNode
}

// LastChildNode returns the last child node.
func (n *Node) LastChildNode() *Node {
	return n.lastChildNode
}

// ChildNodes returns a slice of all child nodes.
func (n *Node) ChildNodes() []*Node {
	var children []*Node
	for child := n.firstChildNode; child != nil; child = child.nextChildNode {
		children = append(children, child)
	}
	return children
}

// SafeChildNodes returns a slice of all child nodes (safe during modification).
func (n *Node) SafeChildNodes() []*Node {
	var children []*Node
	child := n.firstChildNode
	for child != nil {
		next := child.nextChildNode
		children = append(children, child)
		child = next
	}
	return children
}

// ============================================================================
// GetRuleSym returns the symbol for the rule this node matches, if any.
// ============================================================================

// GetRuleSym returns the rule symbol if this node represents a rule.
func (n *Node) GetRuleSym() *Sym {
	if n.ParseResult == nil || n.ParseResult.RuleParent() == nil {
		return nil
	}
	return n.ParseResult.RuleParent().Sym
}

// GetKeywordSym returns the keyword symbol if this node represents a keyword.
func (n *Node) GetKeywordSym() *Sym {
	if n.Token == nil || n.Token.Type != TokenTypeKeyword {
		return nil
	}
	if n.Token.Keyword == nil {
		return nil
	}
	return n.Token.Keyword.Sym
}

// GetIdentSym returns the identifier symbol if this node represents an identifier.
func (n *Node) GetIdentSym() *Sym {
	if n.Token == nil || n.Token.Type != TokenTypeIdent {
		return nil
	}
	if n.Token.Value.Val == nil {
		return nil
	}
	if sym, ok := n.Token.Value.Val.(*Sym); ok {
		return sym
	}
	return nil
}

// ============================================================================
// AST simplification
// ============================================================================

// Simplify simplifies the AST node by removing weak rules and merging single children.
func (n *Node) Simplify() {
	// First recursively simplify all children
	for _, child := range n.SafeChildNodes() {
		child.Simplify()
	}

	// Remove weak rules and tokens
	// Rune logic: if (isnull(rule) || rule.weak) && (isnull(token) || token.pexpr.weak)
	// BOTH conditions must be true to remove:
	// 1. Rule is null OR weak
	// 2. Token is null OR weak
	for _, child := range n.SafeChildNodes() {
		if n.firstChildNode == nil {
			break // Node already simplified away
		}

		if child.firstChildNode == nil {
			// Leaf node - check if it should be removed
			token := child.Token
			rule := (*Rule)(nil)
			if child.ParseResult != nil {
				rule = child.ParseResult.RuleParent()
			}

			// Condition 1: rule is null OR rule is weak
			ruleCondition := (rule == nil || rule.Weak)
			
			// Condition 2: token is null OR token.pexpr is weak
			tokenCondition := true
			if token != nil && token.Pexpr != nil {
				pexpr := token.Pexpr.(*Pexpr)
				tokenCondition = pexpr.Weak
			}

			// Remove only if BOTH conditions are true
			if ruleCondition && tokenCondition {
				n.RemoveChildNode(child)
			}
		}
	}

	// Merge single child into parent unless both are strong
	if n.firstChildNode != nil && n.firstChildNode == n.lastChildNode {
		n.mergeChildNode()
	}
}

// mergeChildNode merges this node's sole child into this node.
func (n *Node) mergeChildNode() {
	child := n.firstChildNode
	if child == nil || child.nextChildNode != nil {
		return
	}

	parentRule := (*Rule)(nil)
	childRule := (*Rule)(nil)

	if n.ParseResult != nil {
		parentRule = n.ParseResult.RuleParent()
	}
	if child.ParseResult != nil {
		childRule = child.ParseResult.RuleParent()
	}

	parentStrong := parentRule != nil && !parentRule.Weak
	childStrong := childRule != nil && !childRule.Weak

	// Don't merge strong nodes or tokens into strong rule nodes
	if parentStrong && (childStrong || child.Token != nil) {
		return
	}

	// Don't merge multi-element children into strong rule nodes
	if parentStrong && child.firstChildNode != nil && child.firstChildNode != child.lastChildNode {
		return
	}

	// Move child's children to this node
	for _, grandchild := range child.SafeChildNodes() {
		child.RemoveChildNode(grandchild)
		n.AppendChildNode(grandchild)
	}

	// Inherit child's token if any
	n.Token = child.Token

	// If child was a non-strong rule, adopt its ParseResult
	if childRule != nil && !parentStrong {
		if n.ParseResult != nil {
			n.ParseResult.node = nil
		}
		if child.ParseResult != nil {
			child.ParseResult.node = nil
			child.ParseResult.InsertNode(n)
		}
	}

	n.RemoveChildNode(child)
}

// ============================================================================
// String representation
// ============================================================================

// ToString returns a string representation of the AST.
func (n *Node) ToString() string {
	printSpace := false
	return n.toStringIndented(0, printSpace)
}

// toStringIndented returns indented string representation.
func (n *Node) toStringIndented(depth uint32, printSpace bool) string {
	s := ""
	needsParen := n.firstChildNode != nil
	rule := (*Rule)(nil)

	if n.ParseResult != nil {
		rule = n.ParseResult.Rule
	}

	if rule != nil && n.Token == nil {
		s += "\n"
		printSpace = false

		indent := ""
		for i := uint32(0); i < depth*2; i++ {
			indent += " "
		}
		s += indent + rule.Sym.Name

		needsParen = true
	}

	if needsParen {
		if printSpace {
			s += " "
			printSpace = false
		}
		s += "("
	}

	if n.Token != nil {
		token := n.Token
		if printSpace {
			s += " "
		}

		isStrongKeyword := token.Type == TokenTypeKeyword && token.Pexpr != nil
		pexpr := token.Pexpr.(*Pexpr)
		if token.Type == TokenTypeKeyword && pexpr != nil {
			isStrongKeyword = !pexpr.Weak
		}

		if isStrongKeyword {
			s += "\""
		}

		s += token.GetName()

		if isStrongKeyword {
			s += "\""
		}

		printSpace = true
	} else {
		for _, child := range n.ChildNodes() {
			s += child.toStringIndented(depth+1, printSpace)
		}
	}

	if needsParen {
		s += ")"
		printSpace = true
	}

	return s
}

// Dump outputs debugging information about this node.
func (n *Node) Dump() {
	fmt.Println(n.ToString())
}

// ============================================================================
// Helper methods
// ============================================================================

// SetToken sets the token for this node (for token-based nodes).
func (n *Node) SetToken(token *Token) {
	n.Token = token
	if token != nil {
		n.Location = token.Location
	}
}

// computeLocation computes the location from token positions.
func (n *Node) computeLocation() {
	// This will be filled in based on tokens when available
	// For now, use default location
	n.Location = Location{}
}

// CountChildNodes returns the number of child nodes.
func (n *Node) CountChildNodes() uint32 {
	count := uint32(0)
	for child := n.firstChildNode; child != nil; child = child.nextChildNode {
		count++
	}
	return count
}

// IndexChildNode returns the child at the given index, or nil.
func (n *Node) IndexChildNode(index uint32) *Node {
	count := uint32(0)
	for child := n.firstChildNode; child != nil; child = child.nextChildNode {
		if count == index {
			return child
		}
		count++
	}
	return nil
}
