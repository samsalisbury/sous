package cli

import (
	"bytes"
	"strings"
	"testing"
)

type TestCommand struct{}

func (tc *TestCommand) Help() *Help { return ParseHelp("") }
func (tc *TestCommand) Execute(args []string) Result {
	return Success("Congratulations, caller:", args)
}

func TestCli(t *testing.T) {

	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}

	c := &CLI{
		OutWriter: outBuf,
		ErrWriter: errBuf,
	}

	args := makeArgs("sous")

	result := c.Invoke(&TestCommand{}, args)

	actual := result.ExitCode()
	expected := 0

	if result.ExitCode() != 0 {
		t.Errorf("got exit code %d; want %d", actual, expected)
		t.Error(result)
	}

	success, isSuccess := result.(SuccessResult)
	if !isSuccess {
		t.Errorf("got a %T; want %T", result, SuccessResult{})
	}

	commandOut := string(success.Data)
	expectedCommandOut := "Congratulations, caller: []\n"

	if commandOut != expectedCommandOut {
		t.Errorf("got %q; want %q", commandOut, expectedCommandOut)
	}

	cliOut := outBuf.String()
	if cliOut != expectedCommandOut {
		t.Errorf("got %q; want %q", cliOut, expectedCommandOut)
	}

	if errBuf.Len() != 0 {
		t.Errorf("unexpected write to stderr: %q", errBuf)
	}

}

func makeArgs(s string) []string {
	return strings.Split(s, " ")
}
