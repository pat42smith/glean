// Copyright 2021-2022 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

// Package earley contains an implementation of glean.Grammar using
// a simple Earley-style parser.
package earley

import (
	"fmt"
	"go/token"
	"strconv"
	"strings"
)

// *Grammar is an Earley implentation of glean.Grammar.
type Grammar struct {
	rulenames                        map[string]struct{}
	name2symbol                      map[string]*symbol
	rules                            []*rule
	symbols, terminals, nonterminals []*symbol
	prefixes                         []*prefix
	goalname, packname, prepend      string // WriteParser arguments
	goal                             *symbol
	builder                          *strings.Builder // accumulates parser text
}

// Implements glean.RuleAdder.AddRule.
func (g *Grammar) AddRule(name, target string, items []string) error {
	if !token.IsIdentifier(name) {
		return fmt.Errorf("rule name '%s' is not a valid Go identifier", name)
	}
	if !token.IsIdentifier(target) {
		return fmt.Errorf("target symbol '%s' is not a valid Go identifier", target)
	}
	for _, item := range items {
		if !token.IsIdentifier(item) {
			return fmt.Errorf("rule item '%s' is not a valid Go identifier", item)
		}
	}

	if g.rulenames == nil {
		g.rulenames = make(map[string]struct{})
	}
	if g.name2symbol == nil {
		g.name2symbol = make(map[string]*symbol)
	}

	if _, have := g.rulenames[name]; have {
		return fmt.Errorf("duplicate rule name: %s", name)
	}
	g.rulenames[name] = struct{}{}

	var r rule
	r.name = name
	r.target = g.findSymbol(target)
	r.items = make([]*symbol, len(items))
	for n, i := range items {
		r.items[n] = g.findSymbol(i)
	}
	r.id = len(g.rules)
	g.rules = append(g.rules, &r)
	r.target.rules = append(r.target.rules, &r)

	return nil
}

// Finds or creates a symbol from its name
func (g *Grammar) findSymbol(name string) *symbol {
	if s, have := g.name2symbol[name]; have {
		return s
	}
	s := &symbol{name: name}
	g.name2symbol[name] = s
	return s
}

// Implements glean.ParserWriter.WriteParser.
func (g *Grammar) WriteParser(goal, packname, prepend string) (string, error) {
	if len(g.rulenames) == 0 {
		return "", fmt.Errorf("grammar has no rules")
	}
	if !token.IsIdentifier(goal) {
		return "", fmt.Errorf("goal '%s' is not a valid Go identifier", goal)
	}
	if !token.IsIdentifier(packname) {
		return "", fmt.Errorf("package name '%s' is not a valid Go identifier", packname)
	}
	if prepend != "" && !token.IsIdentifier(prepend) {
		return "", fmt.Errorf("prefix '%s' is not a valid Go identifier", prepend)
	}
	g.goalname = goal
	g.packname = packname
	g.prepend = prepend

	g.sortSymbols()
	for _, s := range g.symbols {
		s.sortRules()
	}
	if len(g.terminals) == 0 {
		return "", fmt.Errorf("grammar has no terminal symbols")
	}
	if len(g.nonterminals) == 0 {
		panic("bug: how can we have rules but no nonterminals?")
	}

	g.goal = g.name2symbol[g.goalname]
	if g.goal == nil {
		return "", fmt.Errorf("unknown goal symbol '%s'", g.goalname)
	}
	if g.goal.isTerminal() {
		return "", fmt.Errorf("goal '%s' is a terminal symbol", g.goalname)
	}

	g.makePrefixes()

	g.builder = new(strings.Builder)
	g.addText(boilerplate)
	g.addParserType()
	g.addApplyTrace()

	g.addFollowers()
	g.addLastTerminal()
	g.addExtendedBy()
	g.addExtensions()
	g.addSymbolFinished()
	g.addTokenType()
	g.addGoalPrefixes()
	g.addApplyTerminal()
	g.addAppliers()
	g.addPrefix2Rule()
	g.addRuleDescriptions()

	return g.builder.String(), nil
}

