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
		Help() string
	}
	// CanExecute means the command can itself be executed to do something.
	Executor interface {
		Execute(args []string) Result
	}
	// HasSubcommands means this command has subcommands.
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
