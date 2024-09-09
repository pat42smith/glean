// Copyright 2024 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package glean

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// ruleStringer collects rules found in scanning source files,
// so they can be checked against the rules we expect.
type ruleStringer []string

func (r *ruleStringer) AddRule(name string, target Symbol, items []Symbol) error {
	for _, s := range *r {
		if strings.HasPrefix(s, name+" ") {
			panic("duplicate rule name")
		}
	}
	*r = append(*r, fmt.Sprint(name, " ", target, " ", items))
	return nil
}

func (r *ruleStringer) String() string {
	sort.Strings(*r)
	return strings.Join(*r, "\n")
}

func writeFile(name, data string) {
	e := os.WriteFile(name, []byte(data), 0444)
	if e != nil {
		panic(e)
	}
}

func expectNoWarnings(t *testing.T, warnings []error, err error) {
	if err != nil {
		t.Error(err)
	}
	if len(warnings) != 0 {
		t.Error(warnings)
	}
}

func expectWarnings(t *testing.T, warnings []error, expect ...string) {
	ws := make([]string, len(warnings))
	for n, w := range warnings {
		ws[n] = w.Error()
	}

	ok := true
	if len(warnings) != len(expect) {
		t.Error("Expected", len(expect), "warnings; got", len(warnings))
		ok = false
	}
	for _, e := range expect {
		found := false
		for _, w := range ws {
			if strings.Index(w, "warning: "+e) >= 0 {
				found = true
				break
			}
		}
		if !found {
			t.Error("Did not get expected warning:", e)
			ok = false
		}
	}

	if !ok {
		t.Error("Warnings received:", ws)
	}
}

func expectPackage(t *testing.T, got, expected string) {
	if got != expected {
		t.Error("Expected package name", expected, "but got", got)
	}
}

func expectGrammar(t *testing.T, got *ruleStringer, expected string) {
	s := got.String()
	if s != expected {
		t.Error("Incorrect grammar returned:\n" + s)
	}
}

func TestBasic(t *testing.T) {
	tmp := t.TempDir()
	f := tmp + "/foo.go"
	writeFile(f, `package foo
func RuleAdd(Expr, Plus, Expr) Expr { return nil }
func NotARule(Tiger, Lion) Liger { return nil }
`)

	var rs ruleStringer
	p, w, e := ScanFiles(&rs, f)
	expectNoWarnings(t, w, e)
	expectPackage(t, p, "foo")
	expectGrammar(t, &rs, "RuleAdd Expr [Expr Plus Expr]")

	rs = nil
	p, w, e = ScanDir(&rs, tmp)
	expectNoWarnings(t, w, e)
	expectPackage(t, p, "foo")
	expectGrammar(t, &rs, "RuleAdd Expr [Expr Plus Expr]")
}

func TestShortNames(t *testing.T) {
	tmp := t.TempDir()
	f := tmp + "/foo.go"
	writeFile(f, `package bar
func rule() Exprs
func Rule(list Exprs, extra Expr) Exprs
`)

	var rs ruleStringer
	p, w, e := ScanFiles(&rs, f)
	expectNoWarnings(t, w, e)
	expectPackage(t, p, "bar")
	expectGrammar(t, &rs, "Rule Exprs [Exprs Expr]\nrule Exprs []")
}

func TestMultipleFiles(t *testing.T) {
	tmp := t.TempDir()
	f1 := tmp + "/alpha.go"
	f2 := tmp + "/beta.go"
	f3 := tmp + "/gamma.go"

	writeFile(f1, `package greek
func RuleIncrement(int, PlusPlus) int
`)
	writeFile(f2, `package greek
func RuleStrings(delta, epsilon, zeta string) triple
`)
	writeFile(f3, `package greek
func RuleParen(o Open, e Expr, c Close) (e2 Expr) { return e }
`)

	var rs ruleStringer
	p, w, e := ScanFiles(&rs, f1, f2, f3)
	expectNoWarnings(t, w, e)
	expectPackage(t, p, "greek")
	expectGrammar(t, &rs,
		`RuleIncrement int [int PlusPlus]
RuleParen Expr [Open Expr Close]
RuleStrings triple [string string string]`)

	rs = nil
	p, w, e = ScanFiles(&rs, f1, f2)
	expectNoWarnings(t, w, e)
	expectPackage(t, p, "greek")
	expectGrammar(t, &rs,
		`RuleIncrement int [int PlusPlus]
RuleStrings triple [string string string]`)

	rs = nil
	p, w, e = ScanDir(&rs, tmp)
	expectNoWarnings(t, w, e)
	expectPackage(t, p, "greek")
	expectGrammar(t, &rs,
		`RuleIncrement int [int PlusPlus]
RuleParen Expr [Open Expr Close]
RuleStrings triple [string string string]`)
}

