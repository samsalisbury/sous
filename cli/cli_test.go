package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/graph"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/cmdr"
	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func prepareCommand(t *testing.T, cl []string) (*CLI, *cmdr.PreparedExecution, fmt.Stringer, fmt.Stringer) {
	require := require.New(t)

	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	s := &Sous{Version: semv.MustParse(`1.2.3`)}
	c, err := NewSousCLI(s, stdin, stdout, stderr)
	require.NoError(err)

	c.baseGraph = graph.BuildTestGraph(stdin, stdout, stderr)

	exe, err := c.Prepare(cl)
	require.NoError(err)

	return c, exe, stdout, stderr
}

func justCommand(t *testing.T, cl []string) *cmdr.PreparedExecution {
	_, exe, _, _ := prepareCommand(t, cl)
	return exe
}

/*
usage: sous config Invoking sous config with no arguments lists all configuration key/value pairs.
If you pass just a single argument (a key) sous config will output just the
value of that key. You can set a key by providing both a key and a value.

usage: sous config [<key> [value]]

*/
func TestInvokeConfig(t *testing.T) {
	assert := assert.New(t)

	exe := justCommand(t, []string{`sous`, `config`})
	assert.NotNil(exe)
	assert.Len(exe.Args, 0)

	exe = justCommand(t, []string{`sous`, `config`, `x`})
	assert.NotNil(exe)
	assert.Len(exe.Args, 1)

	exe = justCommand(t, []string{`sous`, `config`, `x`, `7`})
	assert.NotNil(exe)
	assert.Len(exe.Args, 2)

}

func TestInvokeUpdate(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	exe := justCommand(t, []string{`sous`, `update`})
	assert.NotNil(exe)
	assert.Len(exe.Args, 0)
	update, good := exe.Cmd.(*SousUpdate)
	require.True(good)
	assert.NotNil(update.StateManager)
	assert.NotNil(update.StateManager.StateManager)
}

func TestInvokeDeploy(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	exe := justCommand(t, []string{`sous`, `deploy`, `-cluster`, `ci-sf`, `-tag`, `1.2.3`})
	assert.NotNil(exe)
	assert.Len(exe.Args, 0)
	deploy, good := exe.Cmd.(*SousDeploy)
	require.True(good)
	assert.Equal(deploy.DeployFilterFlags.Cluster, `ci-sf`)
	assert.Equal(deploy.DeployFilterFlags.Tag, `1.2.3`)
}

func TestInvokeDeploy_RepoFlag(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	exe := justCommand(t, []string{`sous`, `deploy`, `-cluster`, `ci-sf`, `-tag`, `1.2.3`})
	sd, ok := exe.Cmd.(*SousDeploy)
	require.True(ok)
	su := &SousUpdate{}
	sps := &SousPlumbingStatus{}
	sous.Log.Debug.Printf("Plumbing Update...")
	require.NoError(sd.CLI.Plumb(su, sps))

	assert.NotEqual(su.ResolveFilter.Repo, "")
	assert.NotEqual(su.ResolveFilter.Repo, "github.com/example/project")
	assert.NotEqual(su.Manifest.ID().Source.Repo, "")
	assert.NotEqual(su.Manifest.ID().Source.Repo, "github.com/example/project")

	sous.Log.Debug.Printf("%#v", sps)
	if assert.NotNil(sps.StatusPoller) {
		assert.NotEqual(sps.StatusPoller.Repo, "")
		assert.Equal(sps.StatusPoller.Repo, su.Manifest.ID().Source.Repo)
	}

	exe = justCommand(t, []string{`sous`, `deploy`, `-cluster`, `ci-sf`, `-repo`, `github.com/example/project`, `-tag`, `1.2.3`})
	sd, ok = exe.Cmd.(*SousDeploy)
	require.True(ok)
	su = &SousUpdate{}
	sd.CLI.Plumb(su, sps)
	assert.Equal(su.ResolveFilter.Repo, "github.com/example/project")

	assert.Equal(su.Manifest.ID().Source.Repo, sps.StatusPoller.Repo)
	assert.Equal(su.Manifest.ID().Source.Repo, "github.com/example/project")

}

/*
usage: sous context

context prints out sous's view of your current context
*/
func TestInvokeContext(t *testing.T) {
	assert := assert.New(t)

	exe := justCommand(t, []string{`sous`, `context`})
	assert.NotNil(exe)
	assert.Len(exe.Args, 0)
}

/*
usage: sous init Sous init uses contextual information from your current source code tree and
repository to generate a basic configuration for that project. You will need to
flesh out some additional details.

usage: sous init

options:
  -ignore-otpl-deploy
    	if specified, ignores OpenTable-specific otpl-deploy configuration
  -use-otpl-deploy
    	if specified, copies OpenTable-specific otpl-deploy configuration to the manifest
*/

