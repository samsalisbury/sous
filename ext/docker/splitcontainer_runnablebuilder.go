package docker

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"

	"github.com/opentable/sous/util/logging"
)

type runnableBuilder struct {
	RunSpec       SplitImageRunSpec
	splitBuilder  *splitBuilder
	deployImageID string
}

func (rb *runnableBuilder) VersionConfig() string {
	return rb.splitBuilder.VersionConfig
}

func (rb *runnableBuilder) RevisionConfig() string {
	return rb.splitBuilder.RevisionConfig
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

		_, err := sb.context.Sh.Stdout("docker", "cp", fromPath, toPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (rb *runnableBuilder) templateDockerfileBytes(dockerfile io.Writer) error {
	logging.Log.Debug.Printf("Templating Dockerfile from: %#v %#v", rb, rb.RunSpec)

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

	out, err := sh.Stdout("docker", "build", ".")
	if err != nil {
		return err
	}

	match := successfulBuildRE.FindStringSubmatch(string(out))
	if match == nil {
		return fmt.Errorf("Couldn't find container id in:\n%s", out)
	}
	rb.deployImageID = match[1]

	return nil
}