// Sort the symbols so terminals precede non-terminals, and assign each symbol a unique id.
func (g *Grammar) sortSymbols() {
	t, u := 0, len(g.name2symbol)
	g.symbols = make([]*symbol, u)
	for _, s := range g.name2symbol {
		if s.isTerminal() {
			g.symbols[t] = s
			t++
		} else {
			u--
			g.symbols[u] = s
		}
	}
	if t != u {
		panic("bug")
	}
	g.terminals = g.symbols[:t]
	g.nonterminals = g.symbols[t:]

	for n, s := range g.symbols {
		s.id = n
	}
}

// Create all the rule prefixes
func (g *Grammar) makePrefixes() {
	g.prefixes = g.prefixes[:0]
	for _, s := range g.nonterminals {
		for _, r := range s.rules {
			r.fullPrefix = nil
		}
		p := g.newPrefix()
		p.target = s
		p.length = 0
		p.rules = s.rules
		g.makeExtensions(p)
		s.prefix0 = p
	}
}

// Extend a state by one symbol in each production
func (g *Grammar) makeExtensions(p *prefix) {
	n := p.length
	r := p.rules
	if len(r[0].items) == n {
		r[0].fullPrefix = p
		r = r[1:]
	}
	p.extensions = p.extensions[:0]
	for i, j := 0, 0; i < len(r); i = j {
		x := r[i].items[n]
		j = i + 1
		for j < len(r) && r[j].items[n] == x {
			j++
		}
		ext := g.newPrefix()
		ext.target = p.target
		ext.length = n + 1
		ext.rules = r[i:j]
		p.extensions = append(p.extensions, ext)
		g.makeExtensions(ext)
	}
}

// Return a new state with its id set correctly
func (g *Grammar) newPrefix() *prefix {
	var p prefix
	p.id = len(g.prefixes)
	g.prefixes = append(g.prefixes, &p)
	return &p
}

// Append a string to the parser text, replacing @ and #? with WriteParser parameters
func (g *Grammar) addText(s string) {
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch c {
		case '@':
			g.addString(g.prepend)
		case '#':
			i++
			d := s[i]
			var t string
			switch d {
			case 'G':
				t = g.goal.name
			case 'g':
				t = strconv.Itoa(g.goal.prefix0.id)
			case 'P':
				t = g.packname
			default:
				t = fmt.Sprintf("#%c", d)
			}
			g.addString(t)
		default:
			e := g.builder.WriteByte(c)
			if e != nil {
				panic(e)
			}
		}
	}
}

// Append a string to the parser text, unchanged
func (g *Grammar) addString(s string) {
	n, e := g.builder.WriteString(s)
	if n != len(s) || e != nil {
		panic(e)
	}
}

// Append to the parser text, with formatting, without parameter replacement
func (g *Grammar) addf(format string, args ...interface{}) {
	_, e := fmt.Fprintf(g.builder, format, args...)
	if e != nil {
		panic(e)
	}
}

// Append an integer slice to the parser text
func (g *Grammar) addSlice(s []int) {
	g.addString("{")
	for n, i := range s {
		if n > 0 {
			g.addString(", ")
		}
		g.addf("%d", i)
	}
	g.addString("}")
}

