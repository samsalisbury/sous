package shelltest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/opentable/sous/util/whitespace"
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
	dir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dir, os.ModeDir|os.ModePerm)
	if err != nil {
		return err
	}

	st.writeDir = dir
	return nil
}

// DebugPrefix directs the ShellTest to write all received bytes to debug files
// in TempDir/Prefix
func (st *ShellTest) DebugPrefix(prefix string) {
	st.shell.stdout.debugTo(prefix)
	st.shell.stderr.debugTo(prefix)
	st.shell.scriptEnv.debugTo(prefix)
	st.shell.scriptErrs.debugTo(prefix)
}

// Block runs a block of shell script, returning a new ShellTest. If the check
// function includes a failing test, however, blocks run on the resulting
// ShellTest will be skipped.
func (st *ShellTest) Block(name, script string, check ...CheckFn) *ShellTest {
	if st.shell == nil { // When a shell fails, follow-on blocks aren't run
		return st
	}
	ran := st.t.Run(name, func(t *testing.T) {
		res, err := st.shell.Run(whitespace.CleanWS(script))
		if st.writeDir != "" {
			res.WriteTo(st.writeDir, name)
		}
		if err != nil {
			if st.writeDir != "" {
				t.Logf("Shell script, output and errors written to %q.", st.writeDir)
			} else {
				t.Logf("No output directory set. No shell artifacts recorded.")
			}
			t.Fatal("Error: ", err)
		}
		if len(check) > 0 {
			check[0](name, res, t)
		}
		if t.Failed() {
			if st.writeDir != "" {
				t.Logf("Shell script, output and errors written to %q.", st.writeDir)
			} else {
				t.Logf("No output directory set. No shell artifacts recorded.")
			}
		}
	})
	shell := st.shell

	if !ran {
		shell = nil
	}

	return &ShellTest{
		t:        st.t,
		shell:    shell,
		writeDir: st.writeDir,
	}
}