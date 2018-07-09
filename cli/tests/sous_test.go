package tests

import (
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/yaml"
)

func TestSous(t *testing.T) {
	term := NewTerminal(t, `0.0.0`)

	// Invoke the CLI
	term.RunCommand("sous")

	t.Log(term.Stderr)
	term.Stdout.ShouldHaveNumLines(0)
	term.Stderr.ShouldHaveNumLines(47)

	term.Stderr.ShouldHaveExactLine("usage: sous <command>")
	term.Stderr.ShouldHaveLineContaining("help      get help with sous")
}

func TestSousVersion(t *testing.T) {
	term := NewTerminal(t, "1.0.0-test")

	// This prints the whole shell session if the test fails.
	defer term.PrintFailureSummary()

	term.RunCommand("sous version")
	term.Stderr.ShouldHaveNumLines(0)
	term.Stdout.ShouldHaveNumLines(1)
	term.Stdout.ShouldHaveLineContaining("sous version")
}

func TestSous_Init(t *testing.T) {
	term := NewTerminal(t, `0.0.0`)

	term.RunCommand("sous init")

	term.Stdout.ShouldHaveNumLines(0)
	term.Stderr.ShouldHaveNumLines(1)

	term.Stderr.ShouldHaveExactLine(`kind "" not defined, pick one of "scheduled", "http-service" or "on-demand"`)
}

func TestSousConfig_validConfig(t *testing.T) {
	term := NewTerminal(t, "0.0.0")

	defer term.PrintFailureSummary()

	testConfig := &config.Config{}
	if err := term.Graph.Realise(testConfig); err != nil {
		t.Fatal(err)
	}

	if err := testConfig.Validate(); err != nil {
		t.Fatalf("inconclusive; config was not valid to begin with: %s", err)
	}

	term.RunCommand("sous config")

	yamlConfig, err := yaml.Marshal(testConfig)
	if err != nil {
		t.Fatalf("inconclusive: unable to marshal config to YAML: %s", err)
	}
	term.Stdout.ShouldContain(yamlConfig)

	term.Stderr.ShouldBeEmpty()
}

func TestSousConfig_invalidConfig(t *testing.T) {
	term := NewTerminal(t, "0.0.0")

	defer term.PrintFailureSummary()

	testConfig := graph.RawConfig{}
	if err := term.Graph.Realise(&testConfig); err != nil {
		t.Fatal(err)
	}
	testConfig.Server = "not a valid URL"
	if err := testConfig.Validate(); err == nil {
		t.Fatalf("inconclusive; config was not invalid")
	}
	// Put the invalid config back in the graph.
	term.Graph.Replace(testConfig)

	term.RunCommand("sous config")

	yamlConfig, err := yaml.Marshal(testConfig.Config)
	if err != nil {
		t.Fatalf("inconclusive: unable to marshal config to YAML: %s", err)
	}
	term.Stdout.ShouldContain(yamlConfig)

	const configWarning = `WARNING: Invalid configuration: Config.Server: URL "not a valid URL" must begin with http:// or https://`
	term.Stderr.ShouldContainString(configWarning)
}
