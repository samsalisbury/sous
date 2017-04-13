package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/docker/docker/builder/dockerfile/parser"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/firsterr"
	"github.com/pkg/errors"
)

type (
	// A SplitBuildpack implements the pattern of using a build container and producing a separate deploy container
	SplitBuildpack struct {
		registry docker_registry.Client
	}
)

// SOUS_RUN_IMAGE_SPEC is the env name that build containers must declare with the path to their runspec file.
const SOUS_RUN_IMAGE_SPEC = "SOUS_RUN_IMAGE_SPEC"

// NewSplitBuildpack returns a new SplitBuildpack
func NewSplitBuildpack(r docker_registry.Client) *SplitBuildpack {
	return &SplitBuildpack{
		registry: r,
	}
}

func parseDocker(f io.Reader) (*parser.Node, error) {
	d := parser.Directive{LookingForDirectives: true}
	parser.SetEscapeToken(parser.DefaultEscapeToken, &d)

	return parser.Parse(f, &d)
}

func parseDockerfile(path string) (*parser.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseDocker(f)
}

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
		md, err := sd.registry.GetImageMetadata(f.Value, "")
		if err != nil {
			continue
		}

		if path, ok := md.Env[SOUS_RUN_IMAGE_SPEC]; ok {
			sd.runspecPath = path
		}

		buf := bytes.NewBufferString(strings.Join(md.OnBuild, "\n"))
		ast, err := parseDocker(buf)
		if err != nil {
			return err
		}
		return sd.absorbDocker(ast)
	}
	return nil
}

