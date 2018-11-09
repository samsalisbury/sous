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
)

func build(ctx sous.BuildContext) (string, error) {
	fmt.Println("starting runmount build")

	cmd := []interface{}{"build"}
	if ctx.ShouldPullDuringBuild() {
		cmd = append(cmd, "--pull")
	}

	itag := intermediateTag()
	cmd = append(cmd, "-t", itag)

	cmd = append(cmd, getOffsetDir(ctx))

	_, err := ctx.Sh.Stdout("docker", cmd...)
	if err != nil {
		return "", err
	}

	return itag, nil
}

func run(ctx sous.BuildContext, detection detectData, buildID string) error {
	fmt.Println("starting runmount run")
	// TODO LH may need to house keep /app/product ?? or do that after artifact is fetched, possible to collide on this on the same agent ?
	outputMount := fmt.Sprintf("source=build_output,target=%s", detection.BuildOutPath)
	cmd := []interface{}{"run", "--mount", outputMount}
	if detection.BuildCachePath != "" {
		cacheMount := fmt.Sprintf("source=cache,target=%s", detection.BuildCachePath)
		cmd = append(cmd, "--mount", cacheMount)
	}
	cmd = append(cmd, buildID)

	err := ctx.Sh.Cmd("docker", cmd...).Succeed()
	if err != nil {
		return err
	}

	return nil
}

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
	cmd := []interface{}{"create", "--mount", "source=build_output,target=/build_output"}
	cmd = append(cmd, buildID)
	output, err := ctx.Sh.Stdout("docker", cmd...)
	if err != nil {
		return "", err
	}

	buildContainerID := strings.TrimSpace(output)

	return buildContainerID, nil
}

func extractRunSpec(ctx sous.BuildContext, detection detectData, tempDir string, buildContainerID string) (MultiImageRunSpec, error) {
	// TODO need to figure out how to pass detected data in
	runSpec := MultiImageRunSpec{}
	runspecPath := filepath.Join("/build_output", detection.RunImageSpecPath)
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

func constructImageBuilders(runSpec MultiImageRunSpec) []*runnableBuilder {
	rs := runSpec.Normalized()
	subBuilders := []*runnableBuilder{}

	for _, spec := range rs.Images {
		subBuilders = append(subBuilders, &runnableBuilder{
			RunSpec:      spec,
			splitBuilder: nil,
		})
	}

	return subBuilders
}

func extractFiles(ctx sous.BuildContext, buildContainerID string, buildDir string, runnableBuilders []*runnableBuilder) error {

	for _, builder := range runnableBuilders {
		for _, inst := range builder.RunSpec.Files {
			// needs to copy to the per-target build directory
			fromTemp := filepath.Join("/build_output", inst.Source.Dir)
			fromPath := fmt.Sprintf("%s:%s", buildContainerID, fromTemp)
			toPath := filepath.Join(buildDir, builder.RunSpec.Offset, inst.Destination.Dir)

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
		dockerfile, err := os.Create(filepath.Join(buildDir, rb.RunSpec.Offset, "Dockerfile"))
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
	return ctx.RevID()
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
	workDir := filepath.Join(buildDir, builder.RunSpec.Offset)
	sh.CD(workDir)

	itag := intermediateTag()

	if _, err := sh.Stdout("docker", "build", "-t", itag, "."); err != nil {
		return nil, err
	}

	builder.deployImageID = itag

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
		advisories = append(advisories, sous.NotService)
	}
	sid := ctx.Version()
	sid.Location.Dir = builder.RunSpec.Offset

	bp := &sous.BuildProduct{
		Source:       sid,
		Kind:         builder.RunSpec.Kind,
		ID:           builder.deployImageID, // was ImageID
		Advisories:   advisories,
		VersionName:  versionNameLocal(ctx),
		RevisionName: revisionNameLocal(ctx),
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
