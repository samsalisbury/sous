package cli

import (
	"flag"
	"fmt"
	"io"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/pkg/errors"
	"github.com/samsalisbury/yaml"
)

type SousMetadataGet struct {
	config.DeployFilterFlags
	*sous.ResolveFilter
	graph.TargetManifestID
	graph.HTTPClient
	graph.OutWriter
}

func init() { MetadataSubcommands["get"] = &SousMetadataGet{} }

const sousMetadataGetHelp = `query deployment metadata`

func (*SousMetadataGet) Help() string { return sousMetadataGetHelp }

func (smg *SousMetadataGet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, MetadataFilterFlagsHelp)
}

func (smg *SousMetadataGet) RegisterOn(psy Addable) {
	psy.Add(&smg.DeployFilterFlags)
}

func (smg *SousMetadataGet) Execute(args []string) cmdr.Result {
	mani := sous.Manifest{}
	_, err := smg.HTTPClient.Retrieve("./manifests", smg.TargetManifestID.QueryMap(), &mani, nil)

	if err != nil {
		return EnsureErrorResult(errors.Errorf("No manifest matched by %v.", smg.ResolveFilter))
	}

	for clusterName, deploySpec := range mani.Deployments {
		if !smg.ResolveFilter.FilterClusterName(clusterName) {
			continue
		}
		smg.OutWriter.Write([]byte(fmt.Sprintf("Metadata for deployment in %s\n", clusterName)))
		if err := outputMetadata(deploySpec.Metadata, clusterName, args, smg.OutWriter); err != nil {
			return EnsureErrorResult(err)
		}
	}

	return cmdr.Success()
}

func outputMetadata(metadata sous.Metadata, clusterName string, args []string, out io.Writer) error {
	if len(args) == 0 {
		yml, err := yaml.Marshal(metadata)
		if err != nil {
			return EnsureErrorResult(err)
		}
		out.Write(yml)
		return nil
	}

	value, present := metadata[args[0]]
	if !present {
		return errors.Errorf("No value for %q in cluster %s", args[0], clusterName)
	}
	fmt.Fprint(out, value)

	return nil
}