// Standard text needing only simple modifications
var boilerplate = `package #P

import (
	"fmt"

	"github.com/pat42smith/glean/gleanerrors"
)

type @_Prefix int
type @_Rule int
type @_Symbol int

type @_Match struct {
	prefix          @_Prefix
	completePrefix  @_Prefix
	start, end      int
	shorter, last   *@_Match
	shorter2, last2 *@_Match
}

func @Parse(tokens []interface{}) (#G, error) {
	var parser @_Parser
	parser.tokens = tokens
	return parser.parse()
}

func (parser *@_Parser) parse() (#G, error) {
	// fmt.Fprintln(os.Stderr, parser.tokens)
	parser.matches = make([]map[@_Prefix][]*@_Match, len(parser.tokens)+1)
	parser.todo = make([][]*@_Match, len(parser.tokens)+1)
	for end := range parser.matches {
		parser.matches[end] = make(map[@_Prefix][]*@_Match)
	}

	var zero #G
	if len(parser.tokens) == 0 {
		return zero, gleanerrors.NoInput{}
	}
	if e := parser.findMatches(); e != nil {
		return zero, e
	}
	if e := parser.findTrace(); e != nil {
		return zero, e
	}

	return parser.applyTrace(), nil
}

func (parser *@_Parser) addMatch(prefix @_Prefix, start, end int, shorter, last *@_Match) {
	list := parser.matches[end][prefix]
	for _, m := range list {
		if m.start == start {
			if m.shorter != shorter || m.last != last {
				if m.shorter2 == nil {
					m.shorter2 = shorter
					m.last2 = last
				}
			}
			return
		}
	}
	m := @_Match{prefix, -1, start, end, shorter, last, nil, nil}
	parser.matches[end][prefix] = append(list, &m)
	parser.todo[end] = append(parser.todo[end], &m)
}

func (parser *@_Parser) findMatches() error {
	parser.addMatch(#g, 0, 0, nil, nil)
	var savePrefixes []@_Prefix
	for end := range parser.todo {
		savePrefixes = savePrefixes[:0]
		for p := range parser.matches[end] {
			savePrefixes = append(savePrefixes, p)
		}

		var token @_Symbol = -1
		if end < len(parser.tokens) {
			token = @_tokenType(parser.tokens[end])
		}
		for k := 0; k < len(parser.todo[end]); k++ {
			t := parser.todo[end][k]
			for _, p := range @_followers[t.prefix] {
				parser.addMatch(p, end, end, nil, nil)
			}
			for _, e := range @_extensions[t.prefix] {
				if list, have := parser.matches[end][e.by]; have {
					for _, m := range list {
						if m.start == end {
							parser.addMatch(e.to, t.start, end, t, m)
							break
						}
					}
				}
			}
			if s := @_symbolFinished[t.prefix]; s >= 0 {
				for _, e := range @_extendedBy[s] {
					if list, have := parser.matches[t.start][e.from]; have {
						for _, m := range list {
							parser.addMatch(e.to, m.start, end, m, t)
						}
					}
				}
			}
			if token >= 0 {
				for _, e := range @_extendedBy[token] {
					if list, have := parser.matches[end][e.from]; have {
						for _, m := range list {
							parser.addMatch(e.to, m.start, end+1, m, nil)
						}
					}
				}
			}
		}
		if token >= 0 && len(parser.todo[end+1]) == 0 {
			return gleanerrors.Unexpected{gleanerrors.MakeLocation(parser.tokens, end)}
		}
	}
	parser.endPrefixes = savePrefixes
	return nil
}

func (parser *@_Parser) ambiguous(m1, m2 *@_Match) error {
	return gleanerrors.Ambiguous{
		gleanerrors.MakeRange(parser.tokens, m1.start, m1.end-1),
		@_ruledesc[@_prefix2rule[m1.completePrefix]],
		@_ruledesc[@_prefix2rule[m2.completePrefix]],
	}
}

func (parser *@_Parser) findTrace() error {
	n := len(parser.tokens)
	var goalmatch *@_Match
	for _, p := range @_goalPrefixes {
		if list, have := parser.matches[n][p]; have {
			for _, m := range list {
				if m.start == 0 {
					m.completePrefix = m.prefix
					if goalmatch == nil {
						goalmatch = m
					} else {
						return parser.ambiguous(goalmatch, m)
					}
					break
				}
			}
		}
	}
	if goalmatch == nil {
		return gleanerrors.Unexpected{gleanerrors.Location{len(parser.tokens), nil}}
	}

	parser.trace = parser.trace[:0]
	parser.trace = append(parser.trace, @_appliers[goalmatch.prefix])

	var stack []*@_Match
	stack = append(stack, goalmatch)
	for len(stack) > 0 {
		m := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if m.shorter != nil {
			m.shorter.completePrefix = m.completePrefix
		}
		if m.shorter2 != nil {
			m.shorter2.completePrefix = m.completePrefix
		}
		if m.last != nil {
			m.last.completePrefix = m.last.prefix
		}
		if m.last2 != nil {
			m.last2.completePrefix = m.last2.prefix
		}

		if m.shorter2 != nil || m.last2 != nil {
			if m.shorter2 != nil && m.shorter2 != m.shorter {
				return parser.ambiguous(m, m)
			}
			if m.last2 == nil || m.last2 == m.last {
				panic("bug")
			}
			return parser.ambiguous(m.last, m.last2)
		}

		if m.shorter != nil {
			m.shorter.completePrefix = m.completePrefix
			stack = append(stack, m.shorter)
		}
		if m.last != nil {
			parser.trace = append(parser.trace, @_appliers[m.last.prefix])
			stack = append(stack, m.last)
		} else {
			t := @_lastTerminal[m.prefix]
			if t >= 0 {
				parser.trace = append(parser.trace, @_applyTerminal[t])
			}
		}
	}

	return nil
}
`

