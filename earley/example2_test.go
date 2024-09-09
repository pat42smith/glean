// Copyright 2021-2024 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package earley_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pat42smith/glean"
	"github.com/pat42smith/glean/earley"
)

// A Grammar may be reused:
// 1. Add some rules.
// 2. Write a parser for those rules.
// 3. Add some more rules.
// 4. Write a second parser that handles both sets of rules.
func Example_Reuse() {
	tmp, e := os.MkdirTemp("", "")
	if e != nil {
		panic(e)
	}
	defer func() { os.Remove(tmp) }()

	mainGo := filepath.Join(tmp, "main.go")
	e = os.WriteFile(mainGo, []byte(mainReuseText), 0444)
	if e != nil {
		panic(e)
	}
	defer func() { os.Remove(mainGo) }()

	var g earley.Grammar
	e = g.AddRule("RuleInt", "Sum", []glean.Symbol{"int"})
	if e != nil {
		panic(e)
	}
	e = g.AddRule("RuleAdd", "Sum", []glean.Symbol{"Sum", "Plus", "int"})
	if e != nil {
		panic(e)
	}
	parser1Text, e := g.WriteParser("Sum", "main", "_1_")
	if e != nil {
		panic(e)
	}

	parser1Go := filepath.Join(tmp, "parser1.go")
	e = os.WriteFile(parser1Go, []byte(parser1Text), 0444)
	if e != nil {
		panic(e)
	}
	defer func() { os.Remove(parser1Go) }()

	e = g.AddRule("RuleSubtract", "Sum", []glean.Symbol{"Sum", "Minus", "int"})
	if e != nil {
		panic(e)
	}
	parser2Text, e := g.WriteParser("Sum", "main", "_2_")
	if e != nil {
		panic(e)
	}

	parser2Go := filepath.Join(tmp, "parser2.go")
	e = os.WriteFile(parser2Go, []byte(parser2Text), 0444)
	if e != nil {
		panic(e)
	}
	defer func() { os.Remove(parser2Go) }()

	out, _ := exec.Command("go", "run", mainGo, parser1Go, parser2Go).CombinedOutput()
	fmt.Printf("%s", out)
	// Output:
	// 17
	// -10
}

var mainReuseText = `
package main

import "fmt"

type Sum int
type Plus struct {}
type Minus struct {}

func RuleInt(i int) Sum {
	return Sum(i)
}

func RuleAdd(i Sum, _ Plus, j int) Sum {
	return i + Sum(j)
}

func RuleSubtract(i Sum, _ Minus, j int) Sum {
	return i - Sum(j)
}

func main() {
	sum, e := _1_Parse([]interface{}{ 9, Plus{}, 8 })
	if e != nil {
		panic(e)
	}
	fmt.Println(sum)

	sum, e = _2_Parse([]interface{}{ 7, Minus{}, 20, Plus{}, 3 })
	if e != nil {
		panic(e)
	}
	fmt.Println(sum)
}
`
