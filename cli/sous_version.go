package cli

type SousVersion struct {
	Sous *Sous
}

const sousVersionHelp = `
show version information

args:

version prints the current version of sous. Please include the output from this
command with any bug reports sent to https://github.com/opentable/sous/issues
`

func (*SousVersion) Help() *Help {
	return ParseHelp(sousVersionHelp)
}

func (sv *SousVersion) Execute(args []string, out, errout Output) Result {
	return Successf("sous version %s", sv.Sous.Version)
}
