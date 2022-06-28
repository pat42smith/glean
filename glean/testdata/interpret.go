// Copyright 2022 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.
//
// This is an interpreter for a very simple language, to be used as a
// test case for parsing with glean.

package main

import (
	"fmt"
	"os"
)

type OpenParen struct{}
type CloseParen struct{}
type OpenBrace struct{}
type CloseBrace struct{}
type Plus struct{}
type Minus struct{}
type Times struct{}
type Quo struct{}
type Rem struct{}
type Comma struct{}
type Semicolon struct{}
type Equal struct{}
type Assign struct{}
type Less struct{}
type LessEqual struct{}
type Greater struct{}
type GreaterEqual struct{}
type NotEqual struct{}
type And struct{}
type Or struct{}
type If struct{}
type Else struct{}
type Func struct{}
type While struct{}
type Return struct{}
type Print struct{}

type Identifier string
type Int int

type Scope map[Identifier]int

type Expression interface {
	Evaluate(s Scope) int
}

func (i Int) Evaluate(Scope) int {
	return int(i)
}

func (id Identifier) Evaluate(s Scope) int {
	return s[id]
}

type Binary struct {
	left, right Expression
}

type Item Expression

func RuleInt(i Int) Item {
	return i
}

func RuleIdentifier(id Identifier) Item {
	return id
}

func RuleParentheses(_ OpenParen, e Expression, _ CloseParen) Item {
	return e
}

type Term Expression

func RuleItem(a Item) Term {
	return Term(a)
}

type Multiplication Binary

func (m Multiplication) Evaluate(s Scope) int {
	return m.left.Evaluate(s) * m.right.Evaluate(s)
}

func RuleMultiply(a Term, _ Times, b Item) Term {
	return Multiplication{a, b}
}

type Quotient Binary

func (q Quotient) Evaluate(s Scope) int {
	return q.left.Evaluate(s) / q.right.Evaluate(s)
}

func RuleQuotient(a Term, _ Quo, b Item) Term {
	return Quotient{a, b}
}

type Remainder Binary

func (r Remainder) Evaluate(s Scope) int {
	return r.left.Evaluate(s) % r.right.Evaluate(s)
}

func RuleRemainder(a Term, _ Rem, b Item) Term {
	return Remainder{a, b}
}

func RuleTerm(x Term) Expression {
	return x
}

type Addition Binary

func (a Addition) Evaluate(s Scope) int {
	return a.left.Evaluate(s) + a.right.Evaluate(s)
}

func RuleAdd(a Expression, _ Plus, b Term) Expression {
	return Addition{a, b}
}

type Subtraction Binary

func (sub Subtraction) Evaluate(s Scope) int {
	return sub.left.Evaluate(s) - sub.right.Evaluate(s)
}

func RuleSubtract(a Expression, _ Minus, b Term) Expression {
	return Subtraction{a, b}
}

type BoolExpression interface {
	Evaluate(Scope) bool
}

type Comparison BoolExpression

type LessExpr Binary

func (e LessExpr) Evaluate(s Scope) bool {
	return e.left.Evaluate(s) < e.right.Evaluate(s)
}

func RuleLess(a Expression, _ Less, b Expression) Comparison {
	return LessExpr{a, b}
}

type LessEqExpr Binary

func (e LessEqExpr) Evaluate(s Scope) bool {
	return e.left.Evaluate(s) <= e.right.Evaluate(s)
}

func RuleLessEqual(a Expression, _ LessEqual, b Expression) Comparison {
	return LessEqExpr{a, b}
}

type GreaterExpr Binary

func (e GreaterExpr) Evaluate(s Scope) bool {
	return e.left.Evaluate(s) > e.right.Evaluate(s)
}

func RuleGreater(a Expression, _ Greater, b Expression) Comparison {
	return GreaterExpr{a, b}
}

type GreaterEqExpr Binary

func (e GreaterEqExpr) Evaluate(s Scope) bool {
	return e.left.Evaluate(s) >= e.right.Evaluate(s)
}

func RuleGreaterEqual(a Expression, _ GreaterEqual, b Expression) Comparison {
	return GreaterEqExpr{a, b}
}

type EqualExpr Binary

func (e EqualExpr) Evaluate(s Scope) bool {
	return e.left.Evaluate(s) == e.right.Evaluate(s)
}

func RuleEqual(a Expression, _ Equal, b Expression) Comparison {
	return EqualExpr{a, b}
}

type NotEqualExpr Binary

func (e NotEqualExpr) Evaluate(s Scope) bool {
	return e.left.Evaluate(s) != e.right.Evaluate(s)
}

func RuleNotEqual(a Expression, _ NotEqual, b Expression) Comparison {
	return NotEqualExpr{a, b}
}

