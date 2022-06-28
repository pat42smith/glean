// Copyright 2021 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package earley

// A prefix to one or more rules for the same symbol.
type prefix struct {
	target     *symbol
	length     int // Length of the prefix; 0 to length of longest rule
	rules      []*rule
	id         int
	extensions []*prefix // Prefixes that are longer by one symbol
}

// The rule completely represented by a prefix, if any
func (p *prefix) completedRule() *rule {
	if len(p.rules[0].items) == p.length {
		return p.rules[0]
	}
	return nil
}
