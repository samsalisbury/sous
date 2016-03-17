package cli

import (
	"strings"
	"testing"
)

func TestCli(t *testing.T) {

	c := &CLI{}

	args := makeArgs("sous")

	result := c.invoke(&Sous{}, args, nil)

	actual := result.ExitCode()
	expected := 0

	if result.ExitCode() != 0 {
		t.Errorf("got exit code %d; want %d", actual, expected)
		t.Error(result)
	}

}

func makeArgs(s string) []string {
	return strings.Split(s, " ")
}
