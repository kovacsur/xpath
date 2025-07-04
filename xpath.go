package xpath

import (
	"errors"
	"fmt"
)

// CompileOptions allows customizing the behavior of the XPath parser.
type CompileOptions struct {
	StrictEOF bool // If true, require full input consumption (no trailing tokens)
	// Future strictness options can be added here
}

// StrictPreset enables all strictness options (update as new options are added)
var StrictPreset = CompileOptions{
	StrictEOF: true,
}

// NodeType represents a type of XPath node.
type NodeType int

const (
	// RootNode is a root node of the XML document or node tree.
	RootNode NodeType = iota

	// ElementNode is an element, such as <element>.
	ElementNode

	// AttributeNode is an attribute, such as id='123'.
	AttributeNode

	// TextNode is the text content of a node.
	TextNode

	// CommentNode is a comment node, such as <!-- my comment -->
	CommentNode

	// allNode is any types of node, used by xpath package only to predicate match.
	allNode
)

// NodeNavigator provides cursor model for navigating XML data.
type NodeNavigator interface {
	// NodeType returns the XPathNodeType of the current node.
	NodeType() NodeType

	// LocalName gets the Name of the current node.
	LocalName() string

	// Prefix returns namespace prefix associated with the current node.
	Prefix() string

	// Value gets the value of current node.
	Value() string

	// Copy does a deep copy of the NodeNavigator and all its components.
	Copy() NodeNavigator

	// MoveToRoot moves the NodeNavigator to the root node of the current node.
	MoveToRoot()

	// MoveToParent moves the NodeNavigator to the parent node of the current node.
	MoveToParent() bool

	// MoveToNextAttribute moves the NodeNavigator to the next attribute on current node.
	MoveToNextAttribute() bool

	// MoveToChild moves the NodeNavigator to the first child node of the current node.
	MoveToChild() bool

	// MoveToFirst moves the NodeNavigator to the first sibling node of the current node.
	MoveToFirst() bool

	// MoveToNext moves the NodeNavigator to the next sibling node of the current node.
	MoveToNext() bool

	// MoveToPrevious moves the NodeNavigator to the previous sibling node of the current node.
	MoveToPrevious() bool

	// MoveTo moves the NodeNavigator to the same position as the specified NodeNavigator.
	MoveTo(NodeNavigator) bool
}

// NodeIterator holds all matched Node object.
type NodeIterator struct {
	node  NodeNavigator
	query query
}

// Current returns current node which matched.
func (t *NodeIterator) Current() NodeNavigator {
	return t.node
}

// MoveNext moves Navigator to the next match node.
func (t *NodeIterator) MoveNext() bool {
	n := t.query.Select(t)
	if n == nil {
		return false
	}
	if !t.node.MoveTo(n) {
		t.node = n.Copy()
	}
	return true
}

// Select selects a node set using the specified XPath expression.
// This method is deprecated, recommend using Expr.Select() method instead.
func Select(root NodeNavigator, expr string) *NodeIterator {
	exp, err := Compile(expr)
	if err != nil {
		panic(err)
	}
	return exp.Select(root)
}

// Expr is an XPath expression for query.
type Expr struct {
	s string
	q query
}

type iteratorFunc func() NodeNavigator

func (f iteratorFunc) Current() NodeNavigator {
	return f()
}

// Evaluate returns the result of the expression.
// The result type of the expression is one of the follow: bool,float64,string,NodeIterator).
func (expr *Expr) Evaluate(root NodeNavigator) interface{} {
	val := expr.q.Evaluate(iteratorFunc(func() NodeNavigator { return root }))
	switch val.(type) {
	case query:
		return &NodeIterator{query: expr.q.Clone(), node: root}
	}
	return val
}

// Select selects a node set using the specified XPath expression.
func (expr *Expr) Select(root NodeNavigator) *NodeIterator {
	return &NodeIterator{query: expr.q.Clone(), node: root}
}

// String returns XPath expression string.
func (expr *Expr) String() string {
	return expr.s
}

// Compile compiles an XPath expression string.
func Compile(expr string) (*Expr, error) {
	return CompileWithOptionsAndNS(expr, CompileOptions{}, nil)
}

// MustCompile compiles an XPath expression string and ignored error.
func MustCompile(expr string) *Expr {
	exp, err := Compile(expr)
	if err != nil {
		return &Expr{s: expr, q: nopQuery{}}
	}
	return exp
}

// CompileWithNS compiles an XPath expression string, using given namespaces map.
func CompileWithNS(expr string, namespaces map[string]string) (*Expr, error) {
	return CompileWithOptionsAndNS(expr, CompileOptions{}, namespaces)
}

// CompileWithOptions compiles an XPath expression string with the given options.
func CompileWithOptions(expr string, opts CompileOptions) (*Expr, error) {
	return CompileWithOptionsAndNS(expr, opts, nil)
}

func CompileWithOptionsAndNS(expr string, opts CompileOptions, namespaces map[string]string) (*Expr, error) {
	if expr == "" {
		return nil, errors.New("expr expression is nil")
	}
	q, p, err := build(expr, namespaces)
	if err != nil {
		return nil, err
	}
	if opts.StrictEOF && p != nil && p.r.typ != itemEOF {
		return nil, fmt.Errorf("unexpected token after end of expression: %s", p.r.text[p.r.pos-p.r.currSize-1:])
	}
	if q == nil {
		return nil, fmt.Errorf("undeclared variable in XPath expression: %s", expr)
	}
	return &Expr{s: expr, q: q}, nil
}
