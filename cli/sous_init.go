package cli

import (
	"flag"
	"fmt"
	"strings"

	"github.com/opentable/sous/ext/otpl"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/opentable/sous/util/firsterr"
)

// SousInit is the command description for `sous init`
type SousInit struct {
	SourceContextFunc
	WD          LocalWorkDirShell
	GDM         CurrentGDM
	StateWriter LocalStateWriter
	flags       struct {
		RepoURL, RepoOffset             string
		UseOTPLDeploy, IgnoreOTPLDeploy bool
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

// AddFlags adds the flags for sous init.
func (si *SousInit) AddFlags(fs *flag.FlagSet) {
	fs.BoolVar(&si.flags.UseOTPLDeploy, "use-otpl-deploy", false,
		"if specified, copies OpenTable-specific otpl-deploy configuration to the manifest")
	fs.BoolVar(&si.flags.IgnoreOTPLDeploy, "ignore-otpl-deploy", false,
		"if specified, ignores OpenTable-specific otpl-deploy configuration")
}

// Execute fulfills the cmdr.Executor interface
func (si *SousInit) Execute(args []string) cmdr.Result {
	ctx, err := si.SourceContextFunc()
	if err != nil {
		return EnsureErrorResult(err)
	}
	var repoURL, repoOffset string
	if err := firsterr.Parallel().Set(
		func(e *error) { repoURL, *e = si.resolveRepoURL(ctx) },
		func(e *error) { repoOffset, *e = si.resolveRepoOffset(ctx) },
	); err != nil {
		return EnsureErrorResult(err)
	}

	sourceLocation := sous.NewSourceLocation(repoURL, repoOffset)

	existingManifest := si.GDM.GetManifest(sourceLocation)
	if existingManifest != nil {
		return UsageErrorf("init failed: manifest %q already exists", sourceLocation)
	}

	var deploySpecs, otplDeploySpecs sous.DeploySpecs
	if !si.flags.IgnoreOTPLDeploy {
		otplParser := otpl.NewDeploySpecParser()
		otplDeploySpecs = otplParser.GetDeploySpecs(si.WD.Sh)
	}
	if !si.flags.UseOTPLDeploy && !si.flags.IgnoreOTPLDeploy && len(otplDeploySpecs) != 0 {
		return UsageErrorf("otpl-deploy detected in config/, please specify either -use-otpl-deploy, or -ignore-otpl-deploy to proceed")
	}
	if si.flags.UseOTPLDeploy {
		if len(otplDeploySpecs) == 0 {
			return UsageErrorf("you specified -use-otpl-deploy, but no valid deployments were found in config/")
		}
		deploySpecs = otplDeploySpecs
	}
	if len(deploySpecs) == 0 {
		deploySpecs = defaultDeploySpecs()
	}

	m := &sous.Manifest{
		Source: sous.SourceLocation{
			RepoURL:    sous.RepoURL(repoURL),
			RepoOffset: sous.RepoOffset(repoOffset),
		},
		Deployments: deploySpecs,
	}

	if err := si.GDM.AddManifest(m); err != nil {
		return EnsureErrorResult(err)
	}

	if err := si.StateWriter.WriteState(si.GDM.State); err != nil {
		return EnsureErrorResult(err)
	}

	return SuccessYAML(m)
}

func defaultDeploySpecs() sous.DeploySpecs {
	return sous.DeploySpecs{
		"Global": {
			DeployConfig: sous.DeployConfig{
				Resources:    sous.Resources{},
				Env:          map[string]string{},
				NumInstances: 3,
			},
		},
	}
}

func (si *SousInit) resolveRepoURL(ctx *sous.SourceContext) (string, error) {
	repoURL := si.flags.RepoURL
	if repoURL == "" {
		repoURL = ctx.PrimaryRemoteURL
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

func (si *SousInit) resolveRepoOffset(ctx *sous.SourceContext) (string, error) {
	repoOffset := si.flags.RepoOffset
	if repoOffset == "" {
		repoOffset := ctx.OffsetDir
		sous.Log.Info.Printf("using current workdir repo offset: %q", repoOffset)
	}
	if len(repoOffset) != 0 && repoOffset[:1] == "/" {
		return "", fmt.Errorf("repo offset cannot begin with /, it is relative")
	}
	return repoOffset, nil
}
