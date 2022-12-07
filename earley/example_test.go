// Copyright 2021 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package earley_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pat42smith/glean/earley"
)

// Example uses a parser to add two integers
func Example() {
	tmp, e := os.MkdirTemp("", "")
	if e != nil {
		panic(e)
	}
	defer func() { os.Remove(tmp) }()

	mainGo := filepath.Join(tmp, "main.go")
	e = os.WriteFile(mainGo, []byte(mainText), 0444)
	if e != nil {
		panic(e)
	}
	defer func() { os.Remove(mainGo) }()

	var g earley.Grammar
	e = g.AddRule("RuleAdd", "Sum", []string{"int", "int"})
	if e != nil {
		panic(e)
	}
	parserText, e := g.WriteParser("Sum", "main", "_")
	if e != nil {
		panic(e)
	}

	parserGo := filepath.Join(tmp, "parser.go")
	e = os.WriteFile(parserGo, []byte(parserText), 0444)
	if e != nil {
		panic(e)
	}
	defer func() { os.Remove(parserGo) }()

	out, _ := exec.Command("go", "run", mainGo, parserGo).CombinedOutput()
	fmt.Printf("%s", out)
	// Output: 7
}

var mainText = `
package main

import "fmt"

type Sum int

func RuleAdd(i, j int) Sum {
	return Sum(i + j)
}

func main() {
	sum, e := _Parse([]interface{}{ 2, 5 })
	if e != nil {
		panic(e)
	}
	fmt.Println(sum)
}
`
