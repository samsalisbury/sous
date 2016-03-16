package cli

import "flag"

type (
	Command interface {
		Help() string
	}
	CanExecute interface {
		Execute() Result
	}
	AddsFlags interface {
		AddFlags(*flag.FlagSet)
	}
	HasSubcommand interface {
		Subcommand(name string) (Command, error)
	}
)

func getCommand(name string) Command {
	switch name {
	default:
		return &SuggestCommand{name}
	case "version":
		return &VersionCommand{}
	}
}
