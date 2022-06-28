// Copyright 2021 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package earley

import (
	"sort"
)

// A grammar symbol
type symbol struct {
	name    string
	rules   []*rule
	id      int
	prefix0 *prefix
}

// Terminal symbols are not produced by any rules
func (s *symbol) isTerminal() bool {
	return len(s.rules) == 0
}

// Sort a symbol's rules lexicographically, so rules with common prefixes are together.
func (s *symbol) sortRules() {
	sort.Slice(s.rules, func(i, j int) bool {
		u := s.rules[i].items
		v := s.rules[j].items
		for n := 0; ; n++ {
			if n >= len(v) {
				return false
			} else if n >= len(u) {
				return true
			} else {
				un := u[n].id
				vn := v[n].id
				if un != vn {
					return un < vn
				}
			}
		}
	})
}
