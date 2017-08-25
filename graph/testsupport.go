package graph

import (
	"io"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/restful"
	"github.com/opentable/sous/util/yaml"
)

type (
	configYAML string

	testConfigLoader struct {
		configYAML
	}
)

const defaultConfig = ""

// BuildTestGraph builds a standard graph suitable for testing
func BuildTestGraph(in io.Reader, out, err io.Writer) *SousGraph {
	return TestGraphWithConfig(in, out, err, defaultConfig)
}

// TestGraphWithConfig accepts a custom Sous config string
func TestGraphWithConfig(in io.Reader, out, err io.Writer, cfg string) *SousGraph {
	graph := BuildBaseGraph(in, out, err)
	AddTestConfig(graph, cfg)
	graph.Add(sous.User{Name: "Test User", Email: "testuser@example.com"})
	AddState(graph)
	addTestNetwork(graph)
	graph.Add(newServerStateManager)
	return graph
}

// AddTestConfig adds configuration objects to the DI.
func AddTestConfig(graph adder, cfg string) {
	graph.Add(configYAML(cfg))
	graph.Add(NewTestConfigLoader)
}

func addTestNetwork(graph adder) {
	graph.Add(newDummyHTTPClient)
	graph.Add(newDummyDockerClient)
	graph.Add(newServerHandler)
}

func newDummyHTTPClient() HTTPClient {
	return HTTPClient{HTTPClient: &restful.DummyHTTPClient{}}
}

func newDummyDockerClient() LocalDockerClient {
	return LocalDockerClient{Client: docker_registry.NewDummyClient()}
}

// NewTestConfigLoader produces a faked ConfigLoader
func NewTestConfigLoader(configYAML configYAML) *ConfigLoader {
	cl := &testConfigLoader{configYAML: configYAML}
	return &ConfigLoader{ConfigLoader: cl}
}

func (cl *testConfigLoader) Load(data interface{}, path string) error {
	err := yaml.Unmarshal([]byte(cl.configYAML), data)
	return err
}
