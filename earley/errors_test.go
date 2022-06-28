// Copyright 2021 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package earley

import (
	"testing"
)

func CheckZero(t *testing.T, g Grammar) {
	t.Helper()
	zero := g.name2symbol == nil &&
		g.symbols == nil &&
		g.terminals == nil &&
		g.nonterminals == nil &&
		g.prefixes == nil &&
		g.goalname == "" &&
		g.packname == "" &&
		g.prepend == "" &&
		g.goal == nil &&
		g.builder == nil
	if !zero {
		t.Error("Grammar has been changed")
	}
}

func MustError(t *testing.T, f, want string, e error) {
	t.Helper()
	if e == nil {
		t.Errorf("No error from %s; expected: %s", f, want)
	} else if e.Error() != want {
		t.Errorf("Wrong error from %s\nexpected: %s\ngot: %s", f, want, e)
	}
}

func TestAddRuleErrors(t *testing.T) {
	var g Grammar

	e := g.AddRule("", "target", []string{"foo", "bar"})
	MustError(t, "AddRule", "rule name '' is not a valid Go identifier", e)
	CheckZero(t, g)

	e = g.AddRule("17", "target", []string{"foo", "bar"})
	MustError(t, "AddRule", "rule name '17' is not a valid Go identifier", e)
	CheckZero(t, g)

	e = g.AddRule("Rule", "", []string{"foo", "bar"})
	MustError(t, "AddRule", "target symbol '' is not a valid Go identifier", e)
	CheckZero(t, g)

	e = g.AddRule("Rule", "@@", []string{"foo", "bar"})
	MustError(t, "AddRule", "target symbol '@@' is not a valid Go identifier", e)
	CheckZero(t, g)

	e = g.AddRule("Rule", "target", []string{"foo", "", "bar"})
	MustError(t, "AddRule", "rule item '' is not a valid Go identifier", e)
	CheckZero(t, g)

	e = g.AddRule("Rule", "target", []string{"foo", "x.y.z", "bar"})
	MustError(t, "AddRule", "rule item 'x.y.z' is not a valid Go identifier", e)
	CheckZero(t, g)

	e = g.AddRule("Rule", "target", []string{"foo", "bar"})
	if e != nil {
		t.Fatal("valid call to AddRule failed:", e)
	}

	e = g.AddRule("Rule", "other", []string{"alpha", "beta", "gamma"})
	MustError(t, "AddRule", "duplicate rule name: Rule", e)
}

func WPMustError(t *testing.T, want string, text string, e error) {
	t.Helper()
	MustError(t, "WriteParser", want, e)
	if text != "" {
		t.Errorf("WriteParser returned error with non-empty parser text\nerror was: %s\n", e)
	}
}

func TestWriteParserErrors(t *testing.T) {
	var g Grammar

	text, e := g.WriteParser("Goal", "main", "_")
	WPMustError(t, "grammar has no rules", text, e)

	e = g.AddRule("RuleGoal", "Goal", []string{"step"})
	if e != nil {
		t.Fatal("AddRule failed:", e)
	}

	text, e = g.WriteParser("", "main", "_")
	WPMustError(t, "goal '' is not a valid Go identifier", text, e)
	text, e = g.WriteParser("-", "main", "_")
	WPMustError(t, "goal '-' is not a valid Go identifier", text, e)
	text, e = g.WriteParser("nonesuch", "main", "_")
	WPMustError(t, "unknown goal symbol 'nonesuch'", text, e)
	text, e = g.WriteParser("step", "main", "_")
	WPMustError(t, "goal 'step' is a terminal symbol", text, e)

	text, e = g.WriteParser("Goal", "", "_")
	WPMustError(t, "package name '' is not a valid Go identifier", text, e)
	text, e = g.WriteParser("Goal", "()", "_")
	WPMustError(t, "package name '()' is not a valid Go identifier", text, e)

	text, e = g.WriteParser("Goal", "main", "[:]")
	WPMustError(t, "prefix '[:]' is not a valid Go identifier", text, e)

	text, e = g.WriteParser("Goal", "main", "_")
	if e != nil {
		t.Fatal("WriteParser failed:", e)
	}

	e = g.AddRule("RuleStep", "step", []string{})
	if e != nil {
		t.Fatal("AddRule failed:", e)
	}
	text, e = g.WriteParser("Goal", "main", "_")
	WPMustError(t, "grammar has no terminal symbols", text, e)
}
