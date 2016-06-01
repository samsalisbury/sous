package sous

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/opentable/sous/util/shell"
)

//go:generate go run ../scripts/includeTmpls.go

type (
	// Build represents a single build of a project.
	Build struct {
		DockerRegistryHost        string
		Context                   *SourceContext
		SourceShell, ScratchShell shell.Shell
		Pack                      Buildpack
	}
	// BuildTarget represents a single target within a Build.
	BuildTarget interface {
		BuildImage()
		BuildContainer()
	}
)

// RunBuild does a complete build run
func RunBuild(drh string, ctx *SourceContext, source, scratch shell.Shell) (*BuildResult, error) {
	build, err := NewBuildWithShells(drh, ctx, source, scratch)
	if err != nil {
		return nil, err
	}

	return build.Start()
}

// NewBuildWithShells creates a new build using source code in the working
// directory of sourceShell, and using the working dir of scratchShell as
// temporary storage.
func NewBuildWithShells(drh string, c *SourceContext, sourceShell, scratchShell shell.Shell) (*Build, error) {
	b := &Build{
		DockerRegistryHost: drh,
		Context:            c,
		SourceShell:        sourceShell,
		ScratchShell:       scratchShell,
	}

	files, err := scratchShell.List()
	if err != nil {
		return nil, err
	}

	if len(files) != 0 {
		return nil, fmt.Errorf("scratch dir %s was not empty", scratchShell.Dir())
	}

	return b, nil
}

// Start begins the build.
func (b *Build) Start() (*BuildResult, error) {
	bc := &BuildContext{
		Sh: b.SourceShell,
	}

	bp, err := bc.FindBuildpack()
	if err != nil {

		return nil, err
	}

	br, err := bp.Build(bc)
	if err != nil {
		return nil, err
	}

	err = b.ApplyMetadata(br)
	if err != nil {
		return nil, err
	}

	err = b.PushToRegistry(br)
	if err != nil {
		return nil, err
	}

	return br, nil
}

// ApplyMetadata applies container metadata etc. to a container
func (b *Build) ApplyMetadata(br *BuildResult) error {
	br.ImageName = b.ImageTag(b.Context.Version())
	bf := bytes.Buffer{}

	c := b.SourceShell.Cmd("docker", "build", "-t", br.ImageName, "-")
	c.Stdin(&bf)

	sv := b.Context.Version()

	md := template.Must(template.New("metadata").Parse(metadataDockerfileTmpl))
	md.Execute(&bf, struct {
		ImageID string
		Labels  map[string]string
	}{
		br.ImageID,
		sv.DockerLabels(),
	})

	return c.Succeed()
}

// PushToRegistry sends the built image to the registry
func (b *Build) PushToRegistry(br *BuildResult) error {
	return b.SourceShell.Run("docker", "push", br.ImageName)
}

// ImageTag computes an image tag from a SourceVersion
func (b *Build) ImageTag(v SourceVersion) string {
	return filepath.Join(b.DockerRegistryHost, v.DockerImageName())
}

// FindBuildpack finds the appropriate buildpack for a project
// ( except right now, we just return a hardcoded Dockerfile BP )
// returns an error if no buildpack applies
func (c *BuildContext) FindBuildpack() (Buildpack, error) {
	return NewDockerfileBuildpack(), nil
}
