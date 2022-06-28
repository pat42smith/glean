// Copyright 2022 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func runCommandIn(t *testing.T, dir string, cmd string, args ...string) []byte {
	t.Helper()
	command := exec.Command(cmd, args...)
	command.Dir = dir
	out, e := command.CombinedOutput()
	if e != nil {
		t.Fatal(e, "with output:", string(out))
	}
	return out
}

var gocmd string

func init() {
	gocmd = runtime.GOROOT() + "/bin/go"
	if runtime.GOOS == "windows" {
		gocmd += ".exe"
	}
}

// Test the various usage options
func TestUsage(t *testing.T) {
	tmp := t.TempDir()

	gleanCmd := filepath.Join(tmp, "glean")
	out := runCommand(t, gocmd, "build", "-o", gleanCmd)
	if len(out) > 0 {
		t.Fatal("unexpected output building glean:", string(out))
	}

	// We will want to run glean from a directory under tmp, and it should use the
	// gleanerrors package from this copy of glean, so we copy gleanerrors there.
	geDir := filepath.Join(tmp, "gleanerrors")
	geFile := filepath.Join(geDir, "gleanerrors.go")
	geText, e := os.ReadFile("../gleanerrors/gleanerrors.go")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.Mkdir(geDir, 0700); e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(geFile, geText, 0444); e != nil {
		t.Fatal(e)
	}

	// And we need a go.mod file to direct to the right place
	modFile := filepath.Join(tmp, "go.mod")
	if e = os.WriteFile(modFile, []byte("module github.com/pat42smith/glean\n"), 0644); e != nil {
		t.Fatal(e)
	}

	mainText, e := os.ReadFile("testdata/lister.go")
	if e != nil {
		t.Fatal(e)
	}

	t.Run("Defaults", func(t2 *testing.T) {
		tryDefaults(t2, tmp, mainText)
	})
	t.Run("Output", func(t2 *testing.T) {
		tryOutput(t2, tmp, mainText)
	})
	t.Run("Target", func(t2 *testing.T) {
		tryTarget(t2, tmp, mainText)
	})
	t.Run("Prefix", func(t2 *testing.T) {
		tryPrefix(t2, tmp, mainText)
	})
	t.Run("Replace", func(t2 *testing.T) {
		tryReplace(t2, tmp, mainText)
	})
	t.Run("IgnoreTestFiles", func(t2 *testing.T) {
		tryIgnoreTestFiles(t2, tmp, mainText)
	})
	t.Run("Help", func(t2 *testing.T) {
		tryHelp(t2, tmp)
	})
}

func tryDefaults(t *testing.T, tmp string, mainText []byte) {
	dir := filepath.Join(tmp, "defaults")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}

	mainGo := filepath.Join(dir, "main.go")
	if e := os.WriteFile(mainGo, mainText, 0444); e != nil {
		t.Fatal(e)
	}

	if out := runCommandIn(t, dir, "../glean"); len(out) > 0 {
		t.Fatal(string(out))
	}
	if info, e := os.Lstat(filepath.Join(dir, "parse.go")); e != nil {
		t.Fatal(e)
	} else if !info.Mode().IsRegular() {
		t.Fatal("parse.go is not a regular file:", info.Mode())
	}
	if out := runCommandIn(t, dir, gocmd, "build"); len(out) > 0 {
		t.Fatal(string(out))
	}
	out := runCommandIn(t, dir, "./defaults", "3", "1", "2")
	if string(out) != "[1 2 3]\n" {
		t.Fatal(string(out))
	}
}

func tryOutput(t *testing.T, tmp string, mainText []byte) {
	dir := filepath.Join(tmp, "output")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}

	mainGo := filepath.Join(dir, "main.go")
	if e := os.WriteFile(mainGo, mainText, 0444); e != nil {
		t.Fatal(e)
	}

	if out := runCommandIn(t, dir, "../glean", "-o", "myparser.go"); len(out) > 0 {
		t.Fatal(string(out))
	}
	if info, e := os.Lstat(filepath.Join(dir, "myparser.go")); e != nil {
		t.Fatal(e)
	} else if !info.Mode().IsRegular() {
		t.Fatal("myparser.go is not a regular file:", info.Mode())
	}
	if out := runCommandIn(t, dir, gocmd, "build"); len(out) > 0 {
		t.Fatal(string(out))
	}
	out := runCommandIn(t, dir, "./output", "17", "-5", "99", "0")
	if string(out) != "[-5 0 17 99]\n" {
		t.Fatal(string(out))
	}
}

func tryTarget(t *testing.T, tmp string, mainText []byte) {
	dir := filepath.Join(tmp, "target")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}

	mainGo := filepath.Join(dir, "main.go")
	if e := os.WriteFile(mainGo, mainText, 0444); e != nil {
		t.Fatal(e)
	}

	if out := runCommandIn(t, dir, "../glean", "-t", "Adder"); len(out) > 0 {
		t.Fatal(string(out))
	}
	if out := runCommandIn(t, dir, gocmd, "build"); len(out) > 0 {
		t.Fatal(string(out))
	}
	out := runCommandIn(t, dir, "./target", "3", "1", "2")
	if string(out) != "6\n" {
		t.Fatal(string(out))
	}
}

