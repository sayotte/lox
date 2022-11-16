package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestResolver_Resolve_script(t *testing.T) {
	testCases := map[string]struct {
		in          string
		errExpected bool
		expectedErr string
	}{
		"error on redeclare in same scope": {
			in:          "fun bad(){ var a = 1; var a = 2; }",
			errExpected: true,
			expectedErr: "already a variable with name",
		},
		"no error returning without value from top level": {
			in:          "return;",
			errExpected: false,
		},
		"error on returning value from top level": {
			in:          "return 1;",
			errExpected: true,
			expectedErr: "can't return a value from top-level",
		},
		"error on unused local variable": {
			in:          "fun foo(){var a = 1;}",
			errExpected: true,
			expectedErr: "unused local variable",
		},
		"can't use 'this' outside class method": {
			in:          "fun foo(){ print this; }",
			errExpected: true,
			expectedErr: "Cannot use 'this' outside of a class method",
		},
		"can't return a value from init()": {
			in:          "class foo{init(){return \"value\";}}",
			errExpected: true,
			expectedErr: "can't return a value from an initializer",
		},
		"class can't inherit from itself": {
			in:          "class foo < foo {}",
			errExpected: true,
			expectedErr: "class can't inherit from itself",
		},
		"super can't be used outside of a class": {
			in:          "print super.foo();",
			errExpected: true,
			expectedErr: "Can't use 'super' outside of a class",
		},
		"Super can't be used in a top-level class": {
			in:          "class busted { foo(){ return super.foo(); } }",
			errExpected: true,
			expectedErr: "Can't use 'super' in a class with no superclass",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tokens := (&Scanner{}).ScanTokens(tc.in)
			stmts, parseErr := (&Parser{Tokens: tokens}).Parse()
			if parseErr != nil {
				t.Fatalf("parsing error in test input: %s", parseErr)
			}
			interpreter := &Interpreter{}
			interpreter.init()
			resolver := &Resolver{interpreter: interpreter}
			resolveErr := resolver.Resolve(stmts)
			if !tc.errExpected && resolveErr != nil {
				t.Errorf("unexpected error: %s", resolveErr)
			}
			if tc.errExpected && resolveErr == nil {
				t.Error("expected error, didn't get one")
			}
			if tc.errExpected && resolveErr != nil && !strings.Contains(resolveErr.Error(), tc.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tc.expectedErr, resolveErr)
			}
		})
	}
}

func TestResolver_Resolve_AST(t *testing.T) {
	// Create the AST from the bottom up, so we can refer
	//   to its expressions in the tests.
	// Our overall AST will be for this script:
	//   for (var i = 0; i < 3; i = i + 1) {
	//     print i;
	//   }
	// In Lox for-loops are syntactic sugar for while-loops,
	// so this actually looks like this:
	//   {
	//     var i = 0;
	//     while (i < 3) {
	//       { // <-- note the extra block here!
	//         print i;
	//       }
	//       i = i + 1;
	//     }
	//   }
	//
	// We have 4 references to "i" that we'll need to test:
	// #1 in the conditional, "i < 3"
	iRefTokLine1 := Token{Type: IDENTIFIER, Lexeme: "i", Line: 1}
	whileCondLeftVar := Variable{Name: iRefTokLine1, Unique: 0}
	// #2 in the print statement, "print i;"
	printStmtVar := Variable{Name: Token{Type: IDENTIFIER, Lexeme: "i", Line: 2}}
	// #3 on the right side of the incrementor, "i = i + 1"
	bodyIncrementExprRightVar := Variable{Name: iRefTokLine1, Unique: 1}
	// #4 on the left side of the incrementor, "i = i + 1"
	bodyIncrementExpr := Assign{
		Name: iRefTokLine1,
		Value: Binary{
			Left:     bodyIncrementExprRightVar,
			Operator: Token{Type: PLUS, Lexeme: "+", Line: 1},
			Right:    Literal{Value: float64(1)},
		},
	}
	// Now we build the overall structure. Remember it looks like this:
	//   {
	//     var i = 0;
	//     while (i < 3) {
	//       { // <-- note the extra block here!
	//         print i;
	//       }
	//       i = i + 1;
	//     }
	//   }
	outerBlock := BlockStmt{Statements: []Stmt{
		VariableStmt{Name: iRefTokLine1, Initializer: Literal{Value: float64(0)}},
		WhileStmt{
			Condition: Binary{
				Left:     whileCondLeftVar,
				Operator: Token{Type: LESS, Lexeme: "<", Line: 1},
				Right:    Literal{Value: float64(3)},
			},
			Body: BlockStmt{Statements: []Stmt{
				BlockStmt{Statements: []Stmt{
					PrintStmt{Expression: printStmtVar},
				}},
				ExprStmt{Expression: bodyIncrementExpr},
			}},
		},
	}}

	interpreter := &Interpreter{}
	resolver := &Resolver{interpreter: interpreter}
	err := resolver.Resolve([]Stmt{outerBlock})
	if err != nil {
		t.Errorf("unexpected error: %s", err)
	}
	expected := map[Expr]int{
		whileCondLeftVar:          0,
		printStmtVar:              2,
		bodyIncrementExprRightVar: 1,
		bodyIncrementExpr:         1,
	}
	if !reflect.DeepEqual(interpreter.localDistance, expected) {
		t.Errorf("%v != %v", interpreter.localDistance, expected)
	}
}
