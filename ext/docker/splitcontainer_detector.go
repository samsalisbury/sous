package docker

import (
	"bytes"
	"strings"

	"github.com/docker/docker/builder/dockerfile/parser"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/logging"
)

type splitDetector struct {
	versionArg, revisionArg bool
	runspecPath             string
	registry                docker_registry.Client
	rootAst                 *parser.Node
	froms                   []*parser.Node
	envs                    []*parser.Node
}

func (sd *splitDetector) absorbDocker(ast *parser.Node) error {
	// Parse for ENV SOUS_RUN_IMAGE_SPEC
	// Parse for FROM
	for _, node := range ast.Children {
		switch node.Value {
		case "env":
			sd.envs = append(sd.envs, node.Next)
		case "from":
			sd.froms = append(sd.froms, node.Next)
		case "arg":
			pair := strings.SplitN(node.Next.Value, "=", 2)
			switch pair[0] {
			case AppVersionBuildArg:
				sd.versionArg = true
			case AppRevisionBuildArg:
				sd.revisionArg = true
			}
		}
	}
	return nil
}

func (sd *splitDetector) absorbDockerfile() error {
	return sd.absorbDocker(sd.rootAst)
}

func (sd *splitDetector) fetchFromRunSpec() error {
	for _, f := range sd.froms {
		logging.Log.Debug.Printf("Fetching FROM %q...", f.Value)
		md, err := sd.registry.GetImageMetadata(f.Value, "")
		if err != nil {
			logging.Log.Debug.Printf("Error fetching %q: %v.", f.Value, err)
			continue
		}

		if path, ok := md.Env[SOUS_RUN_IMAGE_SPEC]; ok {
			logging.Log.Debug.Printf("RunSpec path %q found in %q", path, f.Value)
			sd.runspecPath = path
		}

		buf := bytes.NewBufferString(strings.Join(md.OnBuild, "\n"))
		ast, err := parseDocker(buf)
		logging.Log.Debug.Printf("Parsing ONBUILD from %q.", f.Value)
		if err != nil {
			logging.Log.Debug.Printf("Error while parsing ONBUILD from %q: %#v.", f.Value, err)
			return err
		}
		return sd.absorbDocker(ast)
	}
	return nil
}

func (sd *splitDetector) processEnv() error {
	for _, e := range sd.envs {
		if e.Value == SOUS_RUN_IMAGE_SPEC {
			logging.Log.Debug.Printf("RunSpec path %q found Dockerfile ENV or ONBUILD ENV", e.Next.Value)
			sd.runspecPath = e.Next.Value
		}
	}
	return nil
}

func (sd *splitDetector) result() *sous.DetectResult {
	if sd.runspecPath != "" {
		return &sous.DetectResult{Compatible: true, Data: detectData{
			RunImageSpecPath:  sd.runspecPath,
			HasAppVersionArg:  sd.versionArg,
			HasAppRevisionArg: sd.revisionArg,
		}}
	}
	return &sous.DetectResult{Compatible: false}
}
