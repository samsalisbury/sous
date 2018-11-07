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

type splitDetector struct {
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

func (sd *splitDetector) process() error {
	if err := sd.absorbDocker(sd.rootAst); err != nil {
		return err
	}
	return sd.fetchFromRunSpec()
}

func (sd *splitDetector) fetchFromRunSpec() error {
	for _, f := range sd.froms {
		if sd.dev {
			if imageEnv, err := inspectImage(sd.sh, f.Value); err == nil {
				messages.ReportLogFieldsMessage("Inspecting local", logging.DebugLevel, sd.ls, f.Value)
				sd.mergeEnv(parseImageOutput(imageEnv))
				continue
			}
		}
		messages.ReportLogFieldsMessage("Fetching", logging.DebugLevel, sd.ls, f.Value)
		md, err := sd.registry.GetImageMetadata(f.Value, "")
		if err != nil {
			messages.ReportLogFieldsMessage("Error fetching", logging.DebugLevel, sd.ls, f.Value, err)
			if err != nil {
				continue
			}
		}

		sd.mergeEnv(md.Env)

		buf := bytes.NewBufferString(strings.Join(md.OnBuild, "\n"))
		ast, err := parseDocker(buf)
		messages.ReportLogFieldsMessage("Parsing ONBUILD", logging.DebugLevel, sd.ls, f.Value)
		if err != nil {
			messages.ReportLogFieldsMessage("Error while parsing ONBUILD", logging.DebugLevel, sd.ls, f.Value, err)
			return err
		}
		return sd.absorbDocker(ast)
	}
	return nil
}

func (sd *splitDetector) mergeEnv(env map[string]string) {
	for k, v := range env {
		if _, already := sd.envmap[k]; !already {
			sd.envmap[k] = v
		}
	}
}

/*
func (sd *splitDetector) processEnv() error {
	for _, e := range sd.envs {
		if e.Value == SOUS_RUN_IMAGE_SPEC {
			messages.ReportLogFieldsMessage("RunSpec path found Dockerfile ENV or ONBUILD ENV", logging.DebugLevel, sd.ls, e.Next.Value)
			sd.runspecPath = e.Next.Value
		}
	}
	return nil
}
*/

func (sd *splitDetector) absorbDocker(ast *parser.Node) error {
	// Parse for ENV SOUS_RUN_IMAGE_SPEC
	// Parse for FROM
	envs := []*parser.Node{}
	for _, node := range ast.Children {
		switch node.Value {
		case "env":
			sd.envs = append(sd.envs, node.Next)
			envs = append(envs, node.Next)
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
	sd.mergeEnv(envNodesToMap(envs))
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

func inspectImage(sh shell.Shell, imageName string) (string, error) {
	cmd := []interface{}{"image", "inspect", InspectFormat, imageName}
	//docker image inspect docker.otenv.com/sous-otj-autobuild:local
	result, err := sh.Cmd("docker", cmd...).SucceedResult()
	if err != nil {
		return "", err
	}
	return result.Stdout.String(), nil
}

func (sd *splitDetector) envValue(name string) (string, bool) {
	v, t := sd.envmap[name]
	return v, t
}

var InspectFormat = `--format={{range .Config.OnBuild}}{{.}}
{{end}}{{range .Config.Env}}ENV {{.}}
{{end}}`

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
