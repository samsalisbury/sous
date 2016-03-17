package cli

type SousHelp struct {
	*CLI
}

const sousHelpHelp = `
usage: sous help <command>
`

func (sc *SousHelp) Help() string { return sousHelpHelp }

func (sc *SousHelp) Execute(args []string) Result {

	if len(args) == 0 {
		sc.Info(sousHelpHelp)
		return Success()
	}

	return InternalErrorf(nil, "sous help is not yet implemented")
}
