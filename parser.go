package main

import "fmt"

type parseError struct {
	line int
	msg  string
}

func (pe parseError) error() error {
	return fmt.Errorf("parse error on line %d: %s", pe.line, pe.msg)
}

type Parser struct {
	Tokens                    []Token
	current                   int
	uniqueVarReferenceCounter int
}

func (p *Parser) Parse() (returnStmts []Stmt, returnErr error) {
	defer func() {
		if r := recover(); r != nil {
			returnErr = r.(parseError).error()
		}
	}()

	var statements []Stmt
	for !p.isAtEnd() {
		statements = append(statements, p.declaration())
	}
	return statements, nil
}

func (p *Parser) parseError(line int, msg string) {
	panic(parseError{
		line: line,
		msg:  msg,
	})
}

func (p *Parser) declaration() Stmt {
	if p.match(CLASS) {
		return p.classDeclaration()
	}
	if p.match(FUN) {
		return p.funDeclaration("function")
	}
	if p.match(VAR) {
		return p.varDeclaration()
	}
	return p.statement()
}

func (p *Parser) classDeclaration() Stmt {
	name := p.consume(IDENTIFIER, "Expect class name.")

	var superclass *Variable
	if p.match(LESS) {
		p.consume(IDENTIFIER, "Expect superclass name after '<'.")
		superclass = &Variable{
			Name:   p.previous(),
			Unique: p.nextUniqueVarRef(),
		}
	}

	p.consume(LEFT_BRACE, "Expect '{' before class body.")

	var methods []FunctionStmt
	for !p._check(RIGHT_BRACE) && p.current < len(p.Tokens) {
		methods = append(methods, p.funDeclaration("method"))
	}

	p.consume(RIGHT_BRACE, "Expect '}' after class body.")

	return ClassStmt{
		Name:       name,
		Superclass: superclass,
		Methods:    methods,
	}
}

func (p *Parser) funDeclaration(kind string) FunctionStmt {
	// grab function name
	name := p.consume(IDENTIFIER, fmt.Sprintf("Expect %s name.", kind))

	// grab function prototype
	p.consume(LEFT_PAREN, fmt.Sprintf("Expect '(' after %s name.", kind))
	var params []Token
	if !p._check(RIGHT_PAREN) {
		for {
			param := p.consume(IDENTIFIER, "Expect parameter name")
			params = append(params, param)
			if !p.match(COMMA) {
				break
			}
		}
	}
	p.consume(RIGHT_PAREN, "Expect ')' after parameters.")

	// grab function body
	p.consume(LEFT_BRACE, fmt.Sprintf("Expect '{' before %s body.", kind))
	body := p.block()

	return FunctionStmt{
		Name:   name,
		Params: params,
		Body:   body,
	}
}

func (p *Parser) nextUniqueVarRef() int {
	p.uniqueVarReferenceCounter++
	return p.uniqueVarReferenceCounter - 1
}

func (p *Parser) varDeclaration() Stmt {
	name := p.consume(IDENTIFIER, "Expect variable name.")
	var initialializer Expr
	if p.match(EQUAL) {
		initialializer = p.expression()
	}

	p.consume(SEMICOLON, "Expect ';' after variable declaration.")
	return VariableStmt{
		Name:        name,
		Initializer: initialializer,
	}
}

func (p *Parser) statement() Stmt {
	if p.match(FOR) {
		return p.forStatement()
	}
	if p.match(IF) {
		return p.ifStatement()
	}
	if p.match(PRINT) {
		return p.printStatement()
	}
	if p.match(RETURN) {
		return p.returnStatement()
	}
	if p.match(WHILE) {
		return p.whileStatement()
	}
	if p.match(LEFT_BRACE) {
		return BlockStmt{p.block()}
	}
	return p.expressionStatement()
}

