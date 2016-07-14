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

// SousInit is the command description for `sous init`
type SousInit struct {
	Sous          *Sous
	User          LocalUser
	Config        LocalSousConfig
	SourceContext *sous.SourceContext
	flags         struct {
		RepoURL, RepoOffset string
	}
}

func init() { TopLevelCommands["init"] = &SousInit{} }

const sousInitHelp = `
initialise a new sous project

usage: sous init

Sous init uses contextual information from your current source code tree and
repository to generate a basic configuration for that project. You will need to
flesh out some additional details.
`

// Help returns the help string for this command
func (si *SousInit) Help() string { return sousInitHelp }

func (si *SousInit) AddFlags(fs *flag.FlagSet) {
	fs.StringVar(&si.flags.RepoURL, "repo-url", "",
		"the source code repo for this project (e.g. github.com/user/project)")
	fs.StringVar(&si.flags.RepoOffset, "repo-offset", "",
		"the subdir within the repo where the source code lives (empty for root)")
}

// Execute fulfills the cmdr.Executor interface
func (si *SousInit) Execute(args []string) cmdr.Result {
	c := si.SourceContext
	v, err := semv.Parse(c.NearestTagName + "+" + c.Revision)
	if err != nil {
		v = semv.MustParse("0.0.0-unversioned+" + c.Revision)
	}

	var repoURL, repoOffset string
	if err := firsterr.Parallel().Set(
		func(e *error) { repoURL, *e = si.ResolveRepoURL() },
		func(e *error) { repoOffset, *e = si.ResolveRepoOffset() },
	); err != nil {
		return EnsureErrorResult(err)
	}

	m := sous.Manifest{
		Source: sous.SourceLocation{
			RepoURL:    sous.RepoURL(repoURL),
			RepoOffset: sous.RepoOffset(repoOffset),
		},
		Deployments: sous.DeploySpecs{
			"Global": {
				DeployConfig: sous.DeployConfig{
					Resources:    sous.Resources{},
					Env:          map[string]string{},
					NumInstances: 3,
				},
				Version: v,
			},
		},
	}
	return SuccessYAML(m)
}

func (si *SousInit) ResolveRepoURL() (string, error) {
	repoURL := si.flags.RepoURL
	if repoURL == "" {
		repoURL = si.SourceContext.PossiblePrimaryRemoteURL
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

func (si *SousInit) ResolveRepoOffset() (string, error) {
	repoOffset := si.flags.RepoOffset
	if repoOffset == "" {
		repoOffset := si.SourceContext.OffsetDir
		sous.Log.Info.Printf("using current workdir repo offset: %q", repoOffset)
	}
	if repoOffset[:1] == "/" {
		return "", fmt.Errorf("repo offset cannot begin with /, it is relative")
	}
	return repoOffset, nil
}
