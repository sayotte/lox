package main

import "fmt"

type StmtVisitor interface {
	VisitClassStmt(Stmt)
	VisitExpressionStmt(Stmt)
	VisitFunctionStmt(Stmt)
	VisitIfStmt(Stmt)
	VisitPrintStmt(Stmt)
	VisitWhileStmt(Stmt)
	VisitBlockStmt(Stmt)
	VisitReturnStmt(Stmt)
	VisitVarStmt(Stmt)
}

type Stmt interface {
	Accept(visitor StmtVisitor)
}

type ExprStmt struct {
	Expression Expr
}

func (es ExprStmt) Accept(visitor StmtVisitor) {
	visitor.VisitExpressionStmt(es)
}

type ClassStmt struct {
	Name       Token
	Superclass *Variable
	Methods    []FunctionStmt
}

func (cs ClassStmt) Accept(visitor StmtVisitor) {
	visitor.VisitClassStmt(cs)
}

type FunctionStmt struct {
	Name   Token
	Params []Token
	Body   []Stmt
}

func (fs FunctionStmt) Accept(visitor StmtVisitor) {
	visitor.VisitFunctionStmt(fs)
}

type IfStmt struct {
	Condition Expr
	Then      Stmt
	Else      Stmt
}

func (i IfStmt) Accept(visitor StmtVisitor) {
	visitor.VisitIfStmt(i)
}

func (i IfStmt) String() string {
	if i.Else == nil {
		return fmt.Sprintf("if(%v) %v", i.Condition, i.Then)
	}
	return fmt.Sprintf("if(%v) %v else %v", i.Condition, i.Then, i.Else)
}

type PrintStmt struct {
	Expression Expr
}

func (p PrintStmt) Accept(visitor StmtVisitor) {
	visitor.VisitPrintStmt(p)
}

func (p PrintStmt) String() string {
	return fmt.Sprintf("print %v", p.Expression)
}

type ReturnStmt struct {
	Keyword Token
	Value   Expr
}

func (r ReturnStmt) Accept(visitor StmtVisitor) {
	visitor.VisitReturnStmt(r)
}

func (r ReturnStmt) String() string {
	return fmt.Sprintf("return %v; ", r.Value)
}

type WhileStmt struct {
	Condition Expr
	Body      Stmt
}

func (w WhileStmt) Accept(visitor StmtVisitor) {
	visitor.VisitWhileStmt(w)
}

func (w WhileStmt) String() string {
	return fmt.Sprintf("while (%v) %v ", w.Condition, w.Body)
}

type BlockStmt struct {
	Statements []Stmt
}

func (b BlockStmt) Accept(visitor StmtVisitor) {
	visitor.VisitBlockStmt(b)
}

func (b BlockStmt) String() string {
	return fmt.Sprintf("{ %v } ", b.Statements)
}

type VariableStmt struct {
	Name        Token
	Initializer Expr
}

func (vs VariableStmt) Accept(visitor StmtVisitor) {
	visitor.VisitVarStmt(vs)
}