func (p *Parser) forStatement() Stmt {
	p.consume(LEFT_PAREN, "Expect '(' after 'for'.")

	var initializer Stmt
	if p.match(SEMICOLON) {
		initializer = nil
	} else if p.match(VAR) {
		initializer = p.varDeclaration()
	} else {
		initializer = p.expressionStatement()
	}

	var condition Expr
	if !p._check(SEMICOLON) {
		condition = p.expression()
	}
	p.consume(SEMICOLON, "Expect ';' after loop condition.")

	var increment Expr
	if !p._check(RIGHT_PAREN) {
		increment = p.expression()
	}
	p.consume(RIGHT_PAREN, "Expect ')' after for clause.")

	body := p.statement()

	// in Lox, a for loop is just syntactic sugar for a while loop.

	// construct a synthetic body block, which includes the original
	// body plus the increment at the end of it
	if increment != nil {
		body = BlockStmt{
			Statements: []Stmt{
				body,
				ExprStmt{increment},
			},
		}
	}

	// attach the body to a while loop
	if condition == nil {
		// if condition is unspecified, replace it with "true"
		condition = Literal{Value: true}
	}
	body = WhileStmt{
		Condition: condition,
		Body:      body,
	}

	// if there's an initializer, construct an outer block
	// around the while loop, and invoke the initializer
	// there-- if it's declared, it will be scoped to this
	// block
	if initializer != nil {
		body = BlockStmt{
			Statements: []Stmt{
				initializer,
				body,
			},
		}
	}
	/*
		{ var i = 0
		  while COND {
		    {body}
		    increment
		  }
	*/

	return body
}

func (p *Parser) ifStatement() Stmt {
	p.consume(LEFT_PAREN, "Expect '(' after 'if'.")
	condition := p.expression()
	p.consume(RIGHT_PAREN, "Expect ')' after if condition.")

	thenBranch := p.statement()

	var elseBranch Stmt
	if p.match(ELSE) {
		elseBranch = p.statement()
	}

	return IfStmt{
		Condition: condition,
		Then:      thenBranch,
		Else:      elseBranch,
	}
}

func (p *Parser) printStatement() Stmt {
	value := p.expression()
	p.consume(SEMICOLON, "Expect ';' after value.")
	return PrintStmt{value}
}

func (p *Parser) returnStatement() Stmt {
	keyword := p.previous()
	var value Expr
	if !p._check(SEMICOLON) {
		value = p.expression()
	}
	p.consume(SEMICOLON, "Expect ';' after return statement.")

	return ReturnStmt{
		Keyword: keyword,
		Value:   value,
	}
}

func (p *Parser) whileStatement() Stmt {
	p.consume(LEFT_PAREN, "Expect '(' after 'while'.")
	condition := p.expression()
	p.consume(RIGHT_PAREN, "Expect ')' after while condition.")
	body := p.statement()

	return WhileStmt{
		Condition: condition,
		Body:      body,
	}
}

func (p *Parser) block() []Stmt {
	var stmts []Stmt
	for !p._check(RIGHT_BRACE) && !p.isAtEnd() {
		stmts = append(stmts, p.declaration())
	}
	p.consume(RIGHT_BRACE, "Expect '}' after block.")
	return stmts
}

func (p *Parser) expressionStatement() Stmt {
	expr := p.expression()
	p.consume(SEMICOLON, "Expect ';' after expression.")
	return ExprStmt{expr}
}

func (p *Parser) expression() Expr {
	return p.assignment()
}

func (p *Parser) assignment() Expr {
	expr := p.or()
	if p.match(EQUAL) {
		//equals := p.previous()
		rValue := p.assignment()

		switch lValue := expr.(type) {
		case Variable:
			return Assign{
				Name:  lValue.Name,
				Value: rValue,
			}
		case Get:
			return Set{
				Object: lValue.Object,
				Name:   lValue.Name,
				Value:  rValue,
			}
		default:
			p.parseError(p.previous().Line, "Invalid l-value in assignment.")
		}
	}
	return expr
}

