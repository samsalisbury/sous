package cmdr

import (
	"bytes"
	"strings"
	"testing"
)

type TestCommand struct{}

func (tc *TestCommand) Help() string { return "" }

func (tc *TestCommand) Execute(args []string) Result {
	return Success("Congratulations, caller:", args)
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
