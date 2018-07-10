package docker

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/opentable/sous/lib"
)

type runnableBuilder struct {
	RunSpec       SplitImageRunSpec
	splitBuilder  *splitBuilder
	deployImageID string
}

func (rb *runnableBuilder) VersionConfig() string {
	return rb.splitBuilder.versionConfig()
}

func (rb *runnableBuilder) RevisionConfig() string {
	return rb.splitBuilder.revisionConfig()
}

func (rb *runnableBuilder) versionName() string {
	return rb.splitBuilder.versionName()
}

func (rb *runnableBuilder) revisionName() string {
	return rb.splitBuilder.revisionName()
}

func (rb *runnableBuilder) buildDir() string {
	return filepath.Join(rb.splitBuilder.buildDir, rb.RunSpec.Offset)
}

func (rb *runnableBuilder) extractFiles() error {
	sb := rb.splitBuilder

	for _, inst := range rb.RunSpec.Files {
		// needs to copy to the per-target build directory
		fromPath := fmt.Sprintf("%s:%s", sb.buildContainerID, inst.Source.Dir)
		toPath := filepath.Join(rb.buildDir(), inst.Destination.Dir)

		err := os.MkdirAll(filepath.Dir(toPath), os.ModePerm)
		if err != nil {
			return err
		}

		_, err = sb.context.Sh.Stdout("docker", "cp", fromPath, toPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rb *runnableBuilder) templateDockerfileBytes(dockerfile io.Writer) error {
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

	return tmpl.Execute(dockerfile, rb)
}

func (rb *runnableBuilder) templateDockerfile() error {
	dockerfile, err := os.Create(filepath.Join(rb.buildDir(), "Dockerfile"))
	if err != nil {
		return err
	}
	defer dockerfile.Close()

	return rb.templateDockerfileBytes(dockerfile)
}

func (rb *runnableBuilder) build() error {
	sh := rb.splitBuilder.context.Sh.Clone()
	sh.LongRunning(true)
	sh.CD(rb.buildDir())

	itag := intermediateTag()

	_, err := sh.Stdout("docker", "build", "-t", itag, ".")
	if err != nil {
		return err
	}

	rb.deployImageID = itag

	return nil
}

func (rb *runnableBuilder) product() *sous.BuildProduct {
	advisories := rb.splitBuilder.context.Advisories
	if rb.RunSpec.Kind != "" {
		advisories = append(advisories, sous.NotService)
	}
	sid := rb.splitBuilder.context.Version()
	sid.Location.Dir = rb.RunSpec.Offset

	bp := &sous.BuildProduct{
		Source:       sid,
		Kind:         rb.RunSpec.Kind,
		ID:           rb.deployImageID, // was ImageID
		Advisories:   advisories,
		VersionName:  rb.versionName(),
		RevisionName: rb.revisionName(),
	}

	return bp
}
