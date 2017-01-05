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

func TestFindBottomCmd(t *testing.T) {
	// TestFindBottomCmd relies on &TestCommandWithSubcommands having
	// subcommands that are plain commands that do not satisfy
	// the Subcommander interface.
	c := &TestCommandWithSubcommands{}

	bcp := findBottomCommand(c, []string{"test", "nonCommandArg"})
	bc := *bcp
	if _, ok := bc.(Subcommander); ok {
		t.Fatal("The bottom command was not expected to have subcommands")
	} else {
		t.Log("The bottom command correctly has no subcommands.")
	}

	bscp := findBottomCommand(c, []string{"not-command"})
	bsc := *bscp
	if _, ok := bsc.(Subcommander); ok {
		t.Log("The provided argument is not a subcommand, thus the top level command is also the bottom command")
	} else {
		t.Fatal("The provided argument is not a subcommand, so the bottom command should have been the top command.")
	}

}