func (p *Parser) or() Expr {
	expr := p.and()

	for p.match(OR) {
		operator := p.previous()
		right := p.and()
		expr = Logical{
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}

	return expr
}

func (p *Parser) and() Expr {
	expr := p.equality()

	for p.match(AND) {
		operator := p.previous()
		right := p.and()
		expr = Logical{
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}
	return expr
}

func (p *Parser) equality() Expr {
	next := func() Expr { return p.comparison() }
	return p._binaryExpr(next, BANG_EQUAL, EQUAL_EQUAL)
}

func (p *Parser) comparison() Expr {
	next := func() Expr { return p.term() }
	return p._binaryExpr(next, GREATER, GREATER_EQUAL, LESS, LESS_EQUAL)
}

func (p *Parser) term() Expr {
	next := func() Expr { return p.factor() }
	return p._binaryExpr(next, MINUS, PLUS)
}

func (p *Parser) factor() Expr {
	next := func() Expr { return p.unary() }
	return p._binaryExpr(next, SLASH, STAR)
}

func (p *Parser) _binaryExpr(next func() Expr, types ...TokenType) Expr {
	expr := next()
	for p.match(types...) {
		operator := p.previous()
		right := next()
		expr = Binary{
			Left:     expr,
			Operator: operator,
			Right:    right,
		}
	}
	return expr
}

func (p *Parser) unary() Expr {
	if p.match(BANG, MINUS) {
		operator := p.previous()
		right := p.unary()
		return Unary{
			Operator: operator,
			Right:    right,
		}
	}
	return p.call()
}

func (p *Parser) call() Expr {
	expr := p.primary()

	for {
		if p.match(LEFT_PAREN) {
			expr = p.finishCall(expr)
		} else if p.match(DOT) {
			name := p.consume(IDENTIFIER, "Expect property name after '.'.")
			expr = Get{expr, name}
		} else {
			break
		}
	}

	return expr
}

func (p *Parser) finishCall(callee Expr) Expr {
	var args []Expr
	if !p._check(RIGHT_PAREN) {
		// keep adding args as long as we find an arg with a
		// trailing comma
		for {
			args = append(args, p.expression())
			if !p.match(COMMA) {
				break
			}
		}
	}
	paren := p.consume(RIGHT_PAREN, "Expect ')' after call arguments.")

	return Call{
		Callee: callee,
		Paren:  paren,
		Args:   args,
	}
}

func (p *Parser) primary() Expr {
	if p.match(FALSE) {
		return Literal{Value: false}
	}
	if p.match(TRUE) {
		return Literal{Value: true}
	}
	if p.match(NIL) {
		return Literal{Value: nil}
	}

	if p.match(NUMBER, STRING) {
		return Literal{Value: p.previous().Literal}
	}

	if p.match(SUPER) {
		keyword := p.previous()
		p.consume(DOT, "Expect '.' after 'super'.")
		method := p.consume(IDENTIFIER, "Expect superclass method name.")
		return Super{
			Keyword: keyword,
			Method:  method,
		}
	}

	if p.match(THIS) {
		return This{p.previous()}
	}

	if p.match(IDENTIFIER) {
		return Variable{Name: p.previous(), Unique: p.nextUniqueVarRef()}
	}

	if p.match(LEFT_PAREN) {
		expr := p.expression()
		p.consume(RIGHT_PAREN, "Expect ')' after expression.")
		return Grouping{
			Expression: expr,
		}
	}
	p.parseError(p.previous().Line, "FIXME: no default case for primary production, and no error handling")
	panic("unreachable")
}

/* Token list operations from here down */
func (p *Parser) advance() Token {
	if !p.isAtEnd() {
		p.current++
	}
	return p.previous()
}

func (p *Parser) previous() Token {
	return p.Tokens[p.current-1]
}

func (p *Parser) peek() Token {
	return p.Tokens[p.current]
}

func (p *Parser) match(types ...TokenType) bool {
	for _, typ := range types {
		if p._check(typ) {
			p.advance()
			return true
		}
	}
	return false
}

func (p *Parser) consume(typ TokenType, errMsg string) Token {
	if p._check(typ) {
		return p.advance()
	}
	p.parseError(p.previous().Line, errMsg)
	return Token{} // unreachable
}

func (p *Parser) _check(typ TokenType) bool {
	if p.isAtEnd() {
		return false
	}
	return p.peek().Type == typ
}

func (p *Parser) isAtEnd() bool {
	return p.current >= len(p.Tokens)
}
