package cli

import (
	"flag"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/samsalisbury/semv"
)

// SousDeploy is the command description for `sous deploy`
type SousDeploy struct {
	SourceContextFunc
	WD          LocalWorkDirShell
	GDM         CurrentGDM
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
	ctx, err := su.SourceContextFunc()
	if err != nil {
		return EnsureErrorResult(err)
	}

	if su.flags.Cluster == "" {
		return UsageErrorf("You must a select a cluster using the -cluster flag.")
	}

	sl := ctx.SourceLocation()

	m := su.GDM.GetManifest(sl)
	if m == nil {
		return UsageErrorf("update failed: manifest %q does not exist", sl)
	}

	clusterName := su.flags.Cluster
	cluster, ok := m.Deployments[clusterName]
	if !ok {
		return UsageErrorf("Cluster %q does not exist.")
	}

	versionStr := su.flags.Version
	newVersion, err := semv.Parse(versionStr)
	if err != nil {
		return UsageErrorf("version not valid: %s", err)
	}
	cluster.Version = newVersion
	m.Deployments[clusterName] = cluster

	if err := su.StateWriter.WriteState(su.GDM.State); err != nil {
		return EnsureErrorResult(err)
	}

	rectify := SousRectify{
		Config:       su.Config,
		DockerClient: su.DockerClient,
		Deployer:     su.Deployer,
		Registry:     su.Registry,
		GDM:          su.GDM,
		flags: rectifyFlags{
			repo:    string(sl.RepoURL),
			offset:  string(sl.RepoOffset),
			cluster: clusterName,
		},
	}
	return rectify.Execute(nil)
}
