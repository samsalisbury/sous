package cmdr

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelp(t *testing.T) {
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	c := &CLI{
		Root: &TestCommandWithSubcommands{},
		Out:  NewOutput(outBuf),
		Err:  NewOutput(errBuf),
	}

	help, err := c.Help(c.Root, []string{})
	if err != nil {
		t.Fatal(err)
	}
	expectedStrings := []string{
		"subcommands:",
		"cmd       Test Command.",
	}
	for _, s := range expectedStrings {
		if strings.Contains(help, s) {
			t.Logf("Found expected output: %s", s)
		} else {
			t.Fatalf("Could not find %s in %s", s, help)
		}
	}
}
