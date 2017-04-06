// +build integration

package integration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/suite"
)

type buildingTestSuite struct {
	suite.Suite
	imagesBefore []string
}

func TestBuilding(t *testing.T) {
	suite.Run(t, new(buildingTestSuite))
}

func (suite *buildingTestSuite) SetupTest() {
	imagesBytes, err := exec.Command("docker", "images").Output()
	suite.Require().NoError(err)
	suite.imagesBefore = strings.Split(string(imagesBytes), "\n")
}

func (suite *buildingTestSuite) TeardownTest() {
	imagesBytes, err := exec.Command("docker", "images").Output()
	suite.Require().NoError(err)
	images := strings.Split(string(imagesBytes), "\n")

	for _, img := range images {
		kept := false
		for _, keep := range suite.imagesBefore {
			if keep == img {
				kept = true
				break
			}
		}

		if !kept {
			exec.Command("docker", "rmi", img)
		}
	}
}

func (suite *buildingTestSuite) TestSplitContainer() {

	//return fmt.Sprintf("%s/%s:%s", registryName, reponame, tag)
	reg := docker_registry.NewClient()
	sbp := docker.NewSplitBuildpack(reg)

	sh, err := shell.DefaultInDir("testdata/split_test")
	suite.Require().NoError(err)

	ctx := &sous.BuildContext{
		Sh: sh,
		Source: sous.SourceContext{
			NearestTagName: "1.2.3",
			Revision:       "cabbagedeadbeef",
		},
	}

	dr, err := sbp.Detect(ctx)
	suite.NoError(err)
	suite.True(dr.Compatible, "Split buildpack reported incompatible project")

	br, err := sbp.Build(ctx, dr)
	suite.NoError(err)
	suite.Require().NotZero(br.ImageID)
	inspectB, err := exec.Command("docker", "inspect", br.ImageID).Output()
	suite.NoError(err)
	inspected := string(inspectB)
	suite.Regexp(`APP_VERSION=1[.]2[.]3`, inspected)
	suite.Regexp(`APP_REVISION=cabbagedeadbeef`, inspected)

}
