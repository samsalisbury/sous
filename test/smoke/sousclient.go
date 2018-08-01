package smoke

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/ext/docker"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/filemap"
	"github.com/opentable/sous/util/yaml"
)

type sousClient struct {
	Bin
	// Config is set after calling Configure()
	Config        config.Config
	FixtureConfig fixtureConfig
}

func makeClient(t *testing.T, f fixtureConfig, sousBin string) *sousClient {
	clientName := "client1"
	baseDir := path.Join(f.BaseDir, clientName)
	c := &sousClient{
		Bin:           NewBin(t, sousBin, clientName, baseDir, f.Finished),
		FixtureConfig: f,
	}

	c.Bin.Env["SOUS_CONFIG_DIR"] = c.Bin.ConfigDir
	c.Bin.Env["SOUS_BUILD_NOPULL"] = "YES"
	addGitEnvVars(c.Bin.Env)

	c.Bin.MassageArgs = c.insertClusterSuffix

	return c
}

func (c *sousClient) Configure(server, dockerReg, userEmail string) error {
	user := strings.Split(userEmail, "@")
	conf := config.Config{
		Server: server,
		Docker: docker.Config{
			RegistryHost: dockerReg,
		},
		User: sous.User{
			Name:  user[0],
			Email: userEmail,
		},
	}
	conf.PollIntervalForClient = 600

	clientDebug := os.Getenv("SOUS_CLIENT_DEBUG") == "true"

	if clientDebug {
		conf.Logging.Basic.Level = "ExtraDebug1"
		conf.Logging.Basic.DisableConsole = false
		conf.Logging.Basic.ExtraConsole = true
	}

	c.Config = conf

	configYAML, err := yaml.Marshal(conf)
	if err != nil {
		return err
	}

	return c.Bin.Configure(filemap.FileMap{
		"config.yaml": string(configYAML),
	})
}

func (c *sousClient) insertClusterSuffix(args []string) []string {
	for i, s := range args {
		if s == "-cluster" && len(args) > i+1 {
			args[i+1] = c.FixtureConfig.IsolatedClusterName(args[i+1])
		}
		if s == "-tag" && len(args) > i+1 {
			args[i+1] = c.FixtureConfig.IsolatedVersionTag(args[i+1])
		}
	}
	return args
}

// TransformManifest applies each of transforms in order to the retrieved
// manifest, then calls 'sous manifest set' to apply them. Any failure is fatal.
func (c *sousClient) TransformManifest(t *testing.T, flags *sousFlags, transforms ...ManifestTransform) {
	t.Helper()
	flags = flags.ManifestIDFlags()
	manifest := c.MustRun(t, "manifest get", flags.ManifestIDFlags())
	var m sous.Manifest
	if err := yaml.Unmarshal([]byte(manifest), &m); err != nil {
		t.Fatalf("manifest get returned invalid YAML: %s\nInvalid YAML was:\n%s", err, manifest)
	}
	for _, f := range transforms {
		m = f(m)
	}
	manifestBytes, err := yaml.Marshal(m)
	if err != nil {
		t.Fatalf("failed to marshal updated manifest: %s\nInvalid manifest was:\n% #v", err, m)
	}
	// TODO SS: remove below invocation, make a top-level RunWithStdin or something.
	i := c.newInvocation("manifest set", flags)
	manifestSetCmd, err := c.configureCommand(i)
	if err != nil {
		t.Fatal(err)
	}
	defer manifestSetCmd.Cancel()
	manifestSetCmd.Cmd.Stdin = ioutil.NopCloser(bytes.NewReader(manifestBytes))
	if err := manifestSetCmd.runWithTimeout(3 * time.Minute); err != nil {
		t.Fatalf("manifest set failed: %s; output:\n%s", err, manifestSetCmd.executed.Combined)
	}
}

func (c *sousClient) setSingularityRequestID(t *testing.T, clusterName, singReqID string) ManifestTransform {
	return func(m sous.Manifest) sous.Manifest {
		clusterName := c.FixtureConfig.IsolatedClusterName(clusterName)
		d, ok := m.Deployments[clusterName]
		if !ok {
			t.Fatalf("no deployment for %q", clusterName)
		}
		c := d.DeployConfig
		c.SingularityRequestID = singReqID
		d.DeployConfig = c
		m.Deployments[clusterName] = d
		m.Deployments = setMemAndCPUForAll(m.Deployments)
		return m
	}
}
