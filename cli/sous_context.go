package cli

import "github.com/opentable/sous/util/cmdr"

// SousContext is the 'sous context' command.
type SousContext struct {
	SourceContextFunc
}

func init() { TopLevelCommands["context"] = &SousContext{} }

const sousContextHelp = `
show the current build context

context prints out sous's view of your current context

args:
`

// Help provides help for sous context.
func (*SousContext) Help() string { return sousContextHelp }

// Execute prints the detected sous context.
func (sv *SousContext) Execute(args []string) cmdr.Result {
	sc, err := sv.SourceContextFunc()
	if err != nil {
		return EnsureErrorResult(err)
	}
	return SuccessYAML(sc)
}
