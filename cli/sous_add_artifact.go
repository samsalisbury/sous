package cli

import (
	"flag"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousAddArtifact is the description of the `sous add artifacat` command
type SousAddArtifact struct {
	SousGraph *graph.SousGraph
	opts      graph.ArtifactOpts
}

func init() { AddSubcommands["artifact"] = &SousAddArtifact{} }

// Help prints the help
func (*SousAddArtifact) Help() string {
	return `Add artifact of docker image.

Tell sous that this docker image represents a particular SourceID.
`
}

// AddFlags adds the flags for sous init.
func (sa *SousAddArtifact) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sa.opts.SourceID, AddArtificatFlagsHelp)

	fs.StringVar(&sa.opts.DockerImage, "docker-image", "",
		"the docker image to store as an artifact")

}

// Execute defines the behavior of `sous plumbing normalizegdm`
func (sa *SousAddArtifact) Execute(args []string) cmdr.Result {

	addArtifact, err := sa.SousGraph.GetArtifact(sa.opts)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := addArtifact.Do(); err != nil {
		return EnsureErrorResult(err)
	}
	return cmdr.Success("Artifact added.")
}
