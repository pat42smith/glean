// Copyright 2022 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.
//
// A small test file to be installed beside lister.go. It contains no tests;
// its purpose is to demonstrate that glean does not scan _test.go files.

package main_test

// This rule should cause an ambiguity if processed by glean with the default target.
func ruleAmbiguous(s Sorted) Target {
	return Target(s)
}
