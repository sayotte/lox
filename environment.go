package main

import "fmt"

type environment struct {
	envMap      map[string]interface{}
	enclosing   *environment
	interpreter *Interpreter
}

func (e *environment) ensureInit() {
	if e.envMap == nil {
		e.envMap = make(map[string]interface{})
	}
}

func (e *environment) define(name string, value interface{}) {
	e.ensureInit()
	e.envMap[name] = value
}

func (e *environment) assign(name Token, value interface{}) {
	e.ensureInit()
	_, found := e.envMap[name.Lexeme]

	if found {
		e.envMap[name.Lexeme] = value
		return
	}

	if e.enclosing != nil {
		e.enclosing.assign(name, value)
		return
	}

	e.interpreter.runtimeError(name.Line, fmt.Sprintf("Undefined (global) variable %q in assignment.", name.Lexeme))
}

func (e *environment) get(tok Token) interface{} {
	value, found := e.envMap[tok.Lexeme]
	if !found {
		if e.enclosing != nil {
			return e.enclosing.get(tok)
		}
		e.interpreter.runtimeError(tok.Line, fmt.Sprintf("Undefined (global) variable %q.", tok.Lexeme))
	}
	return value
}

func (e *environment) ancestor(distance int) *environment {
	env := e
	for i := 0; i < distance; i++ {
		env = env.enclosing
	}
	return env
}

func (e *environment) assignAt(distance int, name Token, value interface{}) {
	env := e.ancestor(distance)
	env.ensureInit()
	_, found := env.envMap[name.Lexeme]
	if !found {
		e.interpreter.runtimeError(name.Line, fmt.Sprintf("Undefined (local) variable %q in assignment.", name.Lexeme))
	}
	env.envMap[name.Lexeme] = value
}

func (e *environment) getAt(distance int, tok Token) interface{} {
	env := e.ancestor(distance)
	value, found := env.envMap[tok.Lexeme]
	if !found {
		e.interpreter.runtimeError(tok.Line, fmt.Sprintf("Undefined (local) variable %q", tok.Lexeme))
	}
	return value
}
