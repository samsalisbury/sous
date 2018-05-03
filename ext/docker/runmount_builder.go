package docker

import (
	"fmt"
	"path"

	sous "github.com/opentable/sous/lib"
)

func build(ctx sous.BuildContext) (string, error) {
	fmt.Println("starting runmount build")

	cmd := []interface{}{"build"}
	// if localImage == false {
	// 	cmd = append(cmd, "--pull")
	// }

	cmd = append(cmd, getDockerFilePath(ctx))

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
	return nil

}

func getDockerFilePath(ctx sous.BuildContext) string {
	workDir := "."
	if offset := ctx.Source.OffsetDir; offset != "" {
		workDir = offset
	}
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
