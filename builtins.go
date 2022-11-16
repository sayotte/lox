package main

import "time"

type ClockBuiltin struct{}

func (cb ClockBuiltin) Arity() int { return 0 }

func (cb ClockBuiltin) Call(i *Interpreter, args []interface{}) interface{} {
	return float64(time.Now().Unix())
}
