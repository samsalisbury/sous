package graph

import (
	"io"

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
	graph := buildBaseGraph(in, out, err)
	addTestFilesystem(graph)
	graph.Add(configYAML(cfg))
	return graph
}

func addTestFilesystem(graph adder) {
	graph.Add(newTestConfigLoader)
}

func newTestConfigLoader(configYAML configYAML) *ConfigLoader {
	cl := &testConfigLoader{configYAML: configYAML}
	return &ConfigLoader{ConfigLoader: cl}
}

func (cl *testConfigLoader) Load(data interface{}, path string) error {
	err := yaml.Unmarshal([]byte(cl.configYAML), data)
	return err
}
