// Copyright 2022 Patrick Smith
// Use of this source code is subject to the MIT-style license in the LICENSE file.

package main_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runCommand(t *testing.T, cmd string, args ...string) []byte {
	t.Helper()
	out, e := exec.Command(cmd, args...).CombinedOutput()
	if e != nil {
		t.Fatal(e, "with output:", string(out))
	}
	return out
}

// Test with a small interpreter
func TestInterpreter(t *testing.T) {
	gocmd, e := exec.LookPath("go")
	if e != nil {
		t.Fatal(e)
	}

	tmp := t.TempDir()

	mainGo := filepath.Join(tmp, "main.go")
	mainText, e := os.ReadFile("testdata/interpret.go")
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(mainGo, mainText, 0444); e != nil {
		t.Fatal(e)
	}

	gleanCmd := filepath.Join(tmp, "glean")
	out := runCommand(t, gocmd, "build", "-o", gleanCmd)
	if len(out) > 0 {
		t.Fatal("unexpected output building glean:", string(out))
	}

	parseGo := filepath.Join(tmp, "parse.go")
	out = runCommand(t, gleanCmd, "-t", "Program", "-o", parseGo, mainGo)
	if len(out) > 0 {
		t.Fatal("unexpected output running glean:", string(out))
	}

	interpreter := filepath.Join(tmp, "interpret")
	out = runCommand(t, gocmd, "build", "-o", interpreter, mainGo, parseGo)
	if len(out) > 0 {
		t.Fatal("unexpected output building interpreter:", string(out))
	}

	t.Run("seventeen", func(t2 *testing.T) {
		out := string(runCommand(t2, interpreter, "testdata/seventeen"))
		if out != "17\n" {
			t2.Fatal("wrong output from seventeen:", out)
		}
	})

	t.Run("fact7", func(t2 *testing.T) {
		out := string(runCommand(t2, interpreter, "testdata/fact7"))
		if out != "5040\n" {
			t2.Fatal("wrong output from fact7:", out)
		}
	})

	t.Run("gcd", func(t2 *testing.T) {
		out := string(runCommand(t2, interpreter, "testdata/gcd"))
		s := strings.NewReader(out)
		for i := 1; i <= 30; i++ {
			for j := 1; j <= 30; j++ {
				var u, v, w int
				_, e := fmt.Fscan(s, &u, &v, &w)
				if e != nil {
					t2.Fatal(e)
				}
				if u != i || v != j {
					t2.Fatal("Bad ij: wanted", i, j, "got", u, v)
				}
				ok := i%w == 0 && j%w == 0
				for x := w + 1; ok && x <= i; x++ {
					ok = i%x != 0 || j%x != 0
				}
				if !ok {
					t2.Fatal("Bad gcd:", i, j, "=>", w)
				}
			}
		}
	})
}
