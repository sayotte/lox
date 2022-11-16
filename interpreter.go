package main

import (
	"fmt"
	"io"
	"reflect"
)

type runtimeError struct {
	line int
	msg  string
}

func (rte runtimeError) error() error {
	return fmt.Errorf("runtime error on line %d: %s", rte.line, rte.msg)
}

type returnable struct {
	Value interface{}
}

type Callable interface {
	Arity() int
	Call(interpreter *Interpreter, args []interface{}) interface{}
}

type Interpreter struct {
	Stdout        io.Writer
	localDistance map[Expr]int // <-- this is so dumb
	globals       *environment
	env           *environment
	initialized   bool
}

func (i *Interpreter) Interpret(stmts []Stmt) (returnErr error) {
	defer func() {
		if r := recover(); r != nil {
			returnErr = r.(runtimeError).error()
		}
	}()

	if i.env == nil {
		i.init()
	}

	for _, stmt := range stmts {
		i.execute(stmt)
	}

	return nil
}

func (i *Interpreter) Resolve(expr Expr, distance int) {
	i.ensureInit()
	i.localDistance[expr] = distance
}

func (i *Interpreter) ensureInit() {
	if !i.initialized {
		i.init()
	}
}

func (i *Interpreter) init() {
	i.env = &environment{interpreter: i}
	i.globals = i.env
	i.globals.define("clock", ClockBuiltin{})
	i.localDistance = make(map[Expr]int)
	i.initialized = true
}

func (i *Interpreter) runtimeError(line int, msg string) {
	panic(runtimeError{
		line: line,
		msg:  msg,
	})
}

func (i *Interpreter) evaluate(expr Expr) interface{} {
	return expr.Accept(i)
}

func (i *Interpreter) execute(stmt Stmt) {
	stmt.Accept(i)
}

func (i *Interpreter) executeBlock(stmts []Stmt, newEnv *environment) {
	prevEnv := i.env
	defer func() {
		i.env = prevEnv
	}()
	i.env = newEnv

	for _, stmt := range stmts {
		i.execute(stmt)
	}
}

func (i *Interpreter) VisitAssign(expr Expr) interface{} {
	assignExpr := expr.(Assign)
	value := i.evaluate(assignExpr.Value)
	distance, found := i.localDistance[assignExpr]
	if found {
		i.env.assignAt(distance, assignExpr.Name, value)
	} else {
		i.globals.assign(assignExpr.Name, value)
	}
	return value
}

func (i *Interpreter) VisitBinary(expr Expr) interface{} {
	b := expr.(Binary)
	left := i.evaluate(b.Left)
	right := i.evaluate(b.Right)
	switch b.Operator.Type {
	case MINUS:
		i.checkNumberOperands(b.Operator, left, right)
		return left.(float64) - right.(float64)
	case SLASH:
		i.checkNumberOperands(b.Operator, left, right)
		return left.(float64) / right.(float64)
	case STAR:
		i.checkNumberOperands(b.Operator, left, right)
		return left.(float64) * right.(float64)
	case PLUS:
		switch leftTyped := left.(type) {
		case float64:
			i.checkNumberOperands(b.Operator, left, right)
			return leftTyped + right.(float64)
		case string:
			i.checkStringOperands(b.Operator, left, right)
			return leftTyped + right.(string)
		default:
			i.runtimeError(
				b.Operator.Line,
				fmt.Sprintf("'+' can operate on numbers or strings, found %T", left),
			)
		}
	case GREATER:
		i.checkNumberOperands(b.Operator, left, right)
		return left.(float64) > right.(float64)
	case GREATER_EQUAL:
		i.checkNumberOperands(b.Operator, left, right)
		return left.(float64) >= right.(float64)
	case LESS:
		i.checkNumberOperands(b.Operator, left, right)
		return left.(float64) < right.(float64)
	case LESS_EQUAL:
		i.checkNumberOperands(b.Operator, left, right)
		return left.(float64) <= right.(float64)
	case BANG_EQUAL:
		return !reflect.DeepEqual(left, right)
	case EQUAL_EQUAL:
		return reflect.DeepEqual(left, right)
	}

	panic("VisitBinary hit intended-unreachable code")
}

