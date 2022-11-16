package main

import "fmt"

type Class struct {
	Name       string
	Methods    map[string]Function
	Superclass *Class
}

func (c Class) String() string {
	return c.Name
}

func (c Class) Call(i *Interpreter, args []interface{}) interface{} {
	inst := &Instance{
		Class: c,
	}
	initializer, found := c.findMethod("init")
	if found {
		initializer.bindMethodToInstance(inst).Call(i, args)
	}
	return inst
}

func (c Class) Arity() int {
	initializer, found := c.findMethod("init")
	if !found {
		return 0
	}
	return initializer.Arity()
}

func (c Class) findMethod(name string) (Function, bool) {
	method, found := c.Methods[name]
	if found {
		return method, found
	}
	if c.Superclass != nil {
		method, found = c.Superclass.findMethod(name)
	}
	return method, found
}

type Instance struct {
	Class  Class
	Fields map[string]interface{}
}

func (i Instance) String() string {
	return i.Class.Name + " instance"
}

func (i *Instance) Get(name Token) (interface{}, error) {
	field, found := i.Fields[name.Lexeme]
	if found {
		return field, nil
	}

	method, found := i.Class.findMethod(name.Lexeme)
	if found {
		return method.bindMethodToInstance(i), nil
	}

	return nil, fmt.Errorf("Undefined property %q.", name.Lexeme)
}

func (i *Instance) Set(name Token, value interface{}) {
	if i.Fields == nil {
		i.Fields = make(map[string]interface{})
	}
	i.Fields[name.Lexeme] = value
}