func RuleBoolParens(_ OpenParen, e BoolExpression, _ CloseParen) Comparison {
	return e
}

type BoolBinary struct {
	left, right BoolExpression
}

type Anding BoolExpression

func RuleComparison(a Comparison) Anding {
	return a
}

type AndExpr BoolBinary

func (a AndExpr) Evaluate(s Scope) bool {
	return a.left.Evaluate(s) && a.right.Evaluate(s)
}

func RuleAnd(a Anding, _ And, b Comparison) Anding {
	return AndExpr{a, b}
}

func RuleAnding(a Anding) BoolExpression {
	return a
}

type OrExpr BoolBinary

func (o OrExpr) Evaluate(s Scope) bool {
	return o.left.Evaluate(s) || o.right.Evaluate(s)
}

func RuleOr(a BoolExpression, _ Or, b Anding) BoolExpression {
	return OrExpr{a, b}
}

type Statement interface {
	Execute(s Scope) bool
}

type Assignment struct {
	left  Identifier
	right Expression
}

func (a Assignment) Execute(s Scope) bool {
	s[a.left] = a.right.Evaluate(s)
	return false
}

func RuleAssign(a Identifier, _ Assign, e Expression) Statement {
	return Assignment{a, e}
}

type NullStatement struct{}

func (s NullStatement) Execute(Scope) bool {
	return false
}

func RuleNull() Statement {
	return NullStatement{}
}

type IfStatement struct {
	condition          BoolExpression
	thenpart, elsepart Block
}

func (i IfStatement) Execute(s Scope) bool {
	if i.condition.Evaluate(s) {
		return i.thenpart.Execute(s)
	} else {
		return i.elsepart.Execute(s)
	}
}

func RuleIf(_ If, c BoolExpression, t Block) Statement {
	return IfStatement{c, t, Block{nil}}
}

func RuleIfElse(_ If, c BoolExpression, t Block, _ Else, e Block) Statement {
	return IfStatement{c, t, e}
}

type WhileStatement struct {
	condition BoolExpression
	body      Block
}

func (w WhileStatement) Execute(s Scope) bool {
	for w.condition.Evaluate(s) {
		if w.body.Execute(s) {
			return true
		}
	}
	return false
}

func RuleWhile(_ While, c BoolExpression, b Block) Statement {
	return WhileStatement{c, b}
}

type ExpressionList []Expression

type EmptyExpressionList ExpressionList
type NonEmptyExpressionList ExpressionList

func RuleEL0(_ EmptyExpressionList) ExpressionList {
	return nil
}

func RuleELn(el NonEmptyExpressionList) ExpressionList {
	return ExpressionList(el)
}

func RuleEmptyExpressionList() EmptyExpressionList {
	return nil
}

func RuleSingleExpression(e Expression) NonEmptyExpressionList {
	return NonEmptyExpressionList{e}
}

func RuleExpressionList(el NonEmptyExpressionList, _ Comma, e Expression) NonEmptyExpressionList {
	return append(el, e)
}

type PrintStatement ExpressionList

func (p PrintStatement) Execute(s Scope) bool {
	values := make([]int, len(p))
	for n, e := range p {
		values[n] = e.Evaluate(s)
	}
	for n, v := range values {
		if n > 0 {
			fmt.Print(" ")
		}
		fmt.Print(v)
	}
	fmt.Println()
	return false
}

func RulePrint(_ Print, el ExpressionList) Statement {
	return PrintStatement(el)
}

type StatementList []Statement

func (sl StatementList) Execute(s Scope) bool {
	for _, stmt := range sl {
		if stmt.Execute(s) {
			return true
		}
	}
	return false
}

func RuleOneStatement(s Statement) StatementList {
	return StatementList{s}
}

func RuleStatementList(sl StatementList, _ Semicolon, s Statement) StatementList {
	return append(sl, s)
}

type Block struct {
	StatementList
}

func RuleBlock(_ OpenBrace, sl StatementList, _ CloseBrace) Block {
	return Block{sl}
}

type IdentifierList []Identifier

type EmptyIdentifierList IdentifierList
type NonEmptyIdentifierList IdentifierList

func RuleIL0(EmptyIdentifierList) IdentifierList {
	return nil
}

func RuleILn(il NonEmptyIdentifierList) IdentifierList {
	return IdentifierList(il)
}

func RuleEmptyIdList() EmptyIdentifierList {
	return nil
}

func RuleOneIdentifier(i Identifier) NonEmptyIdentifierList {
	return NonEmptyIdentifierList{i}
}

func RuleNEIdentifierList(il NonEmptyIdentifierList, _ Comma, i Identifier) NonEmptyIdentifierList {
	return append(il, i)
}

