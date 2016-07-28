package cli

import (
	"flag"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/samsalisbury/semv"
)

// SousDeploy is the command description for `sous deploy`
type SousDeploy struct {
	*sous.SourceContext
	WD          LocalWorkDirShell
	GDM         *sous.State
	StateWriter LocalStateWriter

	// Rectify fields
	Config       LocalSousConfig
	DockerClient LocalDockerClient
	Deployer     sous.Deployer
	Registry     sous.Registry

	flags struct {
		RepoURL, RepoOffset, Cluster, Version string
	}
}

func init() { TopLevelCommands["deploy"] = &SousDeploy{} }

const sousUpdateHelp = `
deploy a new version

usage: sous deploy -cluster <name> -version <semver>
`

// Help returns the help string for this command
func (su *SousDeploy) Help() string { return sousInitHelp }

// AddFlags adds the flags for sous init.
func (su *SousDeploy) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&su.flags.Cluster, "cluster", "",
		"which cluster to update the config in")
	fs.StringVar(&su.flags.Version, "version", "",
		"which version to deploy to this cluster")
}

// Execute fulfills the cmdr.Executor interface.
func (su *SousDeploy) Execute(args []string) cmdr.Result {
	if su.flags.Cluster == "" {
		return UsageErrorf("You must a select a cluster using the -cluster flag.")
	}

	sl := su.SourceContext.SourceLocation()

	clusterName := su.flags.Cluster
	deployments, err := su.GDM.Deployments()
	if err != nil {
		return EnsureErrorResult(err)
	}
	id := sous.DeployID{Source: sl, Cluster: clusterName}
	deployment, ok := deployments.Get(id)
	if !ok {
		return UsageErrorf("Cluster %q does not exist.")
	}

	versionStr := su.flags.Version
	newVersion, err := semv.Parse(versionStr)
	if err != nil {
		return UsageErrorf("version not valid: %s", err)
	}
	deployment.SourceID.Version = newVersion
	deployments.Set(id, deployment)

	if err := su.StateWriter.WriteState(su.GDM); err != nil {
		return EnsureErrorResult(err)
	}

	rectify := SousRectify{
		Config:       su.Config,
		DockerClient: su.DockerClient,
		Deployer:     su.Deployer,
		Registry:     su.Registry,
		//GDM:          su.GDM,
		//flags: rectifyFlags{
		//	repo:    string(sl.RepoURL),
		//	offset:  string(sl.RepoOffset),
		//	cluster: clusterName,
		//},
	}
	return rectify.Execute(nil)
}
