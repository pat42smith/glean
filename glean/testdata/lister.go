// Copyright 2022 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.
//
// This program is a trivial test case for parsing with glean.  It uses a trivial
// grammar to manipulate a list of integers from the command line arguments.

package main

import (
	"fmt"
	"os"
	"sort"
	"strconv"
)

// The default target sorts the list.
type Sorted []int
type Target Sorted

func RuleSort0() Sorted {
	return nil
}

func RuleSort(s Sorted, i int) Sorted {
	s = append(s, i)
	sort.Ints([]int(s))
	return s
}

func RuleDefault(s Sorted) Target {
	return Target(s)
}

// Alternatively, we can add the list elements.
type Adder int

func RuleAdd0() Adder {
	return 0
}

func RuleAdd(a Adder, i int) Adder {
	return a + Adder(i)
}

func main() {
	var tokens []interface{}
	args := os.Args
	for n := 1; n < len(args); n++ {
		if x, e := strconv.Atoi(args[n]); e != nil {
			panic(e)
		} else {
			tokens = append(tokens, x)
		}
	}

	if result, e := _glean_Parse(tokens); e != nil {
		panic(e)
	} else {
		fmt.Println(result)
	}
}