func TestInvokeInit(t *testing.T) {
	assert := assert.New(t)

	exe := justCommand(t, []string{`sous`, `init`})
	init := exe.Cmd.(*SousInit)
	assert.NotNil(init)
	assert.False(init.Flags.IgnoreOTPLDeploy)
	assert.False(init.Flags.IgnoreOTPLDeploy)
}

/*
usage: sous query [path]

build builds the project in your current directory by default. If you pass it a
path, it will instead build the project at that path.

subcommands:
  ads  build your project
  gdm  Loads the current deployment configuration and prints it out

options:
usage: sous query ads [path]

build builds the project in your current directory by default. If you pass it a
path, it will instead build the project at that path.

usage: sous query gdm

This should resemble the manifest that was used to establish the intended state of deployment.
*/

func TestInvokeQuery(t *testing.T) {
	assert := assert.New(t)

	exe := justCommand(t, []string{`sous`, `query`})
	assert.NotNil(exe)

	exe = justCommand(t, []string{`sous`, `query`, `ads`})
	assert.NotNil(exe)

	exe = justCommand(t, []string{`sous`, `query`, `gdm`})
	assert.NotNil(exe)
}

func TestInvokeQueryArtifacts(t *testing.T) {
	assert := assert.New(t)

	exe := justCommand(t, []string{`sous`, `query`, `artifacts`})
	assert.NotNil(exe)
}

func TestInvokeMetadataGet(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	exe := justCommand(t, []string{`sous`, `metadata`, `get`, `-repo`, `github.com/opentable/sous`})
	assert.NotNil(exe)
	metaGet, good := exe.Cmd.(*SousMetadataGet)
	require.True(good)
	assert.NotNil(metaGet.State)
}

func TestInvokeMetadataSet(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	exe := justCommand(t, []string{`sous`, `metadata`, `set`, `-repo`, `github.com/opentable/sous`, `BuildBranch`, `master`})
	assert.NotNil(exe)
	metaSet, good := exe.Cmd.(*SousMetadataSet)
	require.True(good)
	assert.NotNil(metaSet.State)
}

func TestInvokeManifestGet(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	exe := justCommand(t, []string{`sous`, `manifest`, `get`, `-repo`, `github.com/opentable/sous`})
	assert.NotNil(exe)
	maniGet, good := exe.Cmd.(*SousManifestGet)
	require.True(good)
	assert.NotNil(maniGet.State)
}

func TestInvokeManifestSet(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)
	exe := justCommand(t, []string{`sous`, `manifest`, `set`, `-repo`, `github.com/opentable/sous`})
	assert.NotNil(exe)
	maniSet, good := exe.Cmd.(*SousManifestSet)
	require.True(good)
	assert.NotNil(maniSet.StateWriter)
}

func TestInvokeServer(t *testing.T) {
	exe := justCommand(t, []string{`sous`, `server`})
	assert.NotNil(t, exe)

	exe = justCommand(t, []string{`sous`, `server`, `-cluster`, `test`})
	assert.NotNil(t, exe)
	server, good := exe.Cmd.(*SousServer)
	require.True(t, good)
	assert.NotNil(t, server.SousGraph)
	assert.True(t, server.AutoResolver.ResolveFilter.Offset.All, "server.AutoResolver.ResolveFilter.Offset.All")
	assert.True(t, server.AutoResolver.ResolveFilter.Flavor.All, "server.AutoResolver.ResolveFilter.Flavor.All")
}

/*
usage: sous version

prints the current version of sous. Please include the output from this
command with any bug reports sent to https://github.com/opentable/sous/issues
*/

func TestInvokeVersion(t *testing.T) {
	assert := assert.New(t)

	exe := justCommand(t, []string{`sous`, `version`})
	assert.NotNil(exe)
}

func TestInvokeHarvest(t *testing.T) {
	assert := assert.New(t)

	exe := justCommand(t, []string{`sous`, `harvest`, `sms-continual-test`})
	assert.NotNil(exe)
	assert.Len(exe.Args, 1)
}

/*
usage: sous <command>

sous is a tool to help speed up the build/test/deploy cycle at your organisation

subcommands:
  build    build your project
  config   view and edit sous configuration
  context  show the current build context
  deploy   initialise a new sous project
  help     get help with sous
  init     initialise a new sous project
  query    build your project
  rectify  force Sous to make the deployment match the contents of the local state directory
  version  print the version of sous

options:
  -d	debug: output detailed logs of internal operations
  -q	quiet: output only essential error messages
  -s	silent: silence all non-essential output
  -v	loud: output extra info, including all shell commands
*/
func TestInvokeBareSous(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	c, exe, _, _ := prepareCommand(t, []string{`sous`})
	assert.Len(exe.Args, 0)

	var r cmdr.Result
	c.InvokeWithoutPrinting([]string{"sous", "help"})
	require.NotPanics(func() { r = c.InvokeWithoutPrinting([]string{"sous", "help"}) })
	assert.IsType(cmdr.SuccessResult{}, r)
}

