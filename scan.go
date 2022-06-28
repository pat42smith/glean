// Copyright 2021 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

// Package glean contains useful utilities for parser generators.
package glean

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
)

// ScanFiles searches one or more files for grammar rules.
//
// For each rule found, rules.AddRule is called. All the files must belong
// to the same package; the name of that package is the first returned value.
func ScanFiles(rules RuleAdder, filenames ...string) (pkg string, warnings []error, err error) {
	if len(filenames) == 0 {
		panic("ScanFiles: no files listed")
	}

	var s scanner
	s.init(rules)

	for _, fname := range filenames {
		file, e := parser.ParseFile(s.fset, fname, nil, 0)
		if e != nil {
			return "", nil, e
		}
		if pkg == "" {
			pkg = file.Name.Name
		} else if pkg != file.Name.Name {
			return "", nil, fmt.Errorf("different package names found: %s and %s", pkg, file.Name.Name)
		}

		e = s.scanFile(file)
		if e != nil {
			return "", nil, e
		}
	}

	return pkg, s.warnings, nil
}

// ScanDir searches for grammar rules in the .go files in a directory
//
// Files named *_test.go are ignored.
// For each rule found, rules.AddRule is called. All the files must belong
// to the same package; the name of that package is the first returned value.
func ScanDir(rules RuleAdder, dirname string) (pkg string, warnings []error, err error) {
	var s scanner
	s.init(rules)

	notTest := func(info fs.FileInfo) bool {
		return !strings.HasSuffix(info.Name(), "_test.go")
	}

	packages, e := parser.ParseDir(s.fset, dirname, notTest, 0)
	if e != nil {
		return "", nil, e
	}
	if len(packages) == 0 {
		return "", nil, fmt.Errorf("no Go files found in directory %s", dirname)
	}
	if len(packages) > 1 {
		names := ""
		for p := range packages {
			if names != "" {
				names += " "
			}
			names += p
		}
		return "", nil, fmt.Errorf("multiple package names found in directory %s: %s", dirname, names)
	}
	for p := range packages {
		pkg = p
	}

	for _, p := range packages {
		for _, file := range p.Files {
			if file.Name.Name != pkg {
				return "", nil, fmt.Errorf("Inconsistency from Go parser: package names %s and %s differ", pkg, file.Name.Name)
			}
			e = s.scanFile(file)
			if e != nil {
				return "", nil, e
			}
		}
	}
	return pkg, s.warnings, nil
}

// A scanner contains the machinery with which to scan Go files for grammar rules
type scanner struct {
	rules    RuleAdder
	fset     *token.FileSet
	warnings []error
	funcPos  map[string]token.Pos
}

// init initializes a scanner
func (s *scanner) init(rules RuleAdder) {
	s.rules = rules
	s.fset = token.NewFileSet()
	s.warnings = nil
	s.funcPos = make(map[string]token.Pos)
}

// scanFile scans a file for grammar rules.
func (s *scanner) scanFile(f *ast.File) error {
	for _, d := range f.Decls {
		if funcd, ok := d.(*ast.FuncDecl); ok && funcd.Name != nil {
			funcname := funcd.Name.Name
			if len(funcname) < 4 || funcname[:4] != "Rule" && funcname[:4] != "rule" {
				continue
			}
			functype := funcd.Type
			if functype == nil {
				continue
			}
			paramTypes, errpos := typeList(functype.Params, s.fset)
			if errpos != token.NoPos {
				where := s.fset.Position(errpos)
				s.warnings = append(s.warnings,
					fmt.Errorf("%s: warning: ignoring %s: parameter type is not an identifier", where, funcname))
				continue
			}
			resultTypes, errpos := typeList(functype.Results, s.fset)
			if errpos != token.NoPos {
				where := s.fset.Position(errpos)
				s.warnings = append(s.warnings,
					fmt.Errorf("%s: warning: ignoring %s: result type is not an identifier", where, funcname))
				continue
			}
			if len(resultTypes) != 1 {
				var where token.Position
				if functype.Results == nil {
					where = s.fset.Position(functype.Pos())
				} else {
					where = s.fset.Position(functype.Results.Pos())
				}
				s.warnings = append(s.warnings,
					fmt.Errorf("%s: warning: ignoring %s: number of results is not 1", where, funcname))
				continue
			}
			if prevPos, seen := s.funcPos[funcname]; seen {
				return fmt.Errorf("%s: %s previously declared at %s",
					s.fset.Position(funcd.Pos()), funcname, s.fset.Position(prevPos))
			}
			s.funcPos[funcname] = funcd.Pos()
			s.rules.AddRule(funcname, resultTypes[0], paramTypes)
		}
	}
	return nil
}

// typeList returns the types from a parameter list or argument list.
// If the second result is not NoPos, then it indicates the position
// of the first type that is not a simple identifier.
func typeList(fl *ast.FieldList, fset *token.FileSet) ([]string, token.Pos) {
	if fl == nil {
		return nil, token.NoPos
	}
	types := make([]string, 0, len(fl.List))
	for _, field := range fl.List {
		count := len(field.Names)
		if count == 0 {
			count = 1
		}
		typeId, isId := field.Type.(*ast.Ident)
		if !isId {
			return nil, field.Type.Pos()
		}
		typeName := typeId.Name
		for i := 0; i < count; i++ {
			types = append(types, typeName)
		}
	}
	return types, token.NoPos
}