func (i *Interpreter) checkNumberOperands(op Token, left, right interface{}) {
	_, leftOk := left.(float64)
	_, rightOk := right.(float64)
	if !leftOk || !rightOk {
		i.runtimeError(
			op.Line,
			fmt.Sprintf("%q, operands %q and %q must be numbers", op.Lexeme, left, right),
		)
	}
}

func (i *Interpreter) checkStringOperands(op Token, left, right interface{}) {
	_, leftOk := left.(string)
	_, rightOk := right.(string)
	if !leftOk || !rightOk {
		i.runtimeError(
			op.Line,
			fmt.Sprintf("%q, operands %q and %q must be strings", op.Lexeme, left, right),
		)
	}
}

func (i *Interpreter) VisitCall(expr Expr) interface{} {
	callExpr := expr.(Call)
	callee := i.evaluate(callExpr.Callee)
	var args []interface{}
	for _, argExpr := range callExpr.Args {
		args = append(args, i.evaluate(argExpr))
	}
	function, ok := callee.(Callable)
	if !ok {
		i.runtimeError(callExpr.Paren.Line, "Can only call functions and classes.")
	}
	if len(args) != function.Arity() {
		i.runtimeError(
			callExpr.Paren.Line,
			fmt.Sprintf("Expected %d args but got %d.", function.Arity(), len(args)),
		)
	}

	return function.Call(i, args)
}

func (i *Interpreter) VisitGet(expr Expr) interface{} {
	ge := expr.(Get)
	obj := i.evaluate(ge.Object)
	instance, ok := obj.(*Instance)
	if !ok {
		i.runtimeError(ge.Name.Line, "Only class instances have properties.")
	}
	val, err := instance.Get(ge.Name)
	if err != nil {
		i.runtimeError(ge.Name.Line, err.Error())
	}
	return val
}

func (i *Interpreter) VisitGrouping(expr Expr) interface{} {
	grp := expr.(Grouping)
	return i.evaluate(grp.Expression)
}

func (i *Interpreter) VisitLiteral(expr Expr) interface{} {
	lit := expr.(Literal)
	return lit.Value
}

func (i *Interpreter) VisitLogical(expr Expr) interface{} {
	logicalExpr := expr.(Logical)
	left := i.evaluate(logicalExpr.Left)
	leftTruthy := i._isTruthy(left)

	if logicalExpr.Operator.Type == OR {
		// short-circuit OR
		if leftTruthy {
			return left
		}
	} else {
		// short-circuit AND
		if !leftTruthy {
			return left
		}
	}

	return i.evaluate(logicalExpr.Right)
}

func (i *Interpreter) VisitSet(expr Expr) interface{} {
	se := expr.(Set)
	obj := i.evaluate(se.Object)
	instance, ok := obj.(*Instance)
	if !ok {
		i.runtimeError(se.Name.Line, "Only class instances have fields.")
	}
	value := i.evaluate(se.Value)
	instance.Set(se.Name, value)
	return value
}

func (i *Interpreter) VisitSuper(expr Expr) interface{} {
	se := expr.(Super)
	distance := i.localDistance[se]
	superclass := i.env.getAt(distance, se.Keyword).(Class)
	method, found := superclass.findMethod(se.Method.Lexeme)
	if !found {
		i.runtimeError(se.Method.Line, fmt.Sprintf("Undefined property %q.", se.Method.Lexeme))
	}
	instance := i.env.getAt(distance-1, Token{Lexeme: "this"}).(*Instance)
	return method.bindMethodToInstance(instance)
}

func (i *Interpreter) VisitThis(expr Expr) interface{} {
	return i.lookupVariable(expr)
}

func (i *Interpreter) VisityUnary(expr Expr) interface{} {
	u := expr.(Unary)
	right := i.evaluate(u.Right)

	switch u.Operator.Type {
	case MINUS:
		rightNum := right.(float64)
		return -rightNum
	case BANG:
		switch val := right.(type) {
		case nil:
			return true
		case bool:
			return !val
		default:
			return false
		}
	}

	panic("Interpreter hit intended-unreachable code in VisitUnary")
}

