// Copyright 2021 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

// Benchmarks for three methods of passing tokens from the tokenizer to the parser:
// I: (interface) as []interface{}
// M: (method) as []interface{ TypeId() int }
// L: (list) using a customised type to represent a list of tokens
//
// We use a simple expression grammar:
// Expr = Prod
// Expr = Expr '+' Prod
// Expr = Expr '-' Prod
// Prod = Item
// Prod = Prod '*' Item
// Item = Literal
// Item = '(' Expr ')'
// Literal = '0' | '1' | '2' | '3' | '4' | '5' | '6' | '7' | '8' | '9'
// Note that only single digit literals are used, and we do not allow division,
// so we will never try to divide by 0.

package main

import "math/rand"
import "testing"
import "time"

// The string we will parse.
var source string

func init() {
	rand.Seed(time.Now().UnixNano())

	parts := make([]string, 1e4)
	for n := range parts {
		d := rand.Intn(10)
		parts[n] = "0123456789"[d : d+1]
	}
	for len(parts) > 1 {
		x := rand.Intn(3)
		op := "+-*"[x : x+1]
		k := rand.Intn(len(parts) - 1)
		parts[k] = parts[k] + op + parts[k+1]
		if rand.Intn(4) == 0 {
			parts[k] = "(" + parts[k] + ")"
		}
		copy(parts[k+1:], parts[k+2:])
		parts = parts[:len(parts)-1]
	}

	source = parts[0]
}

// Types to represent the grammar symbols.
type Expr = int
type Prod = int
type Item = int
type Plus struct{}
type Minus struct{}
type Times struct{}
type Open struct{}
type Close struct{}
type Literal int

// Token type identifiers.
const (
	PlusId = iota
	MinusId
	TimesId
	OpenId
	CloseId
	LiteralId
)

func (Plus) TypeId() int    { return PlusId }
func (Minus) TypeId() int   { return MinusId }
func (Times) TypeId() int   { return TimesId }
func (Open) TypeId() int    { return OpenId }
func (Close) TypeId() int   { return CloseId }
func (Literal) TypeId() int { return LiteralId }

// I. Parser using []interface{}

type IParser struct {
	saved  []int
	input  []interface{}
	states []IState
}

type IState func(p *IParser)

func (p *IParser) Parse() int {
	p.saved = nil
	p.input = ITokenize()
	p.states = []IState{IExpr0}
	for len(p.states) > 0 {
		s := p.states[len(p.states)-1]
		p.states = p.states[:len(p.states)-1]
		s(p)
	}
	if len(p.saved) != 1 || len(p.input) != 0 {
		panic("bug")
	}
	return p.saved[0]
}

func ITokenize() []interface{} {
	list := make([]interface{}, len(source))
	var nothing struct{}
	for n := 0; n < len(source); n++ {
		switch c := source[n]; c {
		case '+':
			list[n] = Plus(nothing)
		case '-':
			list[n] = Minus(nothing)
		case '*':
			list[n] = Times(nothing)
		case '(':
			list[n] = Open(nothing)
		case ')':
			list[n] = Close(nothing)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			list[n] = Literal(c - '0')
		}
	}
	return list
}

func IExpr0(p *IParser) {
	p.states = append(p.states, IExpr1, IProd0)
}

func IExpr1(p *IParser) {
	if len(p.input) == 0 {
		return
	}
	switch p.input[0].(type) {
	case Plus:
		p.states = append(p.states, IExprPlus, IProd0)
		p.input = p.input[1:]
	case Minus:
		p.states = append(p.states, IExprMinus, IProd0)
		p.input = p.input[1:]
	default:
		return
	}
}

func IExprPlus(p *IParser) {
	l := len(p.saved)
	b := p.saved[l-1]
	p.saved[l-2] += b
	p.saved = p.saved[:l-1]
	p.states = append(p.states, IExpr1)
}

func IExprMinus(p *IParser) {
	l := len(p.saved)
	b := p.saved[l-1]
	p.saved[l-2] -= b
	p.saved = p.saved[:l-1]
	p.states = append(p.states, IExpr1)
}

func IProd0(p *IParser) {
	p.states = append(p.states, IProd1, IItem)
}

func IProd1(p *IParser) {
	if len(p.input) == 0 {
		return
	}
	switch p.input[0].(type) {
	case Times:
		p.states = append(p.states, IProdTimes, IItem)
		p.input = p.input[1:]
	default:
		return
	}
}

func IProdTimes(p *IParser) {
	l := len(p.saved)
	b := p.saved[l-1]
	p.saved[l-2] *= b
	p.saved = p.saved[:l-1]
	p.states = append(p.states, IProd1)
}

