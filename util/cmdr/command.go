package cmdr

import (
	"flag"
	"sort"
)

type (
	// Commands is a mapping of strings (the command name) to commands.
	Commands map[string]Command
	// Command is a command that can be invoked.
	Command interface {
		// Help is the help message for a command. To be a command, all you need
		// is a help message.
		//
		// Help messages must follow some conventions:
		// The first line must be 50 characters or fewer, and describe
		// succinctly what the command does.
		// The second line must be blank.
		// The third line should begin with "args: " followed by a list of named
		// arguments (not flags or options)
		// The remaining non-blank lines should contain a detailed description
		// of how the command works, including usage examples.
		Help() string
	}
	// Executor is a command can itself be executed to do something.
	Executor interface {
		Execute(args []string) Result
	}
	// Subcommander means this command has subcommands.
	Subcommander interface {
		// Subcommands returns a map of command names to Commands.
		Subcommands() Commands
	}
	// AddsFlags means this command has flags to add. These flags will be
	// available not only to this command, but will still be read, and have
	// impact on this command, even if the user applies them to deeper
	// subcommands.
	AddsFlags interface {
		// AddFlags will be passed a flag.FlagSet already named correctly for
		// this command. All you need to do is add whatever flag definitions
		// apply to this command.
		AddFlags(*flag.FlagSet)
	}
)

// SortedKeys returns the names of the commands in alphabetical order.
func (cs Commands) SortedKeys() []string {
	keys := make([]string, len(cs))
	i := 0
	for k := range cs {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}
