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
	// Builder represents a single build of a project.
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

// NewBuilder creates a new build using source code in the working
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

// Build implements sous.Builder.Build
func (b *Builder) Build(bc *sous.BuildContext, bp sous.Buildpack, _ *sous.DetectResult) (*sous.BuildResult, error) {
	br, err := bp.Build(bc)
	if err != nil {
		return nil, err
	}

	err = b.ApplyMetadata(br)
	if err != nil {
		return nil, err
	}

	err = b.Register(br)
	if err != nil {
		return nil, err
	}

	return br, nil
}

// Register registers the build artifact to the the registry
func (b *Builder) Register(br *sous.BuildResult) error {
	err = b.pushToRegistry(br)
	if err != nil {
		return nil, err
	}

	return b.recordName(br)
}

// ApplyMetadata applies container metadata etc. to a container
func (b *Builder) ApplyMetadata(br *sous.BuildResult) error {
	versionName = b.VersionTag(b.Context.Version())
	revisionName = b.RevisionTag(b.Context.Version())
	bf := bytes.Buffer{}

	c := b.SourceShell.Cmd("docker", "build", "-t", versionName, "-t", revisionName, "-")
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
	versionName = b.VersionTag(b.Context.Version())
	revisionName = b.RevisionTag(b.Context.Version())
	verr := b.SourceShell.Run("docker", "push", versionName)
	rerr := b.SourceShell.Run("docker", "push", revisionName)

	if verr == nil {
		return rerr
	}
	return verr
}

// recordName inserts metadata about the newly built image into our local name cache
func (b *Builder) recordName(br *sous.BuildResult) error {
	sv := b.Context.Version()
	in = b.VersionTag(b.Context.Version())
	b.SourceShell.ConsoleEcho(fmt.Sprintf("[recording \"%s\" as the docker name for \"%s\"]", in, sv.String()))
	return b.ImageMapper.insert(sv, in, "")
}

// VersionTag computes an image tag from a SourceVersion's version
func (b *Builder) VersionTag(v sous.SourceVersion) string {
	return filepath.Join(b.DockerRegistryHost, versionName(v))
}

// RevisionTag computes an image tag from a SourceVersion's revision id
func (b *Builder) RevisionTag(v sous.SourceVersion) string {
	return filepath.Join(b.DockerRegistryHost, revisionName(v))
}
