package cli

import (
	"flag"
	"fmt"
	"io"
	"log"

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
	*sous.State
	graph.CurrentGDM
	graph.OutWriter
}

func init() { MetadataSubcommands["get"] = &SousMetadataGet{} }

const sousMetadataGetHelp = `
query deployment metadata
`

func (*SousMetadataGet) Help() string { return sousMetadataHelp }

func (smg *SousMetadataGet) AddFlags(fs *flag.FlagSet) {
	MustAddFlags(fs, &smg.DeployFilterFlags, MetadataFilterFlagsHelp)
}

func (smg *SousMetadataGet) RegisterOn(psy Addable) {
	psy.Add(&smg.DeployFilterFlags)
}

func (smg *SousMetadataGet) Execute(args []string) cmdr.Result {
	if smg.DeployFilterFlags.Repo == "" {
		return EnsureErrorResult(errors.Errorf("-repo is required"))
	}
	filtered := smg.CurrentGDM.Clone().Filter(smg.ResolveFilter.FilterDeployment)
	log.Printf("%#+v", filtered)
	if smg.ResolveFilter.Cluster != "" {
		dep, err := filtered.Only()
		if err != nil {
			return EnsureErrorResult(err)
		}
		if dep == nil {
			return EnsureErrorResult(errors.Errorf("No manifest deploy for %v", smg.DeployFilterFlags))
		}
		log.Printf("%#v", dep)
		return outputMetadata(dep.Metadata, smg.ResolveFilter.Cluster, args, smg.OutWriter)
	}

	manis, err := filtered.Manifests(smg.State.Defs)
	if err != nil {
		return EnsureErrorResult(err)
	}
	mani, err := manis.Only()
	if err != nil {
		return EnsureErrorResult(err)
	}
	if mani == nil {
		return EnsureErrorResult(errors.Errorf("No manifest for %v", smg.DeployFilterFlags))
	}

	var metadata sous.Metadata
	global, hasGlobal := mani.Deployments["Global"]
	if hasGlobal {
		metadata = global.Metadata
	}

	return outputMetadata(metadata, "Global", args, smg.OutWriter)
}

func outputMetadata(metadata sous.Metadata, clusterName string, args []string, out io.Writer) cmdr.Result {
	log.Printf("%v %v %v", clusterName, args, metadata)
	if len(args) == 0 {
		yml, err := yaml.Marshal(metadata)
		if err != nil {
			return EnsureErrorResult(err)
		}
		out.Write(yml)
		return Success()
	}

	value, present := metadata[args[0]]
	if !present {
		return EnsureErrorResult(errors.Errorf("No value for %q in cluster %s", args[0], clusterName))
	}
	fmt.Fprint(out, value)

	return Success()
}