/*
usage: sous rectify Several predicates are available to constrain the action of the rectification.
-repo, -offset and -cluster limit the rectification appropriately. When used
together, the result is the intersection of their images - that is, the
conditions are "anded." By implication, each can only be used once.
NOTE: the successful use of these predicates requires all-team coordination.
Use with great care.

usage: sous rectify

options:
  -all
    	all deployments should be considered
  -cluster string
    	target deployment cluster
  -dry-run string
    	prevent rectify from actually changing things - values are none,scheduler,registry,both (default "none")
  -offset string
    	source code relative repository offset
  -repo string
    	source code repository location
*/

func TestInvokeWithUnknownFlags(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	stdin := &bytes.Buffer{}
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}

	s := &Sous{Version: semv.MustParse(`1.2.3`)}
	c, err := NewSousCLI(s, stdin, stdout, stderr)
	require.NoError(err)

	c.Invoke([]string{`sous`, `-cobblers`})
	assert.Regexp(`flag provided but not defined`, stderr.String())
}

func TestInvokeRectifyWithoutFilterFlags(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	_, exe, _, _ := prepareCommand(t, []string{`sous`, `rectify`})
	assert.Len(exe.Args, 0)
	require.IsType(&SousRectify{}, exe.Cmd)

	rect := exe.Cmd.(*SousRectify)

	assert.NotNil(rect.Config)
	assert.NotNil(rect.GDM)
	require.NotNil(rect.SourceFlags)
	assert.Equal(rect.SourceFlags.All, false)
	require.NotNil(rect.Resolver.ResolveFilter)

	sous.Log.Vomit.Printf("%#v", rect.Resolver.ResolveFilter)
	assert.Equal(rect.Resolver.ResolveFilter.All(), true)
}

func TestInvokeRectifyWithDebugFlags(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	_, exe, _, stderr := prepareCommand(t, []string{`sous`, `rectify`, `-d`, `-v`, `-all`})
	assert.Len(exe.Args, 0)
	require.IsType(&SousRectify{}, exe.Cmd)

	rect := exe.Cmd.(*SousRectify)

	assert.NotNil(rect.Config)
	assert.NotNil(rect.GDM)
	require.NotNil(rect.SourceFlags)
	assert.Equal(rect.SourceFlags.All, true)
	assert.Regexp(`Verbose debugging`, stderr.String())
	assert.Regexp(`Regular debugging`, stderr.String())
}

func TestInvokeRectifyDryruns(t *testing.T) {
	assert := assert.New(t)
	require := require.New(t)

	testDryRun := func(which string) (sous.Deployer, sous.Registry) {
		exe := justCommand(t, []string{`sous`, `rectify`, `-dry-run`, which, `-repo`, `github.com/somewhere`})
		assert.Len(exe.Args, 0)
		require.IsType(&SousRectify{}, exe.Cmd)
		rect := exe.Cmd.(*SousRectify)
		// currently no easy way to tell if the deploy client is live or dummy
		return nil, rect.Resolver.Registry
	}

	_, r := testDryRun("both")
	assert.IsType(&sous.DummyRegistry{}, r)

	_, r = testDryRun("none")
	assert.IsType(&docker.NameCache{}, r)

	_, r = testDryRun("scheduler")
	assert.IsType(&docker.NameCache{}, r)

	_, r = testDryRun("registry")
	assert.IsType(&sous.DummyRegistry{}, r)
}

/*
usage: sous build [path]

build builds the project in your current directory by default. If you pass it a
path, it will instead build the project at that path.

options:
  -offset string
    	source code relative repository offset
  -repo string
    	source code repository location
  -revision string
    	source code revision ID
  -strict
    	require that the build be pristine
  -tag string
    	source code revision tag

*/
func TestInvokeBuildWithRepoSelector(t *testing.T) {
	assert := assert.New(t)

	_, exe, _, _ := prepareCommand(t, []string{`sous`, `build`, `-repo`, `github.com/opentable/sous`})
	assert.Len(exe.Args, 0)

	build := exe.Cmd.(*SousBuild)

	assert.NotNil(build.Labeller)
	assert.NotNil(build.Registrar)
	assert.Equal(build.DeployFilterFlags.Repo, `github.com/opentable/sous`)
}

func TestSousDeployments(t *testing.T) {
	assert := assert.New(t)

	_, exe, _, _ := prepareCommand(t, []string{`sous`, `deployments`, `-repo`, `github.com/opentable/sous`})
	assert.Len(exe.Args, 0)

	c := exe.Cmd.(*SousDeployments)

	assert.NotNil(c.State)
	assert.NotNil(c.GDM)
	assert.NotNil(c.Deployer)
	assert.NotNil(c.Registry)
	assert.Equal(c.TargetManifestID.Source.Repo, `github.com/opentable/sous`)
}
