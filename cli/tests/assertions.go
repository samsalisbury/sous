package tests

import "github.com/opentable/sous2/cli"

// CanExecute fails the build if the thing passed does not implement the
// CanExecute interface.
func CanExecute(v cli.CanExecute) {}

// HasSubcommands fails the build if the thing passed does not implement the
// HasSubcommands interface.
func HasSubcommands(v cli.HasSubcommands) {}
