package docker

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

func build(ctx sous.BuildContext) (string, error) {
	fmt.Println("starting runmount build")

	cmd := []interface{}{"build"}
	// if localImage == false {
	// 	cmd = append(cmd, "--pull")
	// }

	cmd = append(cmd, getOffsetDir(ctx))

	output, err := ctx.Sh.Stdout("docker", cmd...)
	if err != nil {
		return "", err
	}

	return findBuildID(output)
}

func run(ctx sous.BuildContext, buildID string) error {
	fmt.Println("starting runmount run")
	// TODO LH may need to house keep /app/product ?? or do that after artifact is fetched, possible to collide on this on the same agent ?
	cmd := []interface{}{"run", "--mount", "source=cache,target=/cache",
		"--mount", "source=product,target=/app/product"}
	cmd = append(cmd, buildID)

	err := ctx.Sh.Cmd("docker", cmd...).Succeed()
	if err != nil {
		return err
	}
	// fmt.Println("output : ", output)

	// TODO LH need to figure out what the end state of this should be.
	// Think it needs to detect failure, should test this and return error
	return nil

}

// need to create the container with the mount and then copy out of it
// docker create --mount source=product,target=/app/product ubuntu
// docker cp dee415777a6814df428f4de6a182bf3e545c608306e67e0505aee4676cb16c4a:app/product/. tmp/test/.

func setupTempDir() (string, error) {
	dir, err := ioutil.TempDir("", "sous-runmount-build")
	if err != nil {
		return "", err
	}

	tempDir := filepath.Join(dir, "build")
	err = os.MkdirAll(tempDir, os.ModePerm)
	return tempDir, err
}

func createMountContainer(ctx sous.BuildContext, buildID string) (string, error) {
	cmd := []interface{}{"create", "--mount", "source=product,target=/app/product"}
	cmd = append(cmd, buildID)
	output, err := ctx.Sh.Stdout("docker", cmd...)
	if err != nil {
		return "", err
	}

	buildContainerID := strings.TrimSpace(output)

	return buildContainerID, nil
}

func extractRunSpec(ctx sous.BuildContext, tempDir string, buildContainerID string) (MultiImageRunSpec, error) {
	// TODO need to figure out how to pass detected data in
	runSpec := MultiImageRunSpec{}
	runspecPath := "/app/product/run_spec.json" //sb.detected.Data.(detectData).RunImageSpecPath
	destPath := filepath.Join(tempDir, "run_spec.json")
	_, err := ctx.Sh.Stdout("docker", "cp", fmt.Sprintf("%s:%s", buildContainerID, runspecPath), destPath)
	if err != nil {
		return runSpec, err
	}

	specF, err := os.Open(destPath)
	if err != nil {
		return runSpec, err
	}

	dec := json.NewDecoder(specF)
	err = dec.Decode(&runSpec)
	if err != nil {
		return runSpec, err
	}
	return runSpec, nil
}

func validateRunSpec(runSpec MultiImageRunSpec) error {
	flaws := runSpec.Validate()
	if len(flaws) > 0 {
		msg := "Deploy spec invalid:"
		for _, f := range flaws {
			msg += "\n\t" + f.Repair().Error()
		}
		return errors.New(msg)
	}
	return nil
}

func constructImageBuilders(runSpec MultiImageRunSpec) ([]*runnableBuilder, error) {
	rs := runSpec.Normalized()
	subBuilders := []*runnableBuilder{}

	for _, spec := range rs.Images {
		subBuilders = append(subBuilders, &runnableBuilder{
			RunSpec:      spec,
			splitBuilder: nil,
		})
	}

	return subBuilders, nil
}

