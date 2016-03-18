package cli

type SousVersionCommand struct {
	Version Version
}

const sousVersionHelp = `
version prints the current version and revision of sous
`

func (v *SousVersionCommand) Help() string {
	return sousVersionHelp
}

func (vc *SousVersionCommand) Execute(args []string) Result {
	v := vc.Version.Format("")
	return Successf("sous version %s", v)
}
