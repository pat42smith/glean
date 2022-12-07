// Copyright 2022 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package earley_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/pat42smith/glean/earley"
)

// Test for various parse errors
func TestParseErrors(t *testing.T) {
	tmp := t.TempDir()

	mainGo := filepath.Join(tmp, "main.go")
	e := os.WriteFile(mainGo, []byte(pmainText), 0444)
	if e != nil {
		panic(e)
	}

	var g earley.Grammar
	addrule := func(name, target string, items ...string) {
		if e := g.AddRule(name, target, items); e != nil {
			panic(e)
		}
	}
	addrule("RuleExpr", "Goal", "Expr")
	addrule("RuleInt", "Expr", "int")
	addrule("RuleAdd", "Expr", "Expr", "Plus", "Expr")
	addrule("RuleOpenClose", "Goal", "Open", "Close")
	addrule("RulePair", "Goal", "Pair")
	addrule("RuleMakePair", "Pair", "Open", "Close")
	addrule("RuleParens", "Goal", "Open", "Goal", "Close")
	addrule("RuleNil", "Nil")
	addrule("RuleNull", "Null")
	addrule("RuleNil0", "Nothing", "Nil")
	addrule("RuleNull0", "Nothing", "Null")
	addrule("RuleNothing", "Goal", "Plus", "Nothing", "Open", "Close")
	addrule("RuleBlank", "Blank")
	addrule("RuleUnderscore", "Blank", "Underscore")
	addrule("RuleBlank2", "Blank", "Blank", "Blank")
	addrule("RuleInfinite", "Goal", "Blank", "int", "Open", "Close")

	parserText, e := g.WriteParser("Goal", "main", "_")
	if e != nil {
		panic(e)
	}

	parserGo := filepath.Join(tmp, "parser.go")
	e = os.WriteFile(parserGo, []byte(parserText), 0444)
	if e != nil {
		panic(e)
	}
	//os.WriteFile("/tmp/parser.go", []byte(parserText), 0444)

	run := func(t2 *testing.T, args ...string) (string, error) {
		allargs := append([]string{"run", mainGo, parserGo}, args...)
		out, e := exec.Command("go", allargs...).CombinedOutput()
		return string(out), e
	}

	runok := func(t2 *testing.T, args ...string) string {
		out, e := run(t2, args...)
		if e != nil {
			t2.Fatalf("command failed with output: %s", out)
		}
		return out
	}

	try := func(t2 *testing.T, expect string, args ...string) {
		out := runok(t2, args...)
		if out != expect+"\n" {
			t2.Errorf("wrong error message:\nexpected:\n%s\ngot:\n%s", expect, out)
		}
	}

	ambiguity := func(t2 *testing.T,
		rule1, rule2 string,
		target string, items1, items2 []string,
		where1 int, token1 string, where2 int, token2 string,
		args ...string) {
		f :=
			`gleanerrors.Ambiguous{Range:gleanerrors.Range{First:gleanerrors.Location{Index:%d, Token:%s}, Last:gleanerrors.Location{Index:%d, Token:%s}}, Rule1:gleanerrors.Rule{Name:"%s", Target:"%s", Items:%#v}, Rule2:gleanerrors.Rule{Name:"%s", Target:"%s", Items:%#v}}
ambiguous match for %s
   %s: %s
or %s: %s
`
		i1 := strings.Join(items1, " ")
		i2 := strings.Join(items2, " ")
		expect1 := fmt.Sprintf(f, where1, token1, where2, token2,
			rule1, target, items1, rule2, target, items2,
			target, rule1, i1, rule2, i2)
		expect2 := fmt.Sprintf(f, where1, token1, where2, token2,
			rule2, target, items2, rule1, target, items1,
			target, rule2, i2, rule1, i1)

		out := runok(t2, args...)
		if out != expect1 && out != expect2 {
			t2.Errorf("wrong error message:\nexpected:\n%s\ngot:\n%s", expect1, out)
		}
	}

	t.Run("NoInput", func(t2 *testing.T) {
		try(t2, "gleanerrors.NoInput{}\nno tokens in parser input")
	})

	t.Run("Unexpected", func(t2 *testing.T) {
		try(t2, "gleanerrors.Unexpected{Location:gleanerrors.Location{Index:1, Token:17}}\nunexpected token: 17", "3", "17")
	})

	t.Run("Incomplete", func(t2 *testing.T) {
		try(t2, "gleanerrors.Unexpected{Location:gleanerrors.Location{Index:2, Token:interface {}(nil)}}\nunexpected end of input", "100", "+")
	})

	t.Run("BadToken", func(t2 *testing.T) {
		out, e := run(t2, "@")
		if e == nil {
			t2.Fatalf("parser succeeded when it should have panicked")
		}
		if matched, _ := regexp.MatchString("panic.*type int16", out); !matched {
			t2.Error("incorrect output:", out)
		}
	})

	t.Run("Ambiguous1", func(t2 *testing.T) {
		ambiguity(t2, "RuleAdd", "RuleAdd",
			"Expr", []string{"Expr", "Plus", "Expr"}, []string{"Expr", "Plus", "Expr"},
			0, "2", 4, "5",
			"2", "+", "3", "+", "5")
	})

	t.Run("Ambiguous2", func(t2 *testing.T) {
		ambiguity(t2, "RuleOpenClose", "RulePair",
			"Goal", []string{"Open", "Close"}, []string{"Pair"},
			0, "main.Open{}", 1, "main.Close{}",
			"(", ")")
	})

	t.Run("Ambiguous3", func(t2 *testing.T) {
		ambiguity(t2, "RuleOpenClose", "RulePair",
			"Goal", []string{"Open", "Close"}, []string{"Pair"},
			1, "main.Open{}", 2, "main.Close{}",
			"(", "(", ")", ")")
	})

	t.Run("Ambiguous4", func(t2 *testing.T) {
		ambiguity(t2, "RuleNull0", "RuleNil0",
			"Nothing", []string{"Null"}, []string{"Nil"},
			1, "main.Open{}", 0, "main.Plus{}",
			"+", "(", ")")
	})

	t.Run("Ambiguous5", func(t2 *testing.T) {
		ambiguity(t2, "RuleBlank", "RuleBlank2",
			"Blank", []string{}, []string{"Blank", "Blank"},
			0, "99", -1, "interface {}(nil)",
			"99", "(", ")")
	})
}

