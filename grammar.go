// Copyright 2021-2022 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package glean

// A RuleAdder can have grammar rules added to it.
//
// Grammar symbols are represented by arbitrary strings. Each rule
// has a name, which should be unique.
//
// If the intent is to write a parser for the grammar, then the
// rule names and symbol strings should be valid Go identifiers.
type RuleAdder interface {
	// AddRule adds one rule to the grammar.
	//
	// Callers should ensure the same name is never used in two calls
	// to AddRule.
	AddRule(name, target string, items []string) error
}

// A ParserWriter can write a parser (in Go) for a grammar.
type ParserWriter interface {
	// ParserWriter writes a grammar parser in Go.
	//
	// Typically, the caller will write the result into a .go file,
	// with a comment marking the file as automatically generated.
	//
	// The goal argument is the goal symbol that will be the result
	// of the parse.  The parse function will have the signature
	//
	// func ([]interface{}) (goal, error)
	//
	// The packname argument is copied to the package statment in the
	// generated code. The prefix is prepended to the names of all
	// file-level identifiers; in particular the name of the main
	// parse function will be prefix + "Parse".
	WriteParser(goal, packname, prefix string) (string, error)
}

// A Grammar can accumulate rules and write a parser.
type Grammar interface {
	RuleAdder
	ParserWriter
}