func (i *Interpreter) VisitVariable(expr Expr) interface{} {
	return i.lookupVariable(expr)
}

func (i *Interpreter) lookupVariable(expr Expr) interface{} {
	var name Token
	switch typedExpr := expr.(type) {
	case Variable:
		name = typedExpr.Name
	case This:
		name = typedExpr.Keyword
	default:
		panic("hit intended-unreachable code")
	}

	distance, found := i.localDistance[expr]
	if found {
		return i.env.getAt(distance, name)
	} else {
		return i.globals.get(name)
	}
}

func (i *Interpreter) VisitBlockStmt(stmt Stmt) {
	newEnv := &environment{
		// note that a BlockStmt is only used for non-call
		// operations like if/while/for, and for these the
		// enclosing scope *should* be visible
		enclosing:   i.env,
		interpreter: i,
	}
	blockStmt := stmt.(BlockStmt)
	i.executeBlock(blockStmt.Statements, newEnv)
}

func (i *Interpreter) VisitClassStmt(stmt Stmt) {
	cs := stmt.(ClassStmt)

	var superclass Class
	if cs.Superclass != nil {
		superclassMaybe := i.evaluate(cs.Superclass)
		var ok bool
		superclass, ok = superclassMaybe.(Class)
		if !ok {
			i.runtimeError(cs.Name.Line, "Superclass must be a class.")
		}
	}

	i.env.define(cs.Name.Lexeme, nil)

	if cs.Superclass != nil {
		i.env = &environment{enclosing: i.env}
		i.env.define("super", superclass)
	}

	methods := make(map[string]Function)
	for _, methodStmt := range cs.Methods {
		var isInit bool
		if methodStmt.Name.Lexeme == "init" {
			isInit = true
		}
		method := Function{
			Declaration:   methodStmt,
			Closure:       i.env,
			isInitializer: isInit,
		}
		methods[methodStmt.Name.Lexeme] = method
	}

	class := Class{
		Name:       cs.Name.Lexeme,
		Methods:    methods,
		Superclass: &superclass,
	}
	if cs.Superclass != nil {
		i.env = i.env.enclosing
	}
	i.env.assign(cs.Name, class)
}

func (i *Interpreter) VisitExpressionStmt(stmt Stmt) {
	i.evaluate(stmt.(ExprStmt).Expression)
}

func (i *Interpreter) VisitFunctionStmt(stmt Stmt) {
	funStmt := stmt.(FunctionStmt)
	fun := Function{
		Declaration:   funStmt,
		Closure:       i.env,
		isInitializer: false,
	}
	i.env.define(funStmt.Name.Lexeme, fun)
}

func (i *Interpreter) VisitIfStmt(stmt Stmt) {
	ifStmt := stmt.(IfStmt)
	if i._isTruthy(i.evaluate(ifStmt.Condition)) {
		i.execute(ifStmt.Then)
	} else if ifStmt.Else != nil {
		i.execute(ifStmt.Else)
	}
}

func (i *Interpreter) VisitPrintStmt(stmt Stmt) {
	value := i.evaluate(stmt.(PrintStmt).Expression)
	_, _ = fmt.Fprintln(i.Stdout, value)
}

func (i *Interpreter) VisitReturnStmt(stmt Stmt) {
	returnStmt := stmt.(ReturnStmt)
	var value interface{}
	if returnStmt.Value != nil {
		value = i.evaluate(returnStmt.Value)
	}
	panic(returnable{Value: value})
}

func (i *Interpreter) VisitVarStmt(stmt Stmt) {
	var value interface{}
	vs := stmt.(VariableStmt)
	if vs.Initializer != nil {
		value = i.evaluate(vs.Initializer)
	}
	i.env.define(vs.Name.Lexeme, value)
}

func (i *Interpreter) VisitWhileStmt(stmt Stmt) {
	whileStmt := stmt.(WhileStmt)
	for i._isTruthy(i.evaluate(whileStmt.Condition)) {
		i.execute(whileStmt.Body)
	}
}

func (i *Interpreter) _isTruthy(obj interface{}) bool {
	if obj == nil {
		return false
	}
	boolObj, ok := obj.(bool)
	if ok {
		return boolObj
	}
	return true
}
