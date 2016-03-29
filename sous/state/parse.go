package state

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/opentable/sous/util/yaml"
)

func Parse(configDir string) (*State, error) {
	configFile := fmt.Sprintf("%s/config.yaml", configDir)
	var state State
	if err := parseYAMLFile(configFile, &state); err != nil {
		return nil, err
	}
	dcs, err := parseDatacentres(filepath.Join(configDir, "datacentres"))
	if err != nil {
		return nil, err
	}
	state.Datacentres = dcs

	manifestsDir := filepath.Join(configDir, "manifests")
	manifests, err := parseManifests(manifestsDir)
	if err != nil {
		return nil, err
	}
	state.Manifests = manifests

	contractsDir := filepath.Join(configDir, "contracts")
	contracts, err := ParseContracts(contractsDir)
	if err != nil {
		return nil, err
	}
	state.Contracts = contracts

	buildpacksDir := filepath.Join(configDir, "buildpacks")
	buildpacks, err := ParseBuildpacks(buildpacksDir)
	if err != nil {
		return nil, err
	}
	// TODO: Have base images defined by the buildpack itself, this is
	// a quick patch to try out the new buildpacks.
	for i := range buildpacks {
		switch buildpacks[i].Name {
		default:
			return nil, fmt.Errorf("Buildpack %s not recognised.", buildpacks[i].Name)
		case "golang":
			buildpacks[i].StackVersions = *state.Packs.Go.AvailableVersions
			buildpacks[i].DefaultStackVersion = state.Packs.Go.DefaultGoVersion
		case "nodejs":
			buildpacks[i].StackVersions = *state.Packs.NodeJS.AvailableVersions
			buildpacks[i].DefaultStackVersion = state.Packs.NodeJS.DefaultNodeVersion
		case "maven":
			// TODO: Get rid of this so maven is supported properly
			// by its own definition.
			continue
		}
	}
	state.Buildpacks = buildpacks

	return &state, nil
}

func ParseContracts(contractsDir string) (Contracts, error) {
	contracts := Contracts{}
	serversDir := filepath.Join(contractsDir, "servers")
	servers, err := parseServers(serversDir)
	if err != nil {
		return nil, err
	}
	testsDir := filepath.Join(contractsDir, "tests")
	tests, err := parseTests(testsDir)
	// Now parse the contracts themselves, adding servers and tests
	err = walkYAMLDir(contractsDir, func(path string) error {
		var c Contract
		if err := parseYAMLFile(path, &c); err != nil {
			return err
		}
		c.Filename = path

		// Add servers
		c.Servers = map[string]TestServer{}
		for _, serverName := range c.StartServers {
			if server, ok := servers[serverName]; ok {
				c.Servers[serverName] = server
			} else {
				return fmt.Errorf("Server %q not defined in %q", serverName, serversDir)
			}
		}

		// Add test
		if test, ok := tests[c.Name]; ok {
			c.SelfTest = test
			if err := c.ValidateTest(); err != nil {
				return fmt.Errorf("contract test %q invalid: %s", c.Name, err)
			}
		} else {
			// TODO: Emit a warning message
			//cli.Warn("Contract %q has no tests.", c.Name)
		}

		contracts[c.Name] = c
		return nil
	})
	if err != nil {
		return nil, err
	}
	// Add servers to contracts...
	for name, contract := range contracts {
		contract.Servers = map[string]TestServer{}
		for _, serverName := range contract.StartServers {
			server, ok := servers[serverName]
			if !ok {
			}
			contract.Servers[serverName] = server
			contracts[name] = contract
		}
	}
	return contracts, nil
}

func parseServers(serversDir string) (map[string]TestServer, error) {
	servers := map[string]TestServer{}
	err := walkYAMLDir(serversDir, func(path string) error {
		var s TestServer
		if err := parseYAMLFile(path, &s); err != nil {
			return err
		}
		servers[s.Name] = s
		return nil
	})
	if err != nil {
		return nil, err
	}
	return servers, nil
}

func parseTests(testsDir string) (map[string]ContractTest, error) {
	tests := map[string]ContractTest{}
	err := walkYAMLDir(testsDir, func(path string) error {
		var t ContractTest
		if err := parseYAMLFile(path, &t); err != nil {
			return err
		}
		tests[t.ContractName] = t
		return nil
	})
	if err != nil {
		return nil, err
	}
	return tests, nil
}

func parseDatacentres(datacentresDir string) (Datacentres, error) {
	dcs := Datacentres{}
	err := walkYAMLDir(datacentresDir, func(path string) error {
		var dc Datacentre
		if err := parseYAMLFile(path, &dc); err != nil {
			return err
		}
		dcs[dc.Name] = &dc
		return nil
	})
	if err != nil {
		return nil, err
	}
	return dcs, nil
}

func parseYAMLFile(f string, v interface{}) error {
	b, err := ioutil.ReadFile(f)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, v); err != nil {
		return fmt.Errorf("unable to parse %s as %T: %s", f, v, err)
	}
	return nil
}

func walkYAMLDir(d string, fn func(path string) error) error {
	files, err := filepath.Glob(d + "/*.yaml")
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := fn(f); err != nil {
			return err
		}
	}
	return nil
}

func parseManifests(manifestsDir string) (Manifests, error) {
	manifests := Manifests{}
	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), ".yaml") {
			return nil
		}
		manifest, err := parseManifest(manifestsDir, path)
		if err != nil {
			return err
		}
		manifests[manifest.App.SourceRepo] = *manifest
		return nil
	}
	if err := filepath.Walk(manifestsDir, fn); err != nil {
		return nil, err
	}
	return manifests, nil
}

func parseManifest(manifestsDir, path string) (*Manifest, error) {
	manifest := Manifest{}
	if err := parseYAMLFile(path, &manifest); err != nil {
		return nil, err
	}
	relPath, err := filepath.Rel(manifestsDir, path)
	if err != nil {
		return nil, err
	}
	// Check manifest SourceRepo matches path
	expectedSourceRepo := strings.TrimSuffix(relPath, ".yaml")
	if manifest.App.SourceRepo != expectedSourceRepo {
		return nil, fmt.Errorf("SourceRepo was %q; want %q (%s)\nREST:%+v",
			manifest.App.SourceRepo, expectedSourceRepo, path, manifest)
	}
	return &manifest, nil
}
