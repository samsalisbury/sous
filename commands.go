package main

type (
	Command interface {
		Help() string
		Execute() error
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
