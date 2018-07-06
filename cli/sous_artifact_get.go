package cli

import (
	"flag"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousGetArtifact is the description of the `sous add artifacat` command
type SousGetArtifact struct {
	SousGraph *graph.SousGraph
	opts      graph.ArtifactOpts
}

func init() { AddSubcommands["artifact"] = &SousGetArtifact{} }

// Help prints the help.
func (*SousGetArtifact) Help() string {
	return `Get artifact of docker image.

Tell sous that this docker image represents a particular SourceID.
`
}

// AddFlags adds the flags.
func (sa *SousGetArtifact) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &sa.opts.SourceID, AddArtifactFlagsHelp)

	fs.StringVar(&sa.opts.DockerImage, "docker-image", "",
		"the docker image to store as an artifact")

}

// Execute defines the behavior of 'sous add artifact'.
func (sa *SousGetArtifact) Execute(args []string) cmdr.Result {

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
