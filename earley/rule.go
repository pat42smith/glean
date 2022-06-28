// Copyright 2021 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package earley

// A grammar rule
type rule struct {
	name       string
	target     *symbol
	items      []*symbol
	id         int
	fullPrefix *prefix
}