func (sd *splitDetector) processEnv() error {
	for _, e := range sd.envs {
		if e.Value == SOUS_RUN_IMAGE_SPEC {
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

// Detect implements Buildpack on SplitBuildpack
func (sbp *SplitBuildpack) Detect(ctx *sous.BuildContext) (*sous.DetectResult, error) {
	dfPath := filepath.Join(ctx.Source.OffsetDir, "Dockerfile")
	if !ctx.Sh.Exists(dfPath) {
		return nil, errors.Errorf("%s does not exist", dfPath)
	}

	ast, err := parseDockerfile(ctx.Sh.Abs(dfPath))
	if err != nil {
		return nil, err
	}

	detector := &splitDetector{
		rootAst:  ast,
		registry: sbp.registry,
		froms:    []*parser.Node{},
		envs:     []*parser.Node{},
	}

	err = firsterr.Returned(
		detector.absorbDockerfile,
		detector.fetchFromRunSpec,
		detector.processEnv,
	)

	return detector.result(), err
}

// Build implements Buildpack on SplitBuildpack
func (sbp *SplitBuildpack) Build(ctx *sous.BuildContext, drez *sous.DetectResult) (*sous.BuildResult, error) {
	start := time.Now()

	script := splitBuilder{context: ctx, detected: drez}

	/*
			docker build <args> <offset> #-> Successfully build (image id)
			docker create <image id> #-> container id
			docker cp <container id>:<SOUS_RUN_IMAGE_SPEC> $TMPDIR/runspec.json
			[parse runspec]
			runspec file <- files @
			  docker cp <container id>:<file.sourcedir> $TMPDIR/<file.destdir>
		  in $TMPDIR docker build - < {templated Dockerfile} #-> Successfully built (image id)
	*/
	err := firsterr.Returned(
		script.buildBuild,
		script.setupTempdir,
		script.createBuildContainer,
		script.extractRunSpec,
		script.validateRunSpec,
		script.extractFiles,
		script.teardownBuildContainer,
		script.templateDockerfile,
		script.buildRunnable,
	)

	return &sous.BuildResult{
		ImageID:    script.deployImageID,
		Elapsed:    time.Since(start),
		Advisories: ctx.Advisories,
		ExtraResults: map[string]*sous.BuildResult{
			"builder": {
				ImageID:    script.buildImageID,
				Elapsed:    time.Since(start),
				Advisories: []string{string(sous.IsBuilder)},
			},
		},
	}, err
}

type splitBuilder struct {
	context          *sous.BuildContext
	detected         *sous.DetectResult
	VersionConfig    string
	RevisionConfig   string
	buildImageID     string
	buildContainerID string
	deployImageID    string
	tempDir          string
	buildDir         string
	RunSpec          *SplitImageRunSpec
}

// A SplitImageRunSpec is the JSON structure that build containers must emit
// in order that their associated deploy container can be assembled.
type SplitImageRunSpec struct {
	Image sbmImage     `json:"image"`
	Files []sbmInstall `json:"files"`
	Exec  []string     `json:"exec"`
}

type sbmImage struct {
	Type string `json:"type"`
	From string `json:"from"`
}

type sbmInstall struct {
	Source      sbmFile `json:"source"`
	Destination sbmFile `json:"dest"`
}

type sbmFile struct {
	Dir string `json: "dir"`
}

// Validate implements Flawed on SplitImageRunSpec
func (rs *SplitImageRunSpec) Validate() []sous.Flaw {
	fs := []sous.Flaw{}
	if strings.ToLower(rs.Image.Type) != "docker" {
		fs = append(fs, sous.FatalFlaw("Only 'docker' is recognized currently as an image type, was %q", rs.Image.Type))
	}
	if rs.Image.From == "" {
		fs = append(fs, sous.FatalFlaw("Required image.from was empty or missing."))
	}
	if len(rs.Files) == 0 {
		fs = append(fs, sous.FatalFlaw("Deploy image doesn't make sense with empty list of files."))
	}
	if len(rs.Exec) == 0 {
		fs = append(fs, sous.FatalFlaw("Need an exec list."))
	}

	return fs
}

func (sb *splitBuilder) buildBuild() error {
	offset := sb.context.Source.OffsetDir
	if offset == "" {
		offset = "."
	}

	v := sb.context.Version().Version
	v.Meta = ""
	sb.VersionConfig = fmt.Sprintf("%s=%s", AppVersionBuildArg, v)
	sb.RevisionConfig = fmt.Sprintf("%s=%s", AppRevisionBuildArg, sb.context.Version().RevID())

	cmd := []interface{}{"build"}
	r := sb.detected.Data.(detectData)
	if r.HasAppVersionArg {
		cmd = append(cmd, "--build-arg", sb.VersionConfig)
	}
	if r.HasAppRevisionArg {
		cmd = append(cmd, "--build-arg", sb.RevisionConfig)
	}

	cmd = append(cmd, offset)

	output, err := sb.context.Sh.Stdout("docker", cmd...)
	if err != nil {
		return err
	}

	match := successfulBuildRE.FindStringSubmatch(string(output))
	if match == nil {
		return fmt.Errorf("Couldn't find container id in:\n%s", output)
	}
	sb.buildImageID = match[1]

	return nil
}

func (sb *splitBuilder) setupTempdir() error {
	dir, err := ioutil.TempDir("", "sous-split-build")
	if err != nil {
		return err
	}
	sb.tempDir = dir
	sb.buildDir = filepath.Join(sb.tempDir, "build")
	return os.MkdirAll(sb.buildDir, os.ModePerm)
}

func (sb *splitBuilder) createBuildContainer() error {
	output, err := sb.context.Sh.Stdout("docker", "create", sb.buildImageID)
	if err != nil {
		return err
	}
	sb.buildContainerID = strings.TrimSpace(output)

	return nil
}

func (sb *splitBuilder) extractRunSpec() error {
	runspecPath := sb.detected.Data.(detectData).RunImageSpecPath
	destPath := filepath.Join(sb.tempDir, "runspec.json")
	_, err := sb.context.Sh.Stdout("docker", "cp", fmt.Sprintf("%s:%s", sb.buildContainerID, runspecPath), destPath)
	if err != nil {
		return err
	}

	specF, err := os.Open(destPath)
	if err != nil {
		return err
	}

	sb.RunSpec = &SplitImageRunSpec{}
	dec := json.NewDecoder(specF)
	return dec.Decode(sb.RunSpec)
}

func (sb *splitBuilder) validateRunSpec() error {
	flaws := sb.RunSpec.Validate()
	if len(flaws) > 0 {
		msg := "Deploy spec invalid:"
		for _, f := range flaws {
			msg += "\n\t" + f.Repair().Error()
		}
		return errors.New(msg)
	}
	return nil
}

func (sb *splitBuilder) extractFiles() error {
	for _, inst := range sb.RunSpec.Files {
		_, err := sb.context.Sh.Stdout("docker", "cp", fmt.Sprintf("%s:%s", sb.buildContainerID, inst.Source.Dir), filepath.Join(sb.buildDir, inst.Destination.Dir))
		if err != nil {
			return err
		}
	}

	return nil
}

func (sb *splitBuilder) teardownBuildContainer() error {
	_, err := sb.context.Sh.Stdout("docker", "rm", sb.buildContainerID)
	if err != nil {
		return err
	}
	return nil
}

func (sb *splitBuilder) templateDockerfileBytes(dockerfile io.Writer) error {
	sous.Log.Debug.Printf("Templating Dockerfile from: %#v %#v", sb, sb.RunSpec)

	tmpl, err := template.New("Dockerfile").Parse(`
	FROM {{.RunSpec.Image.From}}
	{{range .RunSpec.Files }}
	COPY {{.Destination.Dir}} {{.Destination.Dir}}
	{{end}}
	ENV {{.VersionConfig}} {{.RevisionConfig}}
	CMD [
	{{- range $n, $part := .RunSpec.Exec -}}
	  {{if $n}}, {{- end -}}"{{.}}"
	{{- end -}}
	]
	`)
	if err != nil {
		return err
	}

	return tmpl.Execute(dockerfile, sb)
}

func (sb *splitBuilder) templateDockerfile() error {
	dockerfile, err := os.Create(filepath.Join(sb.buildDir, "Dockerfile"))
	if err != nil {
		return err
	}
	defer dockerfile.Close()

	return sb.templateDockerfileBytes(dockerfile)
}

func (sb *splitBuilder) buildRunnable() error {
	sh := sb.context.Sh.Clone()
	sh.LongRunning(true)
	sh.CD(sb.buildDir)

	out, err := sh.Stdout("docker", "build", ".")
	if err != nil {
		return err
	}

	match := successfulBuildRE.FindStringSubmatch(string(out))
	if match == nil {
		return fmt.Errorf("Couldn't find container id in:\n%s", out)
	}
	sb.deployImageID = match[1]

	return nil
}