var pmainText = `
package main

import (
	"fmt"
	"os"
	"strconv"
)

type Goal int
type Expr int
type Plus struct{}
type Open struct{}
type Close struct{}
type Pair struct{}
type Nil struct{}
type Null struct{}
type Nothing struct{}
type Blank struct{}
type Underscore struct{}

func RuleExpr(e Expr) Goal {
	return Goal(e)
}

func RuleInt(i int) Expr {
	return Expr(i)
}

func RuleAdd(i Expr, _ Plus, j Expr) Expr {
	return i + j
}

func RuleOpenClose(Open, Close) Goal {
	return Goal(0)
}

func RulePair(Pair) Goal {
	return Goal(1)
}

func RuleMakePair(Open, Close) Pair {
	return Pair{}
}

func RuleParens(Open, Goal, Close) Goal {
	return Goal(2)
}

func RuleNil() Nil {
	return Nil{}
}

func RuleNull() Null {
	return Null{}
}

func RuleNil0(Nil) Nothing {
	return Nothing{}
}

func RuleNull0(Null) Nothing {
	return Nothing{}
}

func RuleNothing(Plus, Nothing, Open, Close) Goal {
	return Goal(3)
}

func RuleBlank() Blank {
	return Blank{}
}

func RuleUnderscore(Underscore) Blank {
	return Blank{}
}

func RuleBlank2(Blank, Blank) Blank {
	return Blank{}
}

func RuleInfinite(Blank, int, Open, Close) Goal {
	return Goal(4)
}

func main() {
	args := os.Args
	if len(args) > 0 {
		args = args[1:]
	}

	tokens := make([]interface{}, len(args))
	for n, a := range args {
		switch a {
		case "+":
			tokens[n] = Plus{}
		case "(":
			tokens[n] = Open{}
		case ")":
			tokens[n] = Close{}
		case "@":
			tokens[n] = int16('a')
		case "_":
			tokens[n] = Underscore{}
		default:
			var e error
			tokens[n], e = strconv.Atoi(a)
			if e != nil {
				panic(e)
			}
		}
	}

	_, e := _Parse(tokens)
	if e == nil {
		panic("no error")
	}
	fmt.Printf("%#v\n", e)
	fmt.Println(e)
}
`
