package docker

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
)

//go:generate go run ../../bin/includeTmpls.go

type (
	// Build represents a single build of a project.
	Builder struct {
		ImageMapper               *NameCache
		DockerRegistryHost        string
		Context                   *sous.SourceContext
		SourceShell, ScratchShell shell.Shell
		Pack                      sous.Buildpack
	}
	// BuildTarget represents a single target within a Build.
	BuildTarget interface {
		BuildImage()
		BuildContainer()
	}
)

// NewBuildWithShells creates a new build using source code in the working
// directory of sourceShell, and using the working dir of scratchShell as
// temporary storage.
func NewBuilder(nc *NameCache, drh string, c *sous.SourceContext, sourceShell, scratchShell shell.Shell) (*Builder, error) {
	b := &Builder{
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

// GetArtifact implements sous.Registry.GetArtifact
func (b *Builder) GetArtifact(sv sous.SourceVersion) (*sous.BuildArtifact, error) {
	return b.ImageMapper.GetArtifact(sv)
}

// GetSourceVersion implements sous.Registry.GetSourceVersion
func (b *Builder) GetSourceVersion(a *sous.BuildArtifact) (sous.SourceVersion, error) {
	return b.ImageMapper.GetSourceVersion(a)
}

// Build implements sous.Builder.Build
func (b *Builder) Build(bc *sous.BuildContext, bp sous.Buildpack, _ *sous.DetectResult) (*sous.BuildResult, error) {
	br, err := bp.Build(bc)
	if err != nil {
		return nil, err
	}

	err = b.applyMetadata(br)
	if err != nil {
		return nil, err
	}

	err = b.pushToRegistry(br)
	if err != nil {
		return nil, err
	}

	err = b.recordName(br)
	if err != nil {
		return nil, err
	}

	return br, nil
}

// applyMetadata applies container metadata etc. to a container
func (b *Builder) applyMetadata(br *sous.BuildResult) error {
	br.ImageName = b.imageTag(b.Context.Version())
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
		DockerLabels(sv),
	})

	return c.Succeed()
}

// pushToRegistry sends the built image to the registry
func (b *Builder) pushToRegistry(br *sous.BuildResult) error {
	return b.SourceShell.Run("docker", "push", br.ImageName)
}

// recordName inserts metadata about the newly built image into our local name cache
func (b *Builder) recordName(br *sous.BuildResult) error {
	sv := b.Context.Version()
	in := br.ImageName
	b.SourceShell.ConsoleEcho(fmt.Sprintf("[recording \"%s\" as the docker name for \"%s\"]", in, sv.String()))
	return b.ImageMapper.insert(sv, in, "")
}

// imageTag computes an image tag from a SourceVersion
func (b *Builder) imageTag(v sous.SourceVersion) string {
	return filepath.Join(b.DockerRegistryHost, DockerImageName(v))
}
