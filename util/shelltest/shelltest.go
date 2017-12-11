package shelltest

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/opentable/sous/util/whitespace"
)

type (
	// A ShellTest is a context for executing CLI commands and testing the results
	ShellTest struct {
		seq         *int
		t           *testing.T
		writeDir    string
		shell       *captiveShell
		tmplContext interface{}
	}

	// A CheckFn receives the Result of running a script, and should inspect it,
	// calling methods on the testing.T as appropriate
	CheckFn func(string, Result, *testing.T)
)

// New creates a new ShellTest
func New(t *testing.T, name string, ctx interface{}, env map[string]string) *ShellTest {
	sh, err := newShell(env)
	if err != nil {
		t.Fatal(err)
		sh = nil
	}
	seq := 0
	return &ShellTest{
		seq:         &seq,
		t:           t,
		shell:       sh,
		tmplContext: ctx,
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
	st.shell.WriteTo(dir)
	return nil
}

// DebugPrefix directs the ShellTest to write all received bytes to debug files
// in TempDir/Prefix
func (st *ShellTest) DebugPrefix(prefix string) {
	st.shell.stdout.debugTo("stdout", prefix)
	st.shell.stderr.debugTo("stderr", prefix)
	st.shell.scriptEnv.debugTo("scriptEnv", prefix)
	st.shell.scriptErrs.debugTo("scriptErrs", prefix)
}

// Template processes a tmplSrc as a text/template with the ShellTest's
// template context. Useful for user advice, for instance.
func (st *ShellTest) Template(name, tmplSrc string) (string, error) {

	clean := whitespace.CleanWS(tmplSrc)
	tmpl, err := template.New(name).Parse(clean)
	if err != nil {
		return "", err
	}
	buf := &bytes.Buffer{}
	if err = tmpl.Execute(buf, st.tmplContext); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// Block runs a block of shell script, returning a new ShellTest. If the check
// function includes a failing test, however, blocks run on the resulting
// ShellTest will be skipped.
func (st *ShellTest) Block(name, oldScript string, check ...CheckFn) *ShellTest {
	if st.shell == nil { // When a shell fails, follow-on blocks aren't run
		return st
	}

	ran := st.t.Run(name, func(t *testing.T) {
		(*st.seq)++
		blockName := fmt.Sprintf("%03d_%s", (*st.seq), name)
		st.shell.BlockName(blockName)
		newScript, err := st.Template(name, oldScript)
		if err != nil {
			t.Fatalf("Script loading err: %v", err)
		}
		res, err := st.shell.Run(newScript)
		if err != nil {
			if st.writeDir != "" {
				t.Logf("Shell script, output and errors written to %q.", filepath.Join(st.writeDir, blockName))
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
				t.Logf("Shell script, output and errors written to %q.", filepath.Join(st.writeDir, blockName))
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
		seq:         st.seq,
		t:           st.t,
		shell:       shell,
		writeDir:    st.writeDir,
		tmplContext: st.tmplContext,
	}
}
