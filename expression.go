package main

import (
	"fmt"
)

type Expr interface {
	Accept(ExprVisitor) interface{}
}

type ExprVisitor interface {
	VisitAssign(Expr) interface{}
	VisitBinary(Expr) interface{}
	VisitCall(Expr) interface{}
	VisitGet(Expr) interface{}
	VisitGrouping(Expr) interface{}
	VisitLiteral(Expr) interface{}
	VisitLogical(Expr) interface{}
	VisitSet(Expr) interface{}
	VisitSuper(Expr) interface{}
	VisitThis(Expr) interface{}
	VisityUnary(Expr) interface{}
	VisitVariable(Expr) interface{}
}

type Assign struct {
	Name  Token
	Value Expr
}

func (a Assign) Accept(v ExprVisitor) interface{} {
	return v.VisitAssign(a)
}

func (a Assign) String() string {
	return fmt.Sprintf("%v = %v;", a.Name, a.Value)
}

type Binary struct {
	Left     Expr
	Operator Token
	Right    Expr
}

func (b Binary) Accept(v ExprVisitor) interface{} {
	return v.VisitBinary(b)
}

type Call struct {
	Callee Expr
	Paren  Token
	Args   []Expr
}

func (c Call) Accept(v ExprVisitor) interface{} {
	return v.VisitCall(c)
}

func (c Call) String() string {
	return fmt.Sprintf("call<%v>(%v)", c.Callee, c.Args)
}

type Get struct {
	Object Expr
	Name   Token
}

func (g Get) Accept(v ExprVisitor) interface{} {
	return v.VisitGet(g)
}

func (g Get) String() string {
	return fmt.Sprintf("%s.get(%q)", g.Object, g.Name.Lexeme)
}

type Set struct {
	Object Expr
	Name   Token
	Value  Expr
}

func (s Set) Accept(v ExprVisitor) interface{} {
	return v.VisitSet(s)
}

func (s Set) String() string {
	return fmt.Sprintf("%s.set(%q) = %v", s.Object, s.Name.Lexeme, s.Value)
}

type Grouping struct {
	Expression Expr
}

func (g Grouping) Accept(v ExprVisitor) interface{} {
	return v.VisitGrouping(g)
}

type Literal struct {
	Value interface{}
}

func (l Literal) Accept(v ExprVisitor) interface{} {
	return v.VisitLiteral(l)
}

type Logical struct {
	Left     Expr
	Operator Token
	Right    Expr
}

func (l Logical) Accept(v ExprVisitor) interface{} {
	return v.VisitLogical(l)
}

type Super struct {
	Keyword Token
	Method  Token
}

func (s Super) Accept(v ExprVisitor) interface{} {
	return v.VisitSuper(s)
}

type This struct {
	Keyword Token
}

func (t This) Accept(v ExprVisitor) interface{} {
	return v.VisitThis(t)
}

type Unary struct {
	Operator Token
	Right    Expr
}

func (u Unary) Accept(v ExprVisitor) interface{} {
	return v.VisityUnary(u)
}

type Variable struct {
	Name Token
	// Needed due to a quirk of our resolver implementation...
	// We are storing scope-distance lookups in a hash table,
	// with the keys being the variable expressions themselves.
	// In Golang, structs are hashable for this purpose, but
	// they are hashed based on their contents... not just the
	// value of a pointer to the object itself (asin E.g. Java).
	// This allows different expressions to collide with one another
	// if their contents are the same.
	//
	// This surfaces particularly in the syntactic-sugar of for-loops,
	// which end up expanding to a while-loop with a special inner
	// block to contain the incrementor. So in this declaration:
	//   for ; i < 3; i = i + 1 { ... }
	// The "i < 3" and the "i = i + 1" expressions are in different
	// blocks, and should have difference scope-distance values.
	// But because they are declared on the same line, both
	// references to "i" will end up with the same internal
	// representation, and will collide... The resolver will resolve
	// both of them, but the latter will overwrite the former.
	//
	// This field lets us inject a unique identifier for each one
	// to ensure they don't collide.
	Unique int
}

func (v Variable) Accept(visitor ExprVisitor) interface{} {
	return visitor.VisitVariable(v)
}

func (v Variable) String() string {
	return fmt.Sprintf("var(%v)[%d]", v.Name, v.Unique)
}
