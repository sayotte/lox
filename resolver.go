package main

import (
	"fmt"
)

type scope struct {
	declared   map[string]int
	defined    map[string]int
	referenced map[string]bool
}

func (s *scope) containsKey(key string) bool {
	_, found := s.declared[key]
	return found
}

func (s *scope) declare(key string, line int) {
	s.ensureInit()
	s.declared[key] = line
}

func (s *scope) isDeclared(key string) (bool, int) {
	line, found := s.declared[key]
	return found, line
}

func (s *scope) define(key string, line int) {
	s.defined[key] = line
}

func (s *scope) isDefined(key string) (bool, int) {
	line, found := s.defined[key]
	return found, line
}

func (s *scope) reference(key string) {
	s.referenced[key] = true
}

func (s *scope) isReferenced(key string) bool {
	return s.referenced[key]
}

func (s *scope) keys() []string {
	keys := make([]string, 0, len(s.declared))
	for key := range s.declared {
		keys = append(keys, key)
	}
	return keys
}

func (s *scope) ensureInit() {
	if s.declared == nil {
		s.declared = make(map[string]int)
	}
	if s.defined == nil {
		s.defined = make(map[string]int)
	}
	if s.referenced == nil {
		s.referenced = make(map[string]bool)
	}
}

type resolutionError struct {
	line int
	msg  string
}

func (re resolutionError) error() error {
	return fmt.Errorf("resolution error on line %d: %s", re.line, re.msg)
}

type FunctionType int

const (
	NONEFUNC    = FunctionType(0)
	FUNCTION    = FunctionType(1)
	INITIALIZER = FunctionType(2)
	METHOD      = FunctionType(3)
)

type ClassType int

const (
	NONECLASS     = ClassType(0)
	SUBCLASSCLASS = ClassType(1)
	CLASSCLASS    = ClassType(2)
)

type Resolver struct {
	scopes              []*scope
	interpreter         *Interpreter
	currentFunctionType FunctionType
	currentClassType    ClassType
}

func (r *Resolver) Resolve(stmts []Stmt) (returnErr error) {
	defer func() {
		if r := recover(); r != nil {
			returnErr = r.(resolutionError).error()
		}
	}()

	r.resolveStmts(stmts)

	return
}

func (r *Resolver) resolveError(line int, msg string) {
	panic(resolutionError{
		line: line,
		msg:  msg,
	})
}

func (r *Resolver) resolveStmts(stmts []Stmt) {
	for _, stmt := range stmts {
		r.resolveStmt(stmt)
	}
}

func (r *Resolver) resolveStmt(stmt Stmt) {
	stmt.Accept(r)
}

func (r *Resolver) resolveExpr(expr Expr) {
	expr.Accept(r)
}

func (r *Resolver) resolveFunction(fStmt FunctionStmt, typ FunctionType) {
	enclosingFunctionType := r.currentFunctionType
	r.currentFunctionType = typ

	r.beginScope()
	for _, param := range fStmt.Params {
		r.declare(param)
		r.define(param)
	}
	r.resolveStmts(fStmt.Body)
	r.endScope()
	r.currentFunctionType = enclosingFunctionType
}

func (r *Resolver) resolveLocal(expr Expr, name Token) {
	for depth := 0; depth < len(r.scopes); depth++ {
		idx := len(r.scopes) - depth - 1
		if r.scopes[idx].containsKey(name.Lexeme) {
			r.interpreter.Resolve(expr, depth)
			r.scopes[idx].reference(name.Lexeme)
			return
		}
	}
}

func (r *Resolver) beginScope() {
	newScope := &scope{}
	r.scopes = append(r.scopes, newScope)
}

func (r *Resolver) peekScope() *scope {
	if len(r.scopes) == 0 {
		return nil
	}
	return r.scopes[len(r.scopes)-1]
}

func (r *Resolver) endScope() {
	for _, key := range r.peekScope().keys() {
		if !r.peekScope().isReferenced(key) {
			if key == "this" || key == "super" {
				continue
			}
			_, line := r.peekScope().isDeclared(key)
			r.resolveError(line, fmt.Sprintf("unused local variable %q", key))
		}
	}
	r.scopes = r.scopes[:len(r.scopes)-1]
}

func (r *Resolver) declare(name Token) {
	if len(r.scopes) == 0 {
		return // it's global, no resolution needed
	}
	if r.peekScope().containsKey(name.Lexeme) {
		r.resolveError(name.Line, fmt.Sprintf("already a variable with name %q in this scope", name.Lexeme))
	}

	r.peekScope().declare(name.Lexeme, name.Line)
}

func (r *Resolver) define(name Token) {
	if len(r.scopes) == 0 {
		return
	}
	r.peekScope().define(name.Lexeme, name.Line)
}

func (r *Resolver) VisitExpressionStmt(stmt Stmt) {
	eStmt := stmt.(ExprStmt)
	r.resolveExpr(eStmt.Expression)
}

func (r *Resolver) VisitFunctionStmt(stmt Stmt) {
	fStmt := stmt.(FunctionStmt)
	r.declare(fStmt.Name)
	r.define(fStmt.Name)
	r.resolveFunction(fStmt, FUNCTION)
}

func (r *Resolver) VisitIfStmt(stmt Stmt) {
	iStmt := stmt.(IfStmt)
	r.resolveExpr(iStmt.Condition)
	r.resolveStmt(iStmt.Then)
	if iStmt.Else != nil {
		r.resolveStmt(iStmt.Else)
	}
}

