package docker

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/nyarly/inlinefiles/templatestore"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
	"github.com/opentable/sous/util/shell"
)

//go:generate inlinefiles --vfs=templateVFS tmpl/ templates_vfs.go

type (
	// Builder represents a single build of a project.
	Builder struct {
		ImageMapper               sous.Inserter
		DockerRegistryHost        string
		SourceShell, ScratchShell shell.Shell
		Pack                      sous.Buildpack
		log                       logging.LogSink
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
func NewBuilder(nc sous.Inserter, drh string, sourceShell, scratchShell shell.Shell, ls logging.LogSink) (*Builder, error) {
	b := &Builder{
		ImageMapper:        nc,
		DockerRegistryHost: drh,
		SourceShell:        sourceShell,
		ScratchShell:       scratchShell,
		log:                ls,
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

// ApplyMetadata implements sous.Labeller on Builder.
// It applies container metadata etc. to a container.
func (b *Builder) ApplyMetadata(br *sous.BuildResult) error {
	for _, prod := range br.Products {
		err := b.applyMetadata(prod)
		if err != nil {
			return err
		}
	}
	return nil
}

// Register registers the build artifact to the the registry
func (b *Builder) Register(br *sous.BuildResult) error {
	for _, prod := range br.Products {
		err := b.pushToRegistry(prod)
		if err != nil {
			return err
		}

		err = b.recordName(prod)
		if err != nil {
			return err
		}
	}
	return nil
}

func (b *Builder) debug(msg string) {
	messages.ReportLogFieldsMessage(msg, logging.DebugLevel, b.log)
}

func (b *Builder) info(msg string) {
	messages.ReportLogFieldsMessage(msg, logging.InformationLevel, b.log)
}

func (b *Builder) applyMetadata(bp *sous.BuildProduct) error {
	bp.VersionName = b.VersionTag(bp.Source, bp.Kind)
	bp.RevisionName = b.RevisionTag(bp.Source, bp.RevisionName, bp.Kind, time.Now())

	c := b.SourceShell.Cmd("docker", "build", "-t", bp.VersionName, "-t", bp.RevisionName, "-")
	bf := b.metadataDockerfile(bp)
	c.SetStdin(bf)

	return c.Succeed()
}

func (b *Builder) metadataDockerfile(bp *sous.BuildProduct) io.Reader {
	bf := bytes.Buffer{}
	sv := bp.Source
	md, err := templatestore.LoadText(templateVFS, "metadata", "metadataDockerfile.tmpl")
	if err != nil {
		panic(err)
	}

	md.Execute(&bf, struct {
		ImageID    string
		Labels     map[string]string
		Advisories []string
	}{
		bp.ID,
		Labels(sv, bp.RevID),
		bp.Advisories,
	})
	return &bf
}

// pushToRegistry sends the built image to the registry
func (b *Builder) pushToRegistry(bp *sous.BuildProduct) error {
	verr := b.SourceShell.Run("docker", "push", bp.VersionName)
	rerr := b.SourceShell.Run("docker", "push", bp.RevisionName)

	if verr == nil {
		return rerr
	}
	return verr
}

// recordName inserts metadata about the newly built image into our local name cache
func (b *Builder) recordName(bp *sous.BuildProduct) error {
	sv := bp.Source
	logging.DebugConsole(b.log, fmt.Sprintf("[recording \"%s\" as the docker name for \"%s\"]", bp.VersionName, sv.String()))
	return b.ImageMapper.Insert(sv, bp.BuildArtifact())
}

// VersionTag computes an image tag from a SourceVersion's version
func (b *Builder) VersionTag(v sous.SourceID, kind string) string {
	return versionTag(b.DockerRegistryHost, v, kind, b.log)
}

// RevisionTag computes an image tag from a SourceVersion's revision id
func (b *Builder) RevisionTag(v sous.SourceID, rev string, kind string, time time.Time) string {
	return revisionTag(b.DockerRegistryHost, v, rev, kind, time, b.log)
}
