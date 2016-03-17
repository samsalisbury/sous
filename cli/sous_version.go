package cli

type SousVersionCommand struct {
	Version SousVersion
}

const sousVersionHelp = `
version prints the current version and revision of sous
`

func (v *SousVersionCommand) Help() string {
	return sousVersionHelp
}

func (vc *SousVersionCommand) Execute(args []string) Result {
	return Success(vc.Version)
}