func IItem(p *IParser) {
	switch x := p.input[0].(type) {
	case Literal:
		p.saved = append(p.saved, int(x))
		p.input = p.input[1:]
	case Open:
		p.states = append(p.states, IParen, IExpr0)
		p.input = p.input[1:]
	default:
		panic("bug")
	}
}

func IParen(p *IParser) {
	switch p.input[0].(type) {
	case Close:
		p.input = p.input[1:]
	default:
		panic("bug")
	}
}

func BenchmarkInterface(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var p IParser
		p.Parse()
	}
}

// M. Parser using TypeId method

type MParser struct {
	saved  []int
	input  []Typer
	states []MState
}

type Typer interface {
	TypeId() int
}

type MState func(p *MParser)

func (p *MParser) Parse() int {
	p.saved = nil
	p.input = MTokenize()
	p.states = []MState{MExpr0}
	for len(p.states) > 0 {
		s := p.states[len(p.states)-1]
		p.states = p.states[:len(p.states)-1]
		s(p)
	}
	if len(p.saved) != 1 || len(p.input) != 0 {
		panic("bug")
	}
	return p.saved[0]
}

func MTokenize() []Typer {
	list := make([]Typer, len(source))
	var nothing struct{}
	for n := 0; n < len(source); n++ {
		switch c := source[n]; c {
		case '+':
			list[n] = Plus(nothing)
		case '-':
			list[n] = Minus(nothing)
		case '*':
			list[n] = Times(nothing)
		case '(':
			list[n] = Open(nothing)
		case ')':
			list[n] = Close(nothing)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			list[n] = Literal(c - '0')
		}
	}
	return list
}

func MExpr0(p *MParser) {
	p.states = append(p.states, MExpr1, MProd0)
}

func MExpr1(p *MParser) {
	if len(p.input) == 0 {
		return
	}
	switch p.input[0].TypeId() {
	case PlusId:
		p.states = append(p.states, MExprPlus, MProd0)
		p.input = p.input[1:]
	case MinusId:
		p.states = append(p.states, MExprMinus, MProd0)
		p.input = p.input[1:]
	default:
		return
	}
}

func MExprPlus(p *MParser) {
	l := len(p.saved)
	b := p.saved[l-1]
	p.saved[l-2] += b
	p.saved = p.saved[:l-1]
	p.states = append(p.states, MExpr1)
}

func MExprMinus(p *MParser) {
	l := len(p.saved)
	b := p.saved[l-1]
	p.saved[l-2] -= b
	p.saved = p.saved[:l-1]
	p.states = append(p.states, MExpr1)
}

func MProd0(p *MParser) {
	p.states = append(p.states, MProd1, MItem)
}

func MProd1(p *MParser) {
	if len(p.input) == 0 {
		return
	}
	switch p.input[0].TypeId() {
	case TimesId:
		p.states = append(p.states, MProdTimes, MItem)
		p.input = p.input[1:]
	default:
		return
	}
}

func MProdTimes(p *MParser) {
	l := len(p.saved)
	b := p.saved[l-1]
	p.saved[l-2] *= b
	p.saved = p.saved[:l-1]
	p.states = append(p.states, MProd1)
}

func MItem(p *MParser) {
	switch p.input[0].TypeId() {
	case LiteralId:
		p.saved = append(p.saved, int(p.input[0].(Literal)))
		p.input = p.input[1:]
	case OpenId:
		p.states = append(p.states, MParen, MExpr0)
		p.input = p.input[1:]
	default:
		panic("bug")
	}
}

func MParen(p *MParser) {
	switch p.input[0].TypeId() {
	case CloseId:
		p.input = p.input[1:]
	default:
		panic("bug")
	}
}

func BenchmarkMethod(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var p MParser
		p.Parse()
	}
}

// L. Parser using TokenList type

type TokenList struct {
	plus      []Plus
	minus     []Minus
	times     []Times
	opens     []Open
	closes    []Close
	literals  []Literal
	tokenType []int
	where     []int
}

func (tl *TokenList) TypeAt(n int) int {
	return tl.tokenType[n]
}

func (tl *TokenList) PlusAt(n int) Plus       { return tl.plus[tl.where[n]] }
func (tl *TokenList) MinusAt(n int) Minus     { return tl.minus[tl.where[n]] }
func (tl *TokenList) TimesAt(n int) Times     { return tl.times[tl.where[n]] }
func (tl *TokenList) OpenAt(n int) Open       { return tl.opens[tl.where[n]] }
func (tl *TokenList) CloseAt(n int) Close     { return tl.closes[tl.where[n]] }
func (tl *TokenList) LiteralAt(n int) Literal { return tl.literals[tl.where[n]] }