func tryPrefix(t *testing.T, tmp string, mainText []byte) {
	dir := filepath.Join(tmp, "prefix")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}

	mainText = bytes.ReplaceAll(mainText, []byte("_glean_"), []byte("xyz"))
	mainGo := filepath.Join(dir, "main.go")
	if e := os.WriteFile(mainGo, mainText, 0444); e != nil {
		t.Fatal(e)
	}

	if out := runCommandIn(t, dir, "../glean", "-p", "xyz"); len(out) > 0 {
		t.Fatal(string(out))
	}

	if parserText, e := os.ReadFile(filepath.Join(dir, "parse.go")); e != nil {
		t.Fatal(e)
	} else if bytes.Contains(parserText, []byte("_glean_")) {
		t.Fatal("parse.go contains the string '_glean_'")
	} else if !bytes.Contains(parserText, []byte("xyz")) {
		t.Fatal("parse.go does not contain the string 'xyz'")
	}

	if out := runCommandIn(t, dir, gocmd, "build"); len(out) > 0 {
		t.Fatal(string(out))
	}
	out := runCommandIn(t, dir, "./prefix", "9", "8", "7")
	if string(out) != "[7 8 9]\n" {
		t.Fatal(string(out))
	}
}

func tryReplace(t *testing.T, tmp string, mainText []byte) {
	dir := filepath.Join(tmp, "replace")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}

	mainGo := filepath.Join(dir, "main.go")
	if e := os.WriteFile(mainGo, mainText, 0444); e != nil {
		t.Fatal(e)
	}

	if out := runCommandIn(t, dir, "../glean"); len(out) > 0 {
		t.Fatal(string(out))
	}
	parseGo := filepath.Join(dir, "parse.go")
	if info, e := os.Lstat(parseGo); e != nil {
		t.Fatal(e)
	} else if !info.Mode().IsRegular() {
		t.Fatal("parse.go is not a regular file:", info.Mode())
	}
	if out := runCommandIn(t, dir, gocmd, "build"); len(out) > 0 {
		t.Fatal(string(out))
	}
	out := runCommandIn(t, dir, "./replace", "3", "1", "2")
	if string(out) != "[1 2 3]\n" {
		t.Fatal(string(out))
	}

	// A file created by glean can be quietly replaced.
	if out = runCommandIn(t, dir, "../glean", "-t", "Adder"); len(out) > 0 {
		t.Fatal(string(out))
	}
	if out = runCommandIn(t, dir, gocmd, "build"); len(out) > 0 {
		t.Fatal(string(out))
	}
	out = runCommandIn(t, dir, "./replace", "3", "1", "2")
	if string(out) != "6\n" {
		t.Fatal(string(out))
	}

	// But a file not containing the "created by glean" marker is not replaced.
	if e := os.WriteFile(parseGo, []byte("some randome text"), 0444); e != nil {
		t.Fatal(e)
	}
	command := exec.Command("../glean")
	command.Dir = dir
	if _, e := command.CombinedOutput(); e == nil {
		t.Fatal("Replaced a regular file not marked created by glean.")
	}

	// Nor is a directory
	if e := os.Remove(parseGo); e != nil {
		t.Fatal(e)
	}
	if e := os.Mkdir(parseGo, 0755); e != nil {
		t.Fatal(e)
	}
	if _, e := command.CombinedOutput(); e == nil {
		t.Fatal("Replaced a directory")
	}

	// Nor a symbolic link
	if e := os.Remove(parseGo); e != nil {
		t.Fatal(e)
	}
	if e := os.Symlink(parseGo+"x", parseGo); e != nil {
		t.Fatal(e)
	}
	if _, e := command.CombinedOutput(); e == nil {
		t.Fatal("Replaced a symbolic link")
	}
}

// *_test.go files are ignored.
func tryIgnoreTestFiles(t *testing.T, tmp string, mainText []byte) {
	dir := filepath.Join(tmp, "ignore")
	if e := os.Mkdir(dir, 0700); e != nil {
		t.Fatal(e)
	}

	mainGo := filepath.Join(dir, "main.go")
	if e := os.WriteFile(mainGo, mainText, 0444); e != nil {
		t.Fatal(e)
	}

	testText, e := os.ReadFile("testdata/lister_test.go")
	if e != nil {
		t.Fatal(e)
	}
	testGo := filepath.Join(dir, "main_test.go")
	if e := os.WriteFile(testGo, testText, 0444); e != nil {
		t.Fatal(e)
	}

	if out := runCommandIn(t, dir, "../glean"); len(out) > 0 {
		t.Fatal(string(out))
	}
	if info, e := os.Lstat(filepath.Join(dir, "parse.go")); e != nil {
		t.Fatal(e)
	} else if !info.Mode().IsRegular() {
		t.Fatal("parse.go is not a regular file:", info.Mode())
	}
	if out := runCommandIn(t, dir, gocmd, "build"); len(out) > 0 {
		t.Fatal(string(out))
	}
	out := runCommandIn(t, dir, "./ignore", "99", "100")
	if string(out) != "[99 100]\n" {
		t.Fatal(string(out))
	}
}

func tryHelp(t *testing.T, tmp string) {
	out := runCommandIn(t, tmp, "./glean", "-h")
	if !bytes.HasPrefix(out, []byte("\nUsage: ")) {
		t.Fatal("Invalid help information:\n", string(out))
	}
}
