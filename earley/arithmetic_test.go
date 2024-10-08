// Copyright 2022-2024 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package earley_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/pat42smith/glean"
	"github.com/pat42smith/glean/earley"
	"github.com/pat42smith/or"
)

// Test with integer arithmetic
func TestArithmetic(t *testing.T) {
	tmp := t.TempDir()

	mainGo := filepath.Join(tmp, "main.go")
	if e := os.WriteFile(mainGo, []byte(arithmeticMainText), 0444); e != nil {
		t.Fatal(e)
	}

	// Check that *earley.Grammar can be converted to glean.Grammar
	var grammar glean.Grammar = new(earley.Grammar)

	grammar.AddRule("RuleSum", "Sum", []glean.Symbol{"Product"})
	grammar.AddRule("RuleAdd", "Sum", []glean.Symbol{"Sum", "Plus", "Product"})
	grammar.AddRule("RuleSubtract", "Sum", []glean.Symbol{"Sum", "Minus", "Product"})
	grammar.AddRule("RuleProduct", "Product", []glean.Symbol{"Item"})
	grammar.AddRule("RuleMultiply", "Product", []glean.Symbol{"Product", "Times", "Item"})
	grammar.AddRule("RuleDivide", "Product", []glean.Symbol{"Product", "Divide", "Item"})
	grammar.AddRule("RuleParenthesis", "Item", []glean.Symbol{"Open", "Sum", "Close"})
	grammar.AddRule("RuleItem", "Item", []glean.Symbol{"Int"})
	parserText, e := grammar.WriteParser("Sum", "main", "_arith")
	or.Fatal0(e)(t)

	parserGo := filepath.Join(tmp, "parser.go")
	if e = os.WriteFile(parserGo, []byte(parserText), 0444); e != nil {
		t.Fatal(e)
	}

	for _, test := range testdata {
		ans := strconv.Itoa(test.answer)
		tokens := strings.Split(test.expr, " ")
		args := []string{"run", mainGo, parserGo}
		args = append(args, tokens...)
		got, e := exec.Command("go", args...).CombinedOutput()
		or.Fatal0(e)(t)
		if string(got) != ans+"\n" {
			t.Errorf("wrong answer %s for %v", got, test)
		}
	}

	gofmt, e := exec.LookPath("gofmt")
	or.Fatal0(e)(t)
	got, e := exec.Command(gofmt, "-d", parserGo).CombinedOutput()
	or.Fatal0(e)(t)
	if len(got) > 0 {
		t.Errorf("formatting differs from gofmt standard:\n%s", got)
	}
}

var arithmeticMainText = `
package main

import (
	"fmt"
	"os"
	"strconv"
)

type Int int
type Item int
type Product int
type Sum int
type Plus struct {}
type Minus struct {}
type Times struct {}
type Divide struct {}
type Open struct {}
type Close struct {}

func RuleSum(i Product) Sum { return Sum(i) }
func RuleAdd(i Sum, _ Plus, j Product) Sum { return i + Sum(j) }
func RuleSubtract(i Sum, _ Minus, j Product) Sum { return i - Sum(j) }
func RuleProduct(i Item) Product { return Product(i) }
func RuleMultiply(i Product, _ Times, j Item) Product { return i * Product(j) }
func RuleDivide(i Product, _ Divide, j Item) Product { return i / Product(j) }
func RuleParenthesis(_ Open, i Sum, _ Close) Item { return Item(i) }
func RuleItem(i Int) Item { return Item(i) }

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
		case "-":
			tokens[n] = Minus{}
		case "*":
			tokens[n] = Times{}
		case "/":
			tokens[n] = Divide{}
		case "(":
			tokens[n] = Open{}
		case ")":
			tokens[n] = Close{}
		default:
			i, e := strconv.Atoi(a)
			if e != nil {
				panic(e)
			}
			tokens[n] = Int(i)
		}
	}

	n, e := _arithParse(tokens)
	if e != nil {
		panic(e)
	}
	fmt.Println(n)
}
`

var testdata = []struct {
	answer int
	expr   string
}{
	{5, "5"},
	{3, "9 / 3"},
	{15, "( 2 + 1 ) * ( 7 - 2 )"},
	{17, "( ( ( ( ( ( ( ( ( 17 ) ) ) ) ) ) ) ) )"},
	{24, "1 * ( 1 + 1 ) * 3 * ( 3 + 1 )"},
	{7, "1 + 2 * 3"},
	{5, "1 * 2 + 3"},
}
