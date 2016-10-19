package docker

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"
	"text/template"

	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/shell"
)

//go:generate go run ../../bin/includeTmpls.go

type (
	// Builder represents a single build of a project.
	Builder struct {
		ImageMapper               sous.Inserter
		DockerRegistryHost        string
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
func NewBuilder(nc sous.Inserter, drh string, sourceShell, scratchShell shell.Shell) (*Builder, error) {
	b := &Builder{
		ImageMapper:        nc,
		DockerRegistryHost: drh,
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

func (b *Builder) debug(msg string) {
	Log.Debug.Printf(msg)
}

func (b *Builder) info(msg string) {
	Log.Info.Printf(msg)
}

// Register registers the build artifact to the the registry
func (b *Builder) Register(br *sous.BuildResult, bc *sous.BuildContext) error {
	err := b.pushToRegistry(br, bc)
	if err != nil {
		return err
	}

	return b.recordName(br, bc)
}

// ApplyMetadata applies container metadata etc. to a container
func (b *Builder) ApplyMetadata(br *sous.BuildResult, bc *sous.BuildContext) error {
	br.VersionName = b.VersionTag(bc.Version())
	br.RevisionName = b.RevisionTag(bc.Version())

	c := b.SourceShell.Cmd("docker", "build", "-t", br.VersionName, "-t", br.RevisionName, "-")
	bf := b.metadataDockerfile(br, bc)
	c.SetStdin(bf)

	return c.Succeed()
}

func (b *Builder) metadataDockerfile(br *sous.BuildResult, bc *sous.BuildContext) io.Reader {
	bf := bytes.Buffer{}
	sv := bc.Version()
	md := template.Must(template.New("metadata").Parse(metadataDockerfileTmpl))
	md.Execute(&bf, struct {
		ImageID    string
		Labels     map[string]string
		Advisories []string
	}{
		br.ImageID,
		Labels(sv),
		br.Advisories,
	})
	return &bf
}

// pushToRegistry sends the built image to the registry
func (b *Builder) pushToRegistry(br *sous.BuildResult, bc *sous.BuildContext) error {
	versionName := b.VersionTag(bc.Version())
	revisionName := b.RevisionTag(bc.Version())
	verr := b.SourceShell.Run("docker", "push", versionName)
	rerr := b.SourceShell.Run("docker", "push", revisionName)

	if verr == nil {
		return rerr
	}
	return verr
}

// recordName inserts metadata about the newly built image into our local name cache
func (b *Builder) recordName(br *sous.BuildResult, bc *sous.BuildContext) error {
	sv := bc.Version()
	in := b.VersionTag(bc.Version())
	b.SourceShell.ConsoleEcho(fmt.Sprintf("[recording \"%s\" as the docker name for \"%s\"]", in, sv.String()))
	var qs []sous.Quality
	for _, adv := range br.Advisories {
		qs = append(qs, sous.Quality{Name: adv, Kind: "advisory"})
	}
	return b.ImageMapper.Insert(sv, in, "", qs)
}

// VersionTag computes an image tag from a SourceVersion's version
func (b *Builder) VersionTag(v sous.SourceID) string {
	Log.Debug.Printf("Version tag: % #v => %s", v, versionName(v))
	return filepath.Join(b.DockerRegistryHost, versionName(v))
}

// RevisionTag computes an image tag from a SourceVersion's revision id
func (b *Builder) RevisionTag(v sous.SourceID) string {
	Log.Debug.Printf("RevisionTag: % #v => %s", v, revisionName(v))
	return filepath.Join(b.DockerRegistryHost, revisionName(v))
}
