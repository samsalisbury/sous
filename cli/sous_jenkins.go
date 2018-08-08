package cli

import (
	"flag"

	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/cmdr"
)

// SousJenkins is the command description for `sous deploy`.
type SousJenkins struct {
	SousGraph *graph.SousGraph
	opts      graph.DeployActionOpts
}

func init() { TopLevelCommands["jenkins"] = &SousJenkins{} }

const sousJenkinsHelp = `generates JenkinsFile for pipeline build and deploy

usage: sous jenkins (options)

sous jenkins will query manifest metadata and generate a JenkinsFile
`

// Help returns the help string for this command.
func (sd *SousJenkins) Help() string { return sousJenkinsHelp }

// AddFlags adds the flags for sous init.
func (sd *SousJenkins) AddFlags(fs *flag.FlagSet) {
	//Not sure if I need?
	MustAddFlags(fs, &sd.opts.DFF, NewDeployFilterFlagsHelp)
}

// Execute fulfills the cmdr.Executor interface.
func (sd *SousJenkins) Execute(args []string) cmdr.Result {
	jenkins, err := sd.SousGraph.GetJenkins(sd.opts)

	if err != nil {
		return cmdr.EnsureErrorResult(err)
	}

	err = jenkins.Do()

	if err != nil {
		return EnsureErrorResult(err)
	}

	return cmdr.Success("Done.")
}
