package docker

import (
	"encoding/json"
	"errors"
	"fmt"
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

	output, err := ctx.Sh.Stdout("docker", cmd...)
	if err != nil {
		return err
	}
	fmt.Println("output : ", output)

	// TODO LH need to figure out what the end state of this should be.
	// Think it needs to detect failure, should test this and return error
	return nil

}

// need to create the container with the mount and then copy out of it
// docker create --mount source=product,target=/app/product ubuntu
// docker cp dee415777a6814df428f4de6a182bf3e545c608306e67e0505aee4676cb16c4a:app/product/. tmp/test/.

func setupTempDir() (string, error) {
	dir, err := ioutil.TempDir("", "sous-split-build")
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
