package tests

import "github.com/opentable/sous/util/cmdr"

// CanExecute fails the build if the thing passed does not implement the
// CanExecute interface.
func CanExecute(v cmdr.Executor) {}

// HasSubcommands fails the build if the thing passed does not implement the
// HasSubcommands interface.
func HasSubcommands(v cmdr.Subcommander) {}
