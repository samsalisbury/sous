package cmdr

import (
	"bytes"
	"strings"
	"testing"
)

type TestCommand struct{}

func (tc *TestCommand) Help() string { return "Test Command." }

func (tc *TestCommand) Execute(args []string) Result {
	return Success("Congratulations, caller:", args)
}

type TestCommandWithSubcommands struct{}

func (tc *TestCommandWithSubcommands) Help() string { return "Test command with subcommands." }

func (tc *TestCommandWithSubcommands) Subcommands() Commands {
	cmds := make(Commands)
	cmds["test"] = &TestCommand{}
	cmds["anothertest"] = &TestCommand{}
	return cmds
}

func TestCli(t *testing.T) {

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	c := &CLI{
		Root: &TestCommand{},
		Out:  NewOutput(outBuf),
		Err:  NewOutput(errBuf),
	}

	args := makeArgs("a-command")

	result := c.Invoke(args)

	actual := result.ExitCode()
	expected := 0

	if result.ExitCode() != 0 {
		t.Errorf("got exit code %d; want %d", actual, expected)
		t.Error(result)
	} else {
		t.Logf("got exit code %d; want %d", actual, expected)
	}

	success, isSuccess := result.(SuccessResult)
	if !isSuccess {
		t.Errorf("got a %T; want %T", result, SuccessResult{})
	} else {
		t.Logf("got a %T; want %T", result, SuccessResult{})
	}

	commandOut := string(success.Data)
	expectedCommandOut := "Congratulations, caller: []\n"

	if commandOut != expectedCommandOut {
		t.Errorf("got %q; want %q", commandOut, expectedCommandOut)
	} else {
		t.Logf("got %q; want %q", commandOut, expectedCommandOut)
	}

	cliOut := outBuf.String()
	if cliOut != expectedCommandOut {
		t.Errorf("got %q; want %q", cliOut, expectedCommandOut)
	} else {
		t.Logf("got %q; want %q", cliOut, expectedCommandOut)
	}

	if errBuf.Len() != 0 {
		t.Errorf("unexpected write to stderr: %q", errBuf)
	} else {
		t.Log("errBuf is empty.")
	}

}

func makeArgs(s string) []string {
	return strings.Split(s, " ")
}

func TestPrintTip(t *testing.T) {
	logTemplate := "got %q, want %q"
	testTable := make(map[string]string)
	testTable[""] = ""
	testTable["test"] = "Tip: test\n"
	for k, v := range testTable {
		outBuf := &bytes.Buffer{}
		errBuf := &bytes.Buffer{}

		c := &CLI{
			Root: &TestCommand{},
			Out:  NewOutput(outBuf),
			Err:  NewOutput(errBuf),
		}

		c.printTip(k)
		out := errBuf.String()
		if out != v {
			t.Errorf(logTemplate, out, v)
		} else {
			t.Logf(logTemplate, out, v)
		}
	}
}

func TestListSubcommands(t *testing.T) {
	logTemplate := "CLI with subcommands has ListSubcommands length of %d"
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	c := &CLI{
		Root: &TestCommandWithSubcommands{},
		Out:  NewOutput(outBuf),
		Err:  NewOutput(errBuf),
	}

	subcommandLength := len(c.ListSubcommands(c.Root))
	if subcommandLength == 0 {
		t.Fatalf(logTemplate, subcommandLength)
	} else {
		t.Logf(logTemplate, subcommandLength)
	}
}