func TestDuplicate(t *testing.T) {
	// Somewhat pointless, but we permit duplicate rules,
	// as long as different names are used.
	tmp := t.TempDir()
	f := tmp + "/twin.go"
	writeFile(f, `package doppelganger
func RuleAdd(Expr, Plus, Expr) Expr
func RuleSum(Expr, Plus, Expr) Expr
`)

	var rs ruleStringer
	p, w, e := ScanFiles(&rs, f)
	expectNoWarnings(t, w, e)
	expectPackage(t, p, "doppelganger")
	expectGrammar(t, &rs, "RuleAdd Expr [Expr Plus Expr]\nRuleSum Expr [Expr Plus Expr]")
}

func TestWarnings(t *testing.T) {
	tmp := t.TempDir()
	f1 := tmp + "/klaxon.go"
	f2 := tmp + "/beeper.go"

	writeFile(f1, `package alert
func RuleDereference(p *Foo) Bar
func RuleConcat(a, b, c int) []int
`)
	writeFile(f2, `package alert
func RuleZap(alpha, beta, gamma)
func RuleMany(int) (alpha, beta, gamma)
func RuleIf(bool, int, int) int
`)

	var rs ruleStringer
	p, w, e := ScanDir(&rs, tmp)
	if e != nil {
		t.Error("Unexpected error:", e)
	}
	expectPackage(t, p, "alert")
	expectGrammar(t, &rs, "RuleIf int [bool int int]")
	expectWarnings(t, w,
		"ignoring RuleDereference: parameter type is not an identifier",
		"ignoring RuleConcat: result type is not an identifier",
		"ignoring RuleZap: number of results is not 1",
		"ignoring RuleMany: number of results is not 1")

	rs = nil
	p, w, e = ScanFiles(&rs, f1, f2)
	if e != nil {
		t.Error("Unexpected error:", e)
	}
	expectPackage(t, p, "alert")
	expectGrammar(t, &rs, "RuleIf int [bool int int]")
	expectWarnings(t, w,
		"ignoring RuleDereference: parameter type is not an identifier",
		"ignoring RuleConcat: result type is not an identifier",
		"ignoring RuleZap: number of results is not 1",
		"ignoring RuleMany: number of results is not 1")
}

func TestNoDir(t *testing.T) {
	tmp := t.TempDir()
	var rs ruleStringer
	_, _, e := ScanDir(&rs, tmp+"/foo")
	if e == nil {
		t.Error("no error when scanning a directory that does not exist")
	}
}

func TestNoFile(t *testing.T) {
	tmp := t.TempDir()
	var rs ruleStringer
	_, _, e := ScanFiles(&rs, tmp+"/foo.go")
	if e == nil {
		t.Error("no error when scanning a directory with no Go files")
	}
}

func TestUnparsable(t *testing.T) {
	tmp := t.TempDir()
	f := tmp + "/broken.go"
	writeFile(f, `package crash
func filter - ( int ) * { return 0 }
`)

	var rs ruleStringer
	_, _, e := ScanFiles(&rs, f)
	if e == nil {
		t.Error("no error when scanning an unparsable file")
	}
}

func TestNoPackage(t *testing.T) {
	tmp := t.TempDir()
	f := tmp + "pkgless.go"
	writeFile(f, `RuleTree(Root, Trunk, Branch, Leaf) Tree
`)

	var rs ruleStringer
	_, _, e := ScanFiles(&rs, f)
	if e == nil {
		t.Error("no error when scanning a file without a package declaration")
	}

	rs = nil
	_, _, e = ScanDir(&rs, tmp)
	if e == nil {
		t.Error("no error when scanning a file without a package declaration")
	}
}

func TestSameName(t *testing.T) {
	tmp := t.TempDir()
	f := tmp + "doubledef.go"
	writeFile(f, `package twins
func RuleTwice(Unit, Unit) Brace
func RuleTwice(Unit, Unit) Brace
`)

	var rs ruleStringer
	_, _, e := ScanFiles(&rs, f)
	if e == nil {
		t.Error("no error when same function name used in two declarations")
	}
}

func TestTwoPackages(t *testing.T) {
	tmp := t.TempDir()
	f1 := tmp + "/alpha.go"
	f2 := tmp + "/omega.go"
	writeFile(f1, `package alpha
func RuleAdd(Expr, Plus, Expr) Expr
`)
	writeFile(f2, `package omega
func RuleMul(Expr, Times, Expr) Expr
`)

	var rs ruleStringer
	_, _, e := ScanFiles(&rs, f1, f2)
	if e == nil {
		t.Error("no error from differing package names")
	}

	rs = nil
	_, _, e = ScanDir(&rs, tmp)
	if e == nil {
		t.Error("no error from differing package names")
	}
}

func TestIgnoreTestFiles(t *testing.T) {
	tmp := t.TempDir()
	f1 := filepath.Join(tmp, "peach.go")
	f2 := filepath.Join(tmp, "peach_test.go")
	writeFile(f1, `package peach
func RuleBite(Peach) Snack`)
	writeFile(f2, `package peach_test
func RuleChoke(Pit) Inedible`)

	var rs ruleStringer
	_, _, e := ScanDir(&rs, tmp)
	if e != nil {
		t.Fatal(e)
	}
	expectGrammar(t, &rs, "RuleBite Snack [Peach]")
}