// Append the parser type
func (g *Grammar) addParserType() {
	g.addText(`
type @_Parser struct {
	tokens      []interface{}
	matches     []map[@_Prefix][]*@_Match
	todo        [][]*@_Match
	trace       []func(*@_Parser)
	tokensUsed  int
	endPrefixes []@_Prefix

`)
	maxLen := 0
	for _, s := range g.symbols {
		if l := len(s.name); l > maxLen {
			maxLen = l
		}
	}
	for _, s := range g.symbols {
		g.addf("\tstack%-*s []%s\n", maxLen, s.name, s.name)
	}
	g.addString("}\n")
}

// Append the function to apply the trace
func (g *Grammar) addApplyTrace() {
	g.addText(`
func (parser *@_Parser) applyTrace() #G {
	parser.tokensUsed = 0
`)
	for _, s := range g.nonterminals {
		g.addf("\tparser.stack%s = parser.stack%s[:0]\n", s.name, s.name)
	}
	g.addText(`
	for n := len(parser.trace) - 1; n >= 0; n-- {
		parser.trace[n](parser)
	}
	return parser.stack#G[0]
}
`)
}

// For each prefix, write the list of prefixes that can follow it through non-terminals
func (g *Grammar) addFollowers() {
	g.addText("\nvar @_followers = [][]@_Prefix{\n")
	for _, p := range g.prefixes {
		var list []int
		for _, ext := range p.extensions {
			s := ext.rules[0].items[p.length]
			if !s.isTerminal() {
				list = append(list, s.prefix0.id)
			}
		}
		g.addString("\t")
		g.addSlice(list)
		g.addString(",\n")
	}
	g.addString("}\n")
}

// For each prefix, write it's last symbol, if that is a terminal symbol
func (g *Grammar) addLastTerminal() {
	g.addText("\nvar @_lastTerminal = []@_Symbol{\n")
	for _, p := range g.prefixes {
		t := -1
		if p.length > 0 {
			s := p.rules[0].items[p.length-1]
			if s.isTerminal() {
				t = s.id
			}
		}
		g.addf("\t%d,\n", t)
	}
	g.addString("}\n")
}

// For each symbol, write how it can extend other prefixes.
func (g *Grammar) addExtendedBy() {
	g.addText(`
type @_ExtBy struct {
	from, to @_Prefix
}

var @_extendedBy = [][]@_ExtBy{
`)

	ext := make([][][2]int, len(g.symbols))
	for _, p := range g.prefixes {
		for _, q := range p.extensions {
			s := q.rules[0].items[p.length]
			ext[s.id] = append(ext[s.id], [2]int{p.id, q.id})
		}
	}

	for _, e := range ext {
		g.addString("\t{")
		for n, en := range e {
			if n > 0 {
				g.addString(", ")
			}
			g.addSlice(en[:])
		}
		g.addString("},\n")
	}
	g.addString("}\n")
}

// For each prefix, write its extensions by nonterminal symbols
func (g *Grammar) addExtensions() {
	g.addText(`
type @_Extend struct {
	by, to @_Prefix
}

var @_extensions = [][]@_Extend{
`)

	ext := make([][][2]int, len(g.prefixes))
	for _, p := range g.prefixes {
		for _, q := range p.extensions {
			s := q.rules[0].items[p.length]
			if !s.isTerminal() {
				for _, r := range s.rules {
					ext[p.id] = append(ext[p.id], [2]int{r.fullPrefix.id, q.id})
				}
			}
		}
	}

	for _, e := range ext {
		g.addString("\t{")
		for n, en := range e {
			if n > 0 {
				g.addString(", ")
			}
			g.addSlice(en[:])
		}
		g.addString("},\n")
	}
	g.addString("}\n")
}

