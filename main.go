package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
)

func main() {
	if len(os.Args) > 2 {
		fmt.Println("Usage: glox [script]")
		os.Exit(64)
	}

	l := NewLox(os.Stdout)
	if len(os.Args) == 2 {
		l.runFile(os.Args[1])
	} else {
		l.runPrompt()
	}

}

type Lox struct {
	interpreter *Interpreter
	hadError    bool
}

func NewLox(stdout io.Writer) *Lox {
	return &Lox{
		interpreter: &Interpreter{Stdout: stdout},
	}
}

func (l *Lox) runFile(path string) {
	fBytes, err := ioutil.ReadFile(path)
	if err != nil {
		panic(err)
	}
	l.run(string(fBytes))
	if l.hadError {
		panic("error interpreting file")
	}
}

func (l *Lox) runPrompt() {
	lineReader := bufio.NewScanner(os.Stdin)
	for lineReader.Scan() {
		l.run(lineReader.Text())
		l.hadError = false
	}
	if err := lineReader.Err(); err != nil {
		panic(err)
	}
}

func (l *Lox) run(src string) {
	tokens := (&Scanner{}).ScanTokens(src)
	stmts, err := (&Parser{Tokens: tokens}).Parse()
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}
	resolver := &Resolver{interpreter: l.interpreter}
	err = resolver.Resolve(stmts)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}
	err = l.interpreter.Interpret(stmts)
	if err != nil {
		fmt.Printf("ERROR: %s\n", err)
		return
	}
}

func (l *Lox) srcError(line int, message string) {
	_, _ = fmt.Fprintf(os.Stderr, "[line %d] Error: %s\n", line, message)
	l.hadError = true
}