type Function struct {
	name       Identifier
	parameters IdentifierList
	body       Block
}

func (f Function) Call(args []int) int {
	s := make(Scope)
	for n, p := range f.parameters {
		if n < len(args) {
			s[p] = args[n]
		}
	}
	f.body.Execute(s)
	return s[""]
}

var functions = make(map[Identifier]Function)

func RuleFunction(_ Func, name Identifier, _ OpenParen, parms IdentifierList, _ CloseParen, body Block) Function {
	f := Function{name, parms, body}
	functions[name] = f
	return f
}

type ReturnStatement struct {
	Expression
}

func (r ReturnStatement) Execute(s Scope) bool {
	s[""] = r.Evaluate(s)
	return true
}

func RuleReturn(_ Return, e Expression) Statement {
	return ReturnStatement{e}
}

type FunctionCall struct {
	fname Identifier
	args  ExpressionList
}

func (fc FunctionCall) Evaluate(s Scope) int {
	f := functions[fc.fname]
	if f.name == "" {
		panic(fmt.Sprint("no function named ", fc.fname))
	}
	var args []int
	for _, a := range fc.args {
		args = append(args, a.Evaluate(s))
	}
	return f.Call(args)
}

func RuleCall(fname Identifier, _ OpenParen, args ExpressionList, _ CloseParen) Item {
	return FunctionCall{fname, args}
}

type Program struct{}

func (p Program) Execute() {
	m := functions["main"]
	if m.name == "" {
		panic("no main function")
	}
	m.Call(nil)
}

func RuleSingleFunction(Function) Program {
	return Program{}
}

func RuleProgram(Program, Function) Program {
	return Program{}
}

func tokenize(s string) []interface{} {
	var tokens []interface{}
loop:
	for n := 0; n < len(s); n++ {
		var t interface{}
		c := s[n]
		switch c {
		case ' ', '\t', '\n':
			continue loop
		case '(':
			t = OpenParen{}
		case ')':
			t = CloseParen{}
		case '{':
			t = OpenBrace{}
		case '}':
			t = CloseBrace{}
		case '+':
			t = Plus{}
		case '-':
			t = Minus{}
		case '*':
			t = Times{}
		case '/':
			t = Quo{}
		case '%':
			t = Rem{}
		case ',':
			t = Comma{}
		case ';':
			t = Semicolon{}
		case '=':
			if n+1 < len(s) && s[n+1] == '=' {
				t = Equal{}
				n++
			} else {
				t = Assign{}
			}
		case '<':
			if n+1 < len(s) && s[n+1] == '=' {
				t = LessEqual{}
				n++
			} else {
				t = Less{}
			}
		case '>':
			if n+1 < len(s) && s[n+1] == '=' {
				t = GreaterEqual{}
				n++
			} else {
				t = Greater{}
			}
		case '!':
			if n+1 < len(s) && s[n+1] == '=' {
				t = NotEqual{}
				n++
			} else {
				panic("! not followed by =")
			}
		case '&':
			if n+1 < len(s) && s[n+1] == '&' {
				t = And{}
				n++
			} else {
				panic("& not followed by &")
			}
		case '|':
			if n+1 < len(s) && s[n+1] == '|' {
				t = Or{}
				n++
			} else {
				panic("| not followed by |")
			}
		}
		if t == nil {
			if isdigit(c) {
				i := int(c - '0')
				for n+1 < len(s) && isdigit(s[n+1]) {
					n++
					i = i*10 + int(s[n]-'0')
				}
				t = Int(i)
			} else if isalpha(c) {
				a := n
				for n+1 < len(s) && isalnum(s[n+1]) {
					n++
				}
				u := s[a : n+1]
				switch u {
				case "if":
					t = If{}
				case "else":
					t = Else{}
				case "func":
					t = Func{}
				case "while":
					t = While{}
				case "return":
					t = Return{}
				case "print":
					t = Print{}
				default:
					t = Identifier(u)
				}
			} else {
				panic(fmt.Sprintf("invalid character '%c' (at position %d)", c, n))
			}
		}
		if t != nil {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

func isdigit(c byte) bool {
	return '0' <= c && c <= '9'
}

func isalpha(c byte) bool {
	return c == '_' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

func isalnum(c byte) bool {
	return isdigit(c) || isalpha(c)
}

func main() {
	args := os.Args
	if len(args) != 2 {
		panic("usage: interpret <filename>")
	}
	file := args[1]

	bytes, e := os.ReadFile(file)
	if e != nil {
		panic(e)
	}
	text := string(bytes)
	tokens := tokenize(text)

	program, e := _glean_Parse(tokens)
	if e != nil {
		panic(e)
	}

	program.Execute()
}
