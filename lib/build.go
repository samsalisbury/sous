package sous

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/opentable/sous/util/shell"
)

//go:generate go run ../bin/includeTmpls.go

type (
	// Build represents a single build of a project.
	Build struct {
		ImageMapper               ImageMapper
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
func RunBuild(nc ImageMapper, drh string, ctx *SourceContext, source, scratch shell.Shell) (*BuildResult, error) {
	build, err := NewBuildWithShells(nc, drh, ctx, source, scratch)
	if err != nil {
		return nil, err
	}

	return build.Start()
}

// NewBuildWithShells creates a new build using source code in the working
// directory of sourceShell, and using the working dir of scratchShell as
// temporary storage.
func NewBuildWithShells(nc ImageMapper, drh string, c *SourceContext, sourceShell, scratchShell shell.Shell) (*Build, error) {
	b := &Build{
		ImageMapper:        nc,
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

	err = b.RecordName(br)
	if err != nil {
		return nil, err
	}

	return br, nil
}

// ApplyMetadata applies container metadata etc. to a container
func (b *Build) ApplyMetadata(br *BuildResult) error {
	br.VersionName = b.VersionTag(b.Context.Version())
	br.RevisionName = b.RevisionTag(b.Context.Version())
	bf := bytes.Buffer{}

	c := b.SourceShell.Cmd("docker", "build", "-t", br.VersionName, "-t", br.RevisionName, "-")
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
	ve := b.SourceShell.Run("docker", "push", br.VersionName)
	re := b.SourceShell.Run("docker", "push", br.RevisionName)
	if ve != nil {
		return ve
	}
	return re
}

// RecordName inserts metadata about the newly built image into our local name cache
func (b *Build) RecordName(br *BuildResult) error {
	sv := b.Context.Version()
	in := br.VersionName
	b.SourceShell.ConsoleEcho(fmt.Sprintf("[recording \"%s\" as the docker name for \"%s\"]", in, sv.String()))
	return b.ImageMapper.Insert(sv, in, "")
}

// VersionTag computes an image tag from a SourceVersion's version
func (b *Build) VersionTag(v SourceVersion) string {
	return filepath.Join(b.DockerRegistryHost, v.DockerVersionName())
}

// RevisionTag computes an image tag from a SourceVersion's revision id
func (b *Build) RevisionTag(v SourceVersion) string {
	return filepath.Join(b.DockerRegistryHost, v.DockerRevisionName())
}

// FindBuildpack finds the appropriate buildpack for a project
// ( except right now, we just return a hardcoded Dockerfile BP )
// returns an error if no buildpack applies
func (c *BuildContext) FindBuildpack() (Buildpack, error) {
	return NewDockerfileBuildpack(), nil
}
