package main

import "fmt"

type Function struct {
	Declaration   FunctionStmt
	Closure       *environment
	isInitializer bool // allows us to return "this" from a re-call to init()
}

func (f Function) bindMethodToInstance(inst *Instance) Function {
	env := &environment{enclosing: f.Closure}
	env.define("this", inst)
	return Function{
		Declaration:   f.Declaration,
		Closure:       env,
		isInitializer: f.isInitializer,
	}
}

func (f Function) Arity() int {
	return len(f.Declaration.Params)
}

func (f Function) Call(i *Interpreter, params []interface{}) (returnVal interface{}) {
	newEnv := &environment{
		// note that call semantics mean we can't see variables
		// in the caller's scope, only globals
		enclosing:   f.Closure,
		interpreter: i,
	}
	for i, param := range params {
		newEnv.define(f.Declaration.Params[i].Lexeme, param)
	}

	// We unwind the interpreter's internal call stack when it
	// hits a "return" statement by throwing an exception/panicking.
	// To harvest the returned value, we pack it into the panic()
	// parameter and catch it here.
	//
	// This seems janky.
	defer func() {
		r := recover()
		if r == nil {
			return
		}
		returned, ok := r.(returnable)
		if !ok {
			panic(r)
		}
		returnVal = returned.Value
	}()

	i.executeBlock(f.Declaration.Body, newEnv)

	if f.isInitializer {
		return f.Closure.getAt(0, Token{Lexeme: "this"})
	}

	return nil
}

func (f Function) String() string {
	return fmt.Sprintf("<fn %s>", f.Declaration.Name.Lexeme)
}
