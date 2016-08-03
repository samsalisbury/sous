package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/firsterr"
	"github.com/samsalisbury/semv"
)

// SousDeploy is the command description for `sous init`
type SousDeploy struct {
	SourceContext *sous.SourceContext
	WD            LocalWorkDirShell
	GDM           CurrentGDM
	StateWriter   LocalStateWriter

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
	fs.StringVar(&su.flags.RepoURL, "repo-url", "",
		"the source code repo for this project (e.g. github.com/user/project)")
	fs.StringVar(&su.flags.RepoOffset, "repo-offset", "",
		"the subdir within the repo where the source code lives (empty for root)")
	fs.StringVar(&su.flags.Cluster, "cluster", "",
		"which cluster to update the config in")
	fs.StringVar(&su.flags.Version, "version", "",
		"which version to deploy to this cluster")
}

// Execute fulfills the cmdr.Executor interface.
func (su *SousDeploy) Execute(args []string) cmdr.Result {
	var repoURL, repoOffset string
	if err := firsterr.Parallel().Set(
		func(e *error) { repoURL, *e = su.resolveRepoURL() },
		func(e *error) { repoOffset, *e = su.resolveRepoOffset() },
	); err != nil {
		return EnsureErrorResult(err)
	}

	if su.flags.Cluster == "" {
		return UsageErrorf("You must a cluster using the -cluster flag.")
	}

	sl := sous.NewSourceLocation(repoURL, repoOffset)

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
			repo:    repoURL,
			offset:  repoOffset,
			cluster: clusterName,
		},
	}
	return rectify.Execute(nil)
}

func (su *SousDeploy) resolveRepoURL() (string, error) {
	repoURL := su.flags.RepoURL
	if repoURL == "" {
		repoURL = su.SourceContext.PossiblePrimaryRemoteURL
		if repoURL == "" {
			return "", fmt.Errorf("no repo URL found, please use -repo-url")
		}
		sous.Log.Info.Printf("using repo URL %q (from git remotes)", repoURL)
	}
	if !strings.HasPrefix(repoURL, "github.com/") {
		return "", fmt.Errorf("repo URL must begin with github.com/")
	}
	return repoURL, nil
}

func (su *SousDeploy) resolveRepoOffset() (string, error) {
	repoOffset := su.flags.RepoOffset
	if repoOffset == "" {
		repoOffset := su.SourceContext.OffsetDir
		sous.Log.Info.Printf("using current workdir repo offset: %q", repoOffset)
	}
	if len(repoOffset) != 0 && repoOffset[:1] == "/" {
		return "", fmt.Errorf("repo offset cannot begin with /, it is relative")
	}
	return repoOffset, nil
}
