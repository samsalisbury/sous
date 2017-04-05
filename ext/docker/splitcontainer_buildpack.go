package docker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
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

// SOUS_BUILD_MANIFEST is the env name that build containers must declare with the path to their manifest file.
const SOUS_BUILD_MANIFEST = "SOUS_BUILD_MANIFEST"

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
	manifestPath            string
	registry                docker_registry.Client
	rootAst                 *parser.Node
	froms                   []*parser.Node
	envs                    []*parser.Node
}

func (sd *splitDetector) absorbDocker(ast *parser.Node) error {
	// Parse for ENV SOUS_BUILD_MANIFEST
	// Parse for FROM
	for n, node := range ast.Children {
		switch node.Value {
		case "env":
			sd.envs = append(sd.envs, node.Next)
			log.Printf("%d %#v", n, node.Next)
		case "from":
			sd.froms = append(sd.froms, node.Next)
			log.Printf("%d %#v", n, node.Next)
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

func (sd *splitDetector) fetchFromManifest() error {
	for _, f := range sd.froms {
		md, err := sd.registry.GetImageMetadata(f.Value, "")
		if err != nil {
			continue
		}

		if path, ok := md.Env[SOUS_BUILD_MANIFEST]; ok {
			sd.manifestPath = path
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
		if e.Value == SOUS_BUILD_MANIFEST {
			sd.manifestPath = e.Next.Value
		}
	}
	return nil
}

func (sd *splitDetector) result() *sous.DetectResult {
	if sd.manifestPath != "" {
		return &sous.DetectResult{Compatible: true, Data: detectData{
			ManifestPath:      sd.manifestPath,
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
		detector.fetchFromManifest,
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
			docker cp <container id>:<SOUS_BUILD_MANIFEST> $TMPDIR/manifest.json
			[parse manifest]
			manifest file <- files @
			  docker cp <container id>:<file.sourcedir> $TMPDIR/<file.destdir>
		  in $TMPDIR docker build - < {templated Dockerfile} #-> Successfully built (image id)
	*/
	err := firsterr.Returned(
		script.buildBuild,
		script.createBuildContainer,
		script.setupTempdir,
		script.extractManifest,
		script.extractFiles,
		script.templateDockerfile,
		script.buildRunnable,
	)

	return &sous.BuildResult{
		ImageID:    script.deployImageID,
		Elapsed:    time.Since(start),
		Advisories: ctx.Advisories,
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
	Manifest         *SplitBuildManifest
}

// A SplitBuildManifest is the JSON structure that build containers must emit
// in order that their associated deploy container can be assembled.
type SplitBuildManifest struct {
	Container sbmContainer `json:"container"`
	Files     []sbmInstall `json:"files"`
	Exec      []string     `json:"exec"`
}

type sbmContainer struct {
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

func (sb *splitBuilder) createBuildContainer() error {
	output, err := sb.context.Sh.Stdout("docker", "create", sb.buildImageID)
	if err != nil {
		return err
	}
	sb.buildContainerID = strings.TrimSpace(output)

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

func (sb *splitBuilder) extractManifest() error {
	manifestPath := sb.detected.Data.(detectData).ManifestPath
	destPath := filepath.Join(sb.tempDir, "manifest.json")
	_, err := sb.context.Sh.Stdout("docker", "cp", fmt.Sprintf("%s:%s", sb.buildContainerID, manifestPath), destPath)
	if err != nil {
		return err
	}

	manifestF, err := os.Open(destPath)
	if err != nil {
		return err
	}

	sb.Manifest = &SplitBuildManifest{}
	dec := json.NewDecoder(manifestF)
	dec.Decode(sb.Manifest)

	return err
}

func (sb *splitBuilder) extractFiles() error {
	for _, inst := range sb.Manifest.Files {
		_, err := sb.context.Sh.Stdout("docker", "cp", fmt.Sprintf("%s:%s", sb.buildContainerID, inst.Source.Dir), filepath.Join(sb.buildDir, inst.Destination.Dir))
		if err != nil {
			return err
		}
	}

	return nil
}

func (sb *splitBuilder) templateDockerfileBytes(dockerfile io.Writer) error {
	tmpl, err := template.New("Dockerfile").Parse(`
	FROM {{.Manifest.Container.From}}
	{{range .Manifest.Files }}
	COPY {{.Destination.Dir}} {{.Destination.Dir}}
	{{end}}
	ENV {{.VersionConfig}} {{.RevisionConfig}}
	CMD [
	{{- range $n, $part := .Manifest.Exec -}}
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
