package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestInterpreter_Interpret_stmts(t *testing.T) {
	stmtTestCases := map[string]struct {
		in          []Stmt
		expected    string
		errExpected bool
		expectedErr string
	}{
		"number literal": {
			in:       []Stmt{PrintStmt{Literal{Value: 10.0}}},
			expected: "10\n",
		},
		"grouped literal": {
			in: []Stmt{PrintStmt{Grouping{
				Expression: Literal{Value: 1.0},
			}}},
			expected: "1\n",
		},
		"numeric addition": {
			in: []Stmt{PrintStmt{Binary{
				Operator: Token{Type: PLUS},
				Left:     Literal{Value: 1.0},
				Right:    Literal{Value: 1.0},
			}}},
			expected: "2\n",
		},
		"string concat": {
			in: []Stmt{PrintStmt{Binary{
				Operator: Token{Type: PLUS},
				Left:     Literal{Value: "one"},
				Right:    Literal{Value: "two"},
			}}},
			expected: "onetwo\n",
		},
		"err: number + string": {
			in: []Stmt{PrintStmt{Binary{
				Operator: Token{Type: PLUS},
				Left:     Literal{Value: 1.0},
				Right:    Literal{Value: "two"},
			}}},
			errExpected: true,
			expectedErr: "must be numbers",
		},
		"err: bool + number": {
			in: []Stmt{PrintStmt{Binary{
				Operator: Token{Type: PLUS},
				Left:     Literal{Value: false},
				Right:    Literal{Value: 1.0},
			}}},
			errExpected: true,
			expectedErr: "can operate on numbers or strings",
		},
		"equality: identical": {
			in: []Stmt{PrintStmt{Binary{
				Operator: Token{Type: EQUAL_EQUAL},
				Left:     Literal{Value: 1.0},
				Right:    Literal{Value: 1.0},
			}}},
			expected: "true\n",
		},
		"equality: different types": {
			in: []Stmt{PrintStmt{Binary{
				Operator: Token{Type: EQUAL_EQUAL},
				Left:     Literal{Value: 1.0},
				Right:    Literal{Value: "one"},
			}}},
			expected: "false\n",
		},
		"unary minus": {
			in: []Stmt{PrintStmt{Unary{
				Operator: Token{Type: MINUS},
				Right:    Literal{Value: 1.0},
			}}},
			expected: "-1\n",
		},
		"unary bang": {
			in: []Stmt{PrintStmt{Unary{
				Operator: Token{Type: BANG},
				Right:    Literal{Value: true},
			}}},
			expected: "false\n",
		},
		"kitchen sink": {
			// !(1 + 1) == 2 * 2 > 3
			in: []Stmt{PrintStmt{Binary{
				Left: Unary{
					Operator: Token{Type: BANG},
					Right: Grouping{
						Expression: Binary{
							Left:     Literal{Value: 1.0},
							Operator: Token{Type: PLUS},
							Right:    Literal{Value: 1.1},
						},
					},
				},
				Operator: Token{Type: EQUAL_EQUAL},
				Right: Binary{
					Left: Binary{
						Left:     Literal{Value: 2.0},
						Operator: Token{Type: STAR},
						Right:    Literal{Value: 2.1},
					},
					Operator: Token{Type: GREATER},
					Right:    Literal{3.0},
				},
			}}},
			expected: "false\n",
		},
		"var assignment and reassignment": {
			in: []Stmt{
				VariableStmt{
					Name:        Token{Lexeme: "a"},
					Initializer: Literal{Value: "1"},
				},
				PrintStmt{
					Expression: Variable{
						Name: Token{Lexeme: "a"},
					},
				},
				ExprStmt{
					Expression: Assign{
						Name:  Token{Lexeme: "a"},
						Value: Literal{"2"},
					},
				},
				PrintStmt{
					Expression: Variable{
						Name: Token{Lexeme: "a"},
					},
				},
			},
			expected: "1\n2\n",
		},
	}

	for name, tc := range stmtTestCases {
		t.Run(name, func(t *testing.T) {
			out := &bytes.Buffer{}
			i := &Interpreter{Stdout: out}
			err := i.Interpret(tc.in)
			actual := out.String()
			if tc.errExpected && err == nil {
				t.Error("err expected, didn't get one")
			} else if !tc.errExpected && err != nil {
				t.Errorf("unexpected error: %s", err)
			} else if err != nil && !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tc.expectedErr, err)
			} else if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestInterpreter_Interpret_script(t *testing.T) {
	testCases := map[string]struct {
		in          string
		expected    string
		errExpected bool
		expectedErr string
	}{
		"block scope": {
			in: `
var a = "global a";
var b = "global b";
var c = "global c";
{
    var a = "outer a";
    var b = "outer b";
    {
        var a = "inner a";
        print a;
        print b;
        print c;
    }
    print a;
    print b;
    print c;
}
print a;
print b;
print c;`,
			expected: "inner a\nouter b\nglobal c\nouter a\nouter b\nglobal c\nglobal a\nglobal b\nglobal c\n",
		},
		"if true": {
			in:       "var a = true; if (a) print \"yes\";",
			expected: "yes\n",
		},
		"else true": {
			in:       "var a = false; if (a) print \"yes\"; else print \"no\";",
			expected: "no\n",
		},
		"true or false": {
			in:       "print true or false;",
			expected: "true\n",
		},
		"true and false": {
			in:       "print true and false;",
			expected: "false\n",
		},
		"trivial for loop": {
			in: `for(var i = 0; i < 3; i = i + 1){
				print i;}`,
			expected: "0\n1\n2\n",
		},
		"recursion with return": {
			in:       "fun fib(n){ if(n<=1) return n; return fib(n-2)+fib(n-1); } print fib(10);",
			expected: "55\n",
		},
		"closures": {
			in:       "fun makeCounter(){ var i=0; fun count(){i=i+1; print i;} return count;} var counter=makeCounter(); counter(); counter();",
			expected: "1\n2\n",
		},
		"scope is static": {
			in: `
var a = "global";
{ 
  fun showA(){
    print a;
  }
  showA();
  var a = "block";
  showA();
  print a; // to suppress unused variable error from resolver
}
`,
			expected: "global\nglobal\nblock\n",
		},
		"class with fields": {
			in: `
class Cake {
  exclaim() {
    return "Hooray, cake!";
  }
}

print Cake;

// can create instance
var c = Cake();
print c;

// can set/get fields
c.foo = "foo";
print c.foo;

// can call a method with no "this" binding needed
print c.exclaim();`,
			expected: "Cake\nCake instance\nfoo\nHooray, cake!\n",
		},
		"class with methods": {
			in: `
class Sammy {
  init(flavor) { this.flavor = flavor; }
  describe() { return "A delicious "+this.flavor+" sandwich."; }
}
var sammy = Sammy("turkey");
print sammy.describe();
// methods are first-class objects, bound to an instance
var x = sammy.describe;
print x();
`,
			expected: "A delicious turkey sandwich.\nA delicious turkey sandwich.\n",
		},
		"implicit+explicit calls to init() all return the instance": {
			in: `
class foo {
  init(myParam) {
    this.myParam = myParam;
  }
  getMyParam() {
    return this.myParam;
  }
}

print foo("foo");
var ie = foo("foo");
print ie;
print ie.init("bar");
`,
			expected: "foo instance\nfoo instance\nfoo instance\n",
		},
		"superclass must be a class": {
			in:          "var foo = 0; class bar < foo {}",
			errExpected: true,
			expectedErr: "Superclass must be a class",
		},
		"inherited methods work": {
			in: `
class foo {
  blah(){ return "foo level blah"; }
}
class bar < foo {}
var x = bar();
print x.blah();
`,
			expected: "foo level blah\n",
		},
		"super methods work": {
			in: `
class bread {
  str(){ return "bread"; }
}
class donut < bread {
  str(){ return super.str() + ", donut"; }
}
class kruller < donut{}
var k = kruller();
print k.str();
`,
			expected: "bread, donut\n",
		},
	}
	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tokens := (&Scanner{}).ScanTokens(tc.in)
			stmts, parseErr := (&Parser{Tokens: tokens}).Parse()
			if parseErr != nil {
				t.Fatalf("parsing error in test input: %s", parseErr)
			}
			out := &bytes.Buffer{}
			interpreter := &Interpreter{Stdout: out}
			interpreter.init()
			resolver := &Resolver{interpreter: interpreter}
			resolveErr := resolver.Resolve(stmts)
			if resolveErr != nil {
				t.Fatalf("resolution error in test input: %s", resolveErr)
			}
			//for key, val := range interpreter.localDistance {
			//	fmt.Printf("%q: %d\n", key, val)
			//}
			err := interpreter.Interpret(stmts)
			actual := out.String()
			if tc.errExpected && err == nil {
				t.Error("expected error, didn't get one")
			} else if !tc.errExpected && err != nil {
				t.Errorf("unexpected error: %s", err)
			} else if err != nil && !strings.Contains(err.Error(), tc.expectedErr) {
				t.Errorf("expected error containing %q, got %q", tc.expectedErr, err)
			} else if actual != tc.expected {
				t.Errorf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}
