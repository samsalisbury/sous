package tests

import (
	"testing"

	"github.com/opentable/sous2/cli"
)

func TestSous(t *testing.T) {
	s := &cli.Sous{}

	CanExecute(s)
	HasSubcommands(s)
}
