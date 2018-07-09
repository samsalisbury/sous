package cli

import (
	"flag"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousArtifactAdd defines the `sous artifact add` command
type SousArtifactAdd struct {
	SousGraph *graph.SousGraph
	opts      graph.ArtifactOpts
}

func init() { ArtifactSubcommands["add"] = &SousArtifactAdd{} }

// Help prints the help.
func (*SousArtifactAdd) Help() string {
	return `Add artifact of docker image.

Tell sous that this docker image represents a particular SourceID.
`
}

// AddFlags adds the flags.
func (sa *SousArtifactAdd) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sa.opts.SourceID, AddArtifactFlagsHelp)

	fs.StringVar(&sa.opts.DockerImage, "docker-image", "",
		"the docker image to store as an artifact")

}

// Execute defines the behavior of 'sous add artifact'.
func (sa *SousArtifactAdd) Execute(args []string) cmdr.Result {

	if sa.opts.DockerImage == "" {
		return cmdr.UsageErrorf("-docker-image flag required")
	}
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

	addArtifact, err := sa.SousGraph.GetAddArtifact(sa.opts)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := addArtifact.Do(); err != nil {
		return EnsureErrorResult(err)
	}
	return cmdr.Success("Artifact added.")
}
