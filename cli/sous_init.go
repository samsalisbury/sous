package cli

import (
	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/cmdr"
	"github.com/samsalisbury/semv"
)

type SousInit struct {
	User          LocalUser
	Config        LocalSousConfig
	SourceContext *sous.SourceContext
}

func init() { TopLevelCommands["init"] = &SousInit{} }

const sousInitHelp = `
initialise a new sous project

usage: sous init

Sous init uses contextual information from your current source code tree and
repository to generate a basic configuration for that project. You will need to
flesh out some additional details.
`

func (si *SousInit) Help() string { return sousInitHelp }

func (si *SousInit) Execute(args []string) cmdr.Result {
	c := si.SourceContext
	v, err := semv.Parse(c.NearestTagName + "+" + c.Revision)
	if err != nil {
		v = semv.MustParse("0.0.0-unversioned+" + c.Revision)
	}
	m := sous.Manifest{
		Source: sous.SourceLocation{
			sous.RepoURL(c.PossiblePrimaryRemoteURL),
			sous.RepoOffset(c.OffsetDir),
		},
		Deployments: map[string]sous.PartialDeploySpec{
			"Global": sous.PartialDeploySpec{
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