func (tl *TokenList) AddPlus(p Plus) {
	tl.tokenType = append(tl.tokenType, PlusId)
	tl.where = append(tl.where, len(tl.plus))
	tl.plus = append(tl.plus, p)
}

func (tl *TokenList) AddMinus(m Minus) {
	tl.tokenType = append(tl.tokenType, MinusId)
	tl.where = append(tl.where, len(tl.minus))
	tl.minus = append(tl.minus, m)
}

func (tl *TokenList) AddTimes(t Times) {
	tl.tokenType = append(tl.tokenType, TimesId)
	tl.where = append(tl.where, len(tl.times))
	tl.times = append(tl.times, t)
}

func (tl *TokenList) AddOpen(o Open) {
	tl.tokenType = append(tl.tokenType, OpenId)
	tl.where = append(tl.where, len(tl.opens))
	tl.opens = append(tl.opens, o)
}

func (tl *TokenList) AddClose(c Close) {
	tl.tokenType = append(tl.tokenType, CloseId)
	tl.where = append(tl.where, len(tl.closes))
	tl.closes = append(tl.closes, c)
}

func (tl *TokenList) AddLiteral(l Literal) {
	tl.tokenType = append(tl.tokenType, LiteralId)
	tl.where = append(tl.where, len(tl.literals))
	tl.literals = append(tl.literals, l)
}

func (tl *TokenList) Pop() {
	tl.tokenType = tl.tokenType[1:]
	tl.where = tl.where[1:]
}

func (tl *TokenList) Empty() bool {
	return len(tl.tokenType) == 0
}

type LParser struct {
	saved  []int
	input  TokenList
	states []LState
}

type LState func(p *LParser)

func (p *LParser) Parse() int {
	p.saved = nil
	LTokenize(&p.input)
	p.states = []LState{LExpr0}
	for len(p.states) > 0 {
		s := p.states[len(p.states)-1]
		p.states = p.states[:len(p.states)-1]
		s(p)
	}
	if len(p.saved) != 1 || !p.input.Empty() {
		panic("bug")
	}
	return p.saved[0]
}

func LTokenize(tl *TokenList) {
	var nothing struct{}
	for n := 0; n < len(source); n++ {
		switch c := source[n]; c {
		case '+':
			tl.AddPlus(Plus(nothing))
		case '-':
			tl.AddMinus(Minus(nothing))
		case '*':
			tl.AddTimes(Times(nothing))
		case '(':
			tl.AddOpen(Open(nothing))
		case ')':
			tl.AddClose(Close(nothing))
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			tl.AddLiteral(Literal(c - '0'))
		}
	}
}

func LExpr0(p *LParser) {
	p.states = append(p.states, LExpr1, LProd0)
}

func LExpr1(p *LParser) {
	if p.input.Empty() {
		return
	}
	switch p.input.TypeAt(0) {
	case PlusId:
		p.states = append(p.states, LExprPlus, LProd0)
		p.input.Pop()
	case MinusId:
		p.states = append(p.states, LExprMinus, LProd0)
		p.input.Pop()
	default:
		return
	}
}

func LExprPlus(p *LParser) {
	l := len(p.saved)
	b := p.saved[l-1]
	p.saved[l-2] += b
	p.saved = p.saved[:l-1]
	p.states = append(p.states, LExpr1)
}

func LExprMinus(p *LParser) {
	l := len(p.saved)
	b := p.saved[l-1]
	p.saved[l-2] -= b
	p.saved = p.saved[:l-1]
	p.states = append(p.states, LExpr1)
}

func LProd0(p *LParser) {
	p.states = append(p.states, LProd1, LItem)
}

func LProd1(p *LParser) {
	if p.input.Empty() {
		return
	}
	switch p.input.TypeAt(0) {
	case TimesId:
		p.states = append(p.states, LProdTimes, LItem)
		p.input.Pop()
	default:
		return
	}
}

func LProdTimes(p *LParser) {
	l := len(p.saved)
	b := p.saved[l-1]
	p.saved[l-2] *= b
	p.saved = p.saved[:l-1]
	p.states = append(p.states, LProd1)
}

func LItem(p *LParser) {
	switch p.input.TypeAt(0) {
	case LiteralId:
		p.saved = append(p.saved, int(p.input.LiteralAt(0)))
		p.input.Pop()
	case OpenId:
		p.states = append(p.states, LParen, LExpr0)
		p.input.Pop()
	default:
		panic("bug")
	}
}

func LParen(p *LParser) {
	switch p.input.TypeAt(0) {
	case CloseId:
		p.input.Pop()
	default:
		panic("bug")
	}
}

func BenchmarkList(b *testing.B) {
	for n := 0; n < b.N; n++ {
		var p LParser
		p.Parse()
	}
}
