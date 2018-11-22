package docker

import (
	"bytes"
	"regexp"
	"strings"

	"github.com/docker/docker/builder/dockerfile/parser"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/shell"
)

type dockerfileInspector struct {
	versionArg, revisionArg bool
	runspecPath             string
	registry                docker_registry.Client
	rootAst                 *parser.Node
	froms                   []*parser.Node
	envs                    []*parser.Node
	ls                      logging.LogSink
	sh                      shell.Shell
	dev                     bool
	envmap                  map[string]string
}

func inspectDockerfile(path string, devBuild bool, sh shell.Shell, dfPath string, registry docker_registry.Client, log logging.LogSink) (*dockerfileInspector, error) {

	ast, err := parseDockerfile(path)
	if err != nil {
		return nil, err
	}

	detector := &dockerfileInspector{
		rootAst:  ast,
		registry: registry,
		froms:    []*parser.Node{},
		envs:     []*parser.Node{},
		ls:       log,
		sh:       sh,
		dev:      devBuild,
	}

	return detector, detector.process()
}

func (dfi *dockerfileInspector) process() error {
	if err := dfi.absorbDocker(dfi.rootAst); err != nil {
		return err
	}
	return dfi.fetchFromRunSpec()
}

func (dfi *dockerfileInspector) fetchFromRunSpec() error {
	for _, f := range dfi.froms {
		if dfi.dev {
			if imageEnv, err := inspectImage(dfi.sh, f.Value); err == nil {
				messages.ReportLogFieldsMessage("Inspecting local", logging.DebugLevel, dfi.ls, f.Value)
				dfi.mergeEnv(parseImageOutput(imageEnv))
				continue
			}
		}
		messages.ReportLogFieldsMessage("Fetching", logging.DebugLevel, dfi.ls, f.Value)
		md, err := dfi.registry.GetImageMetadata(f.Value, "")
		if err != nil {
			messages.ReportLogFieldsMessage("Error fetching", logging.DebugLevel, dfi.ls, f.Value, err)
			if err != nil {
				continue
			}
		}

		dfi.mergeEnv(md.Env)

		buf := bytes.NewBufferString(strings.Join(md.OnBuild, "\n"))
		ast, err := parseDocker(buf)
		messages.ReportLogFieldsMessage("Parsing ONBUILD", logging.DebugLevel, dfi.ls, f.Value)
		if err != nil {
			messages.ReportLogFieldsMessage("Error while parsing ONBUILD", logging.DebugLevel, dfi.ls, f.Value, err)
			return err
		}
		return dfi.absorbDocker(ast)
	}
	return nil
}

func (dfi *dockerfileInspector) mergeEnv(env map[string]string) {
	if dfi.envmap == nil {
		dfi.envmap = map[string]string{}
	}
	for k, v := range env {
		if _, already := dfi.envmap[k]; !already {
			dfi.envmap[k] = v
		}
	}
}

/*
func (dfi *dockerfileInspector) processEnv() error {
	for _, e := range dfi.envs {
		if e.Value == SOUS_RUN_IMAGE_SPEC {
			messages.ReportLogFieldsMessage("RunSpec path found Dockerfile ENV or ONBUILD ENV", logging.DebugLevel, dfi.ls, e.Next.Value)
			dfi.runspecPath = e.Next.Value
		}
	}
	return nil
}
*/

func (dfi *dockerfileInspector) absorbDocker(ast *parser.Node) error {
	// Parse for ENV SOUS_RUN_IMAGE_SPEC
	// Parse for FROM
	envs := []*parser.Node{}
	for _, node := range ast.Children {
		switch node.Value {
		case "env":
			dfi.envs = append(dfi.envs, node.Next)
			envs = append(envs, node.Next)
		case "from":
			dfi.froms = append(dfi.froms, node.Next)
		case "arg":
			pair := strings.SplitN(node.Next.Value, "=", 2)
			switch pair[0] {
			case AppVersionBuildArg:
				dfi.versionArg = true
			case AppRevisionBuildArg:
				dfi.revisionArg = true
			}
		}
	}
	dfi.mergeEnv(envNodesToMap(envs))
	return nil
}

func envNodesToMap(envs []*parser.Node) map[string]string {
	m := map[string]string{}
	for _, e := range envs {
		k := e.Value
		v := e.Next.Value
		m[k] = v
	}
	return m
}

var inspectFormat = `--format={{range .Config.OnBuild}}{{.}}
{{end}}{{range .Config.Env}}ENV {{.}}
{{end}}`

func inspectImage(sh shell.Shell, imageName string) (string, error) {
	cmd := []interface{}{"image", "inspect", inspectFormat, imageName}
	//docker image inspect docker.otenv.com/sous-otj-autobuild:local
	result, err := sh.Cmd("docker", cmd...).SucceedResult()
	if err != nil {
		return "", err
	}
	return result.Stdout.String(), nil
}

func (dfi *dockerfileInspector) envValue(name string) (string, bool) {
	v, t := dfi.envmap[name]
	return v, t
}

var envRE = regexp.MustCompile(`(?i)^env`)

func parseImageOutput(input string) map[string]string {
	envs := make(map[string]string)
	elementSlice := strings.Split(input, "\n")
	for _, env := range elementSlice {
		if !envRE.MatchString(env) {
			continue
		}
		env = envRE.ReplaceAllString(env, "")
		env = strings.Trim(env, "\" ")
		envSplit := strings.Split(env, "=")
		if len(envSplit) == 2 {
			key := envSplit[0]
			val := envSplit[1]
			envs[key] = val
		} else {
			envSplit := strings.Split(env, " ")
			if len(envSplit) == 2 {
				key := envSplit[0]
				val := envSplit[1]
				envs[key] = val
			}
		}
	}
	return envs
}
