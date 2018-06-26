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
	return `Artifact for docker image to deploy.

This is to add the artifact to sous for deploy.
`
}

// AddFlags adds the flags for sous init.
func (sqa *SousAddArtifact) AddFlags(fs *flag.FlagSet) {

	//MustAddFlags(fs, &sqa.opts.DFF, AddArtificatFlagsHelp)
	fs.StringVar(&sqa.opts.Repo, "repo", "",
		"repo for this artifact")
	fs.StringVar(&sqa.opts.Cluster, "cluster", "",
		"cluster for this artifact")
}

// Execute defines the behavior of `sous plumbing normalizegdm`
func (sqa *SousAddArtifact) Execute(args []string) cmdr.Result {

	addArtifact, err := sqa.SousGraph.GetArtifact(sqa.opts)
	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	if err := addArtifact.Do(); err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success("Artifact added.")
}
