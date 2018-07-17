package cli

import (
	"flag"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousArtifactGet is the description of the `sous add artifacat` command
type SousArtifactGet struct {
	SousGraph *graph.SousGraph
	opts      graph.ArtifactOpts
}

func init() { ArtifactSubcommands["get"] = &SousArtifactGet{} }

// Help prints the help.
func (*SousArtifactGet) Help() string {
	return `Get artifact of docker image.

Tell sous that this docker image represents a particular SourceID.
`
}

// AddFlags adds the flags.
func (sa *SousArtifactGet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sa.opts.SourceID, AddArtifactFlagsHelp)
}

// RegisterOn adds flag-derived values to the graph.
func (sa *SousArtifactGet) RegisterOn(psy Addable) {
	dff := sa.opts.SourceID.DeployFilterFlags()
	psy.Add(&dff)
}

// Execute defines the behavior of 'sous add artifact'.
func (sa *SousArtifactGet) Execute(args []string) cmdr.Result {

	if sa.opts.SourceID.Tag == "" {
		return cmdr.UsageErrorf("-tag flag required")
	}
	sid, err := sa.opts.SourceID.SourceID()
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}
	if sid.Location.Repo == "" {
		return cmdr.UsageErrorf("-repo flag required")
	}

	getArtifact, err := sa.SousGraph.GetGetArtifact(sa.opts)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := getArtifact.Do(); err != nil {
		return EnsureErrorResult(err)
	}
	return cmdr.Success("Artifact retrieved.")
}