// For each prefix that is a complete rule, write the symbol id.
func (g *Grammar) addSymbolFinished() {
	g.addText("\nvar @_symbolFinished = []int{\n")
	for _, p := range g.prefixes {
		r := p.completedRule()
		if r != nil {
			g.addf("\t%d,\n", r.target.id)
		} else {
			g.addf("\t-1,\n")
		}
	}
	g.addString("}\n")
}

// Add the function to determine a terminal's symbol id
func (g *Grammar) addTokenType() {
	g.addText(`
func @_tokenType(t interface{}) @_Symbol {
	switch t.(type) {
`)
	for _, s := range g.terminals {
		g.addf("\tcase %s:\n\t\treturn %d\n", s.name, s.id)
	}
	g.addString(
		`	default:
		panic(fmt.Sprintf("input token (type %T) is not a terminal symbol", t))
	}
}
`)
}

// Add the list of prefixes that complete the goal symbol
func (g *Grammar) addGoalPrefixes() {
	g.addText("\nvar @_goalPrefixes = []@_Prefix{\n")
	for _, r := range g.goal.rules {
		g.addf("\t%d,\n", r.fullPrefix.id)
	}
	g.addString("}\n")
}

// Add the functions to apply terminals (copy to the appropriate stack)
func (g *Grammar) addApplyTerminal() {
	g.addText("\nvar @_applyTerminal = []func(*@_Parser){\n")
	for _, t := range g.terminals {
		g.addText("\tfunc(parser *@_Parser) {\n")
		stack := "parser.stack" + t.name
		g.addf("\t\t%s = append(%s, parser.tokens[parser.tokensUsed].(%s))\n", stack, stack, t.name)
		g.addf("\t\tparser.tokensUsed++\n")
		g.addString("\t},\n")
	}
	g.addString("}\n")
}

// Add the functions to apply rules
func (g *Grammar) addAppliers() {
	g.addText("\nvar @_appliers = []func(*@_Parser){\n")
	for _, p := range g.prefixes {
		r := p.completedRule()
		if r == nil {
			g.addString("\tnil,\n")
			continue
		}
		g.addText("\tfunc(parser *@_Parser) {\n")

		for n := len(r.items) - 1; n >= 0; n-- {
			s := r.items[n]
			g.addf("\t\tx%d := parser.stack%s[len(parser.stack%s)-1]\n", n, s.name, s.name)
			g.addf("\t\tparser.stack%s = parser.stack%s[:len(parser.stack%s)-1]\n", s.name, s.name, s.name)
		}
		g.addf("\t\ty := %s(", r.name)
		if len(r.items) > 0 {
			g.addString("x0")
			for n := 1; n < len(r.items); n++ {
				g.addf(", x%d", n)
			}
		}
		g.addString(")\n")
		g.addf("\t\tparser.stack%s = append(parser.stack%s, y)\n", r.target.name, r.target.name)

		g.addString("\t},\n")
	}
	g.addString("}\n")
}

// Add the mapping of prefix to completed rule
func (g *Grammar) addPrefix2Rule() {
	g.addText("\nvar @_prefix2rule = []@_Rule{\n")
	for _, p := range g.prefixes {
		n := -1
		if r := p.completedRule(); r != nil {
			n = r.id
		}
		g.addf("\t%d,\n", n)
	}
	g.addString("}\n")
}

// Add the rule descriptions
func (g *Grammar) addRuleDescriptions() {
	g.addText(`
var @_ruledesc = []gleanerrors.Rule{
`)
	for _, r := range g.rules {
		g.addText("\tgleanerrors.Rule{")
		g.addf("\"%s\", \"%s\", []string{", r.name, r.target.name)
		for n, i := range r.items {
			if n > 0 {
				g.addString(", ")
			}
			g.addf(`"%s"`, i.name)
		}
		g.addString("}},\n")
	}
	g.addString("}\n")
}
