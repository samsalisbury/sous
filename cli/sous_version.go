package cli

import "github.com/opentable/sous/util/cmdr"

// SousVersion is the 'sous version' command.
type SousVersion struct {
	Sous *Sous
}

func init() { TopLevelCommands["version"] = &SousVersion{} }

const sousVersionHelp = `print the version of sous

prints the current version of sous. Please include the output from this
command with any bug reports sent to https://github.com/opentable/sous/issues
`

// Help returns help for sous version.
func (*SousVersion) Help() string { return sousVersionHelp }

// Execute runs the 'sous version' command.
func (sv *SousVersion) Execute(args []string) cmdr.Result {
	out := `sous version %s (%s %s/%s)`
	s := sv.Sous
	return cmdr.Successf(out, s.Version, s.GoVersion, s.OS, s.Arch)
}
