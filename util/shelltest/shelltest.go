package shelltest

import (
	"os"
	"testing"
)

type (
	// A ShellTest is a context for executing CLI commands and testing the results
	ShellTest struct {
		t              *testing.T
		name, writeDir string
		shell          *CaptiveShell
	}

	// A CheckFn receives the Result of running a script, and should inspect it,
	// calling methods on the testing.T as appropriate
	CheckFn func(string, Result, *testing.T)
)

// New creates a new ShellTest
func New(t *testing.T, name string, env map[string]string) *ShellTest {
	sh, err := NewShell(env)
	if err != nil {
		t.Fatal(err)
		sh = nil
	}
	return &ShellTest{
		t:     t,
		shell: sh,
	}
}

// WriteTo directs the ShellTest to write details of its execution into the
// passed directory.
func (st *ShellTest) WriteTo(dir string) error {
	err := os.MkdirAll(dir, os.ModeDir|os.ModePerm)
	if err != nil {
		return err
	}

	st.writeDir = dir
	return nil
}

// Block runs a block of shell script, returning a new ShellTest. If the check
// function includes a failing test, however, blocks run on the resulting
// ShellTest will be skipped.
func (st *ShellTest) Block(name, script string, check ...CheckFn) *ShellTest {
	if st.shell == nil { // When a shell fails, follow-on blocks aren't run
		return st
	}
	ran := st.t.Run(name, func(t *testing.T) {
		res, err := st.shell.Run(script)
		if err != nil {
			t.Fatal(err)
		}
		if st.writeDir != "" {
			res.WriteTo(st.writeDir, name)
		}
		if len(check) > 0 {
			check[0](name, res, t)
		}
	})
	shell := st.shell

	if !ran {
		shell = nil
	}

	return &ShellTest{
		t:     st.t,
		shell: shell,
	}
}