func extractFiles(ctx sous.BuildContext, buildContainerID string, buildDir string, runnableBuilders []*runnableBuilder) error {

	for _, builder := range runnableBuilders {
		for _, inst := range builder.RunSpec.Files {
			// needs to copy to the per-target build directory
			fromPath := fmt.Sprintf("%s:%s", buildContainerID, inst.Source.Dir)
			toPath := filepath.Join(buildDir, inst.Destination.Dir)

			err := os.MkdirAll(filepath.Dir(toPath), os.ModePerm)
			if err != nil {
				return err
			}
			fmt.Println("fromPath : ", fromPath)
			fmt.Println("toPath : ", toPath)
			_, err = ctx.Sh.Stdout("docker", "cp", fromPath, toPath)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func teardownBuildContainer(ctx sous.BuildContext, buildContainerID string) error {
	_, err := ctx.Sh.Stdout("docker", "rm", buildContainerID)
	return err
}

func templateDockerfile(ctx sous.BuildContext, buildDir string, runnableBuilders []*runnableBuilder) error {
	for _, rb := range runnableBuilders {
		dockerfile, err := os.Create(filepath.Join(buildDir, "Dockerfile"))
		if err != nil {
			return err
		}
		defer dockerfile.Close()

		builder := builder{
			RunSpec:        rb.RunSpec,
			VersionConfig:  versionConfigLocal(ctx),
			RevisionConfig: revisionConfigLocal(ctx),
		}

		err = templateDockerfileBytes(dockerfile, builder)
		if err != nil {
			return err
		}
	}
	return nil
}

type builder struct {
	RunSpec        SplitImageRunSpec
	VersionConfig  string
	RevisionConfig string
}

func templateDockerfileBytes(dockerfile io.Writer, builder builder) error {
	messages.ReportLogFieldsMessage("Templating Dockerfile from", logging.DebugLevel, logging.Log, builder, builder.RunSpec)

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
	fmt.Printf("builder : %v", builder)
	return tmpl.Execute(dockerfile, builder)
}

func buildRunnables(ctx sous.BuildContext, buildDir string, runnableBuilders []*runnableBuilder) error {

	for _, builder := range runnableBuilders {
		_, err := buildRunnable(ctx, buildDir, builder)
		if err != nil {
			return err
		}

	}
	fmt.Println("runnableBuilders : ", runnableBuilders)
	return nil
}
func versionNameLocal(ctx sous.BuildContext) string {
	v := ctx.Version().Version
	v.Meta = ""
	return v.String()
}

func revisionNameLocal(ctx sous.BuildContext) string {
	return ctx.Version().RevID()
}

func versionConfigLocal(ctx sous.BuildContext) string {
	return fmt.Sprintf("%s=%s", AppVersionBuildArg, versionNameLocal(ctx))
}

func revisionConfigLocal(ctx sous.BuildContext) string {
	return fmt.Sprintf("%s=%s", AppRevisionBuildArg, revisionNameLocal(ctx))
}

func buildRunnable(ctx sous.BuildContext, buildDir string, builder *runnableBuilder) (*runnableBuilder, error) {
	sh := ctx.Sh.Clone()
	sh.LongRunning(true)
	sh.CD(buildDir)

	out, err := sh.Stdout("docker", "build", ".")
	if err != nil {
		return nil, err
	}

	match := successfulBuildRE.FindStringSubmatch(string(out))
	if match == nil {
		return nil, fmt.Errorf("Couldn't find container id in:\n%s", out)
	}
	builder.deployImageID = match[1]

	return builder, nil
}

func products(ctx sous.BuildContext, runnableBuilders []*runnableBuilder) []*sous.BuildProduct {
	ps := make([]*sous.BuildProduct, len(runnableBuilders))
	for i, builder := range runnableBuilders {
		ps[i] = product(ctx, builder)
	}
	return ps
}

func product(ctx sous.BuildContext, builder *runnableBuilder) *sous.BuildProduct {
	advisories := ctx.Advisories
	if builder.RunSpec.Kind != "" {
		advisories = append(advisories, string(sous.NotService))
	}
	sid := ctx.Version()
	sid.Location.Dir = builder.RunSpec.Offset

	bp := &sous.BuildProduct{
		Source:     sid,
		Kind:       builder.RunSpec.Kind,
		ID:         builder.deployImageID, // was ImageID
		Advisories: advisories,
		// VersionName:  builder.versionName(),
		// RevisionName: builder.revisionName(),
	}

	return bp
}

func getOffsetDir(ctx sous.BuildContext) string {
	offset := ctx.Source.OffsetDir
	if offset == "" {
		offset = "."
	}
	return offset
}

func getDockerFilePath(ctx sous.BuildContext) string {
	workDir := "."
	fmt.Println("offset : ", ctx.Source.OffsetDir)
	if offset := ctx.Source.OffsetDir; offset != "" {
		workDir = offset
	}
	fmt.Println("workDir : ", workDir)

	dockerFilePath := path.Join(workDir, "Dockerfile")
	return dockerFilePath
}

func findBuildID(cmdOut string) (string, error) {
	match := successfulBuildRE.FindStringSubmatch(cmdOut)
	if match == nil {
		return "", fmt.Errorf("Couldn't find container id in:\n%s", cmdOut)
	}
	return match[1], nil
}