func (r *Resolver) VisitPrintStmt(stmt Stmt) {
	pStmt := stmt.(PrintStmt)
	r.resolveExpr(pStmt.Expression)
}

func (r *Resolver) VisitWhileStmt(stmt Stmt) {
	wStmt := stmt.(WhileStmt)
	r.resolveExpr(wStmt.Condition)
	r.resolveStmt(wStmt.Body)
}

func (r *Resolver) VisitBlockStmt(stmt Stmt) {
	blockStmt := stmt.(BlockStmt)
	r.beginScope()
	r.resolveStmts(blockStmt.Statements)
	r.endScope()
}

func (r *Resolver) VisitClassStmt(stmt Stmt) {
	enclosingClassType := r.currentClassType
	r.currentClassType = CLASSCLASS

	cs := stmt.(ClassStmt)
	r.declare(cs.Name)
	r.define(cs.Name)
	if cs.Superclass != nil {
		if cs.Name.Lexeme == cs.Superclass.Name.Lexeme {
			r.resolveError(cs.Name.Line, "A class can't inherit from itself.")
		}
		r.currentClassType = SUBCLASSCLASS
		r.resolveExpr(cs.Superclass)
		r.beginScope()
		r.peekScope().declare("super", cs.Name.Line)
		r.peekScope().define("super", cs.Name.Line)
	}

	r.beginScope()
	r.peekScope().declare("this", cs.Name.Line)
	r.peekScope().define("this", cs.Name.Line)

	for _, method := range cs.Methods {
		funcType := METHOD
		if method.Name.Lexeme == "init" {
			funcType = INITIALIZER
		}
		r.resolveFunction(method, funcType)
	}

	r.endScope()
	if cs.Superclass != nil {
		r.endScope()
	}
	r.currentClassType = enclosingClassType
}

func (r *Resolver) VisitReturnStmt(stmt Stmt) {
	rStmt := stmt.(ReturnStmt)

	if rStmt.Value != nil {
		if r.currentFunctionType == NONEFUNC {
			r.resolveError(rStmt.Keyword.Line, "can't return a value from top-level code.")
		}
		if r.currentFunctionType == INITIALIZER {
			r.resolveError(rStmt.Keyword.Line, "can't return a value from an initializer")
		}
		r.resolveExpr(rStmt.Value)
	}
}

func (r *Resolver) VisitVarStmt(stmt Stmt) {
	varStmt := stmt.(VariableStmt)
	r.declare(varStmt.Name)
	if varStmt.Initializer != nil {
		r.resolveExpr(varStmt.Initializer)
	}
	r.define(varStmt.Name)
}

func (r *Resolver) VisitAssign(expr Expr) interface{} {
	assignExpr := expr.(Assign)
	r.resolveExpr(assignExpr.Value)
	r.resolveLocal(assignExpr, assignExpr.Name)
	return nil
}

func (r *Resolver) VisitBinary(expr Expr) interface{} {
	bExpr := expr.(Binary)
	r.resolveExpr(bExpr.Left)
	r.resolveExpr(bExpr.Right)
	return nil
}

func (r *Resolver) VisitCall(expr Expr) interface{} {
	ce := expr.(Call)
	r.resolveExpr(ce.Callee)
	for _, param := range ce.Args {
		r.resolveExpr(param)
	}
	return nil
}

func (r *Resolver) VisitGet(expr Expr) interface{} {
	ge := expr.(Get)
	r.resolveExpr(ge.Object)
	return nil
}

func (r *Resolver) VisitGrouping(expr Expr) interface{} {
	ge := expr.(Grouping)
	r.resolveExpr(ge.Expression)
	return nil
}

func (r *Resolver) VisitLiteral(expr Expr) interface{} {
	return nil
}

func (r *Resolver) VisitLogical(expr Expr) interface{} {
	le := expr.(Logical)
	r.resolveExpr(le.Left)
	r.resolveExpr(le.Right)
	return nil
}

func (r *Resolver) VisitSet(expr Expr) interface{} {
	se := expr.(Set)
	r.resolveExpr(se.Value)
	r.resolveExpr(se.Object)
	return nil
}

func (r *Resolver) VisitSuper(expr Expr) interface{} {
	se := expr.(Super)
	if r.currentClassType == NONECLASS {
		r.resolveError(se.Keyword.Line, "Can't use 'super' outside of a class.")
	}
	if r.currentClassType != SUBCLASSCLASS {
		r.resolveError(se.Keyword.Line, "Can't use 'super' in a class with no superclass.")
	}
	r.resolveLocal(se, se.Keyword)
	return nil
}

func (r *Resolver) VisitThis(expr Expr) interface{} {
	te := expr.(This)
	if r.currentClassType == NONECLASS {
		r.resolveError(te.Keyword.Line, "Cannot use 'this' outside of a class method.")
	}
	r.resolveLocal(te, te.Keyword)
	return nil
}

func (r *Resolver) VisityUnary(expr Expr) interface{} {
	ue := expr.(Unary)
	r.resolveExpr(ue.Right)
	return nil
}

func (r *Resolver) VisitVariable(expr Expr) interface{} {
	varExpr := expr.(Variable)
	innerScope := r.peekScope()
	if innerScope != nil {
		declared, _ := innerScope.isDeclared(varExpr.Name.Lexeme)
		if declared {
			defined, _ := innerScope.isDefined(varExpr.Name.Lexeme)
			if !defined {
				resErr := resolutionError{
					line: varExpr.Name.Line,
					msg:  "Can't read local variable in its own initializer",
				}
				panic(resErr)
			}
		}
	}
	r.resolveLocal(varExpr, varExpr.Name)
	return nil
}
