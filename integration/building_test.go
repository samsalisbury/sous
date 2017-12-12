// +build integration

package integration

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/opentable/sous/ext/docker"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/opentable/sous/util/logging"
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

func (suite *buildingTestSuite) getImages() []string {
	imagesBytes, err := exec.Command("docker", "images", "-q").Output()
	suite.Require().NoError(err)
	return strings.Split(string(imagesBytes), "\n")
}

func (suite *buildingTestSuite) SetupTest() {
	suite.imagesBefore = suite.getImages()
}

func (suite *buildingTestSuite) TearDownTest() {
	images := suite.getImages()

	for _, img := range images {
		kept := false
		for _, keep := range suite.imagesBefore {
			if keep == img {
				kept = true
				break
			}
		}

		if !kept {
			rmi := exec.Command("docker", "rmi", "-f", img)
			out, err := rmi.Output()
			if err != nil {
				suite.T().Fatalf("Could not remove image %q: %q", img, out)
			} else {
				suite.T().Logf("Removed image: %q", img)
			}
		}
	}
}

func (suite *buildingTestSuite) TestSplitContainer() {

	//return fmt.Sprintf("%s/%s:%s", registryName, reponame, tag)
	reg := docker_registry.NewClient(logging.SilentLogSet())
	sbp := docker.NewSplitBuildpack(reg)

	sh, err := shell.DefaultInDir("testdata/split_test")
	suite.Require().NoError(err)

	ctx := &sous.BuildContext{
		Sh: sh,
		Source: sous.SourceContext{
			NearestTag: sous.Tag{Name: "1.2.3", Revision: "cabbagedeadbeef"},
			Revision:   "cabbagedeadbeef",
		},
	}

	dr, err := sbp.Detect(ctx)
	suite.NoError(err)
	suite.True(dr.Compatible, "Split buildpack reported incompatible project")

	br, err := sbp.Build(ctx)
	suite.NoError(err)
	suite.Require().NotZero(br.Products[0].ID)
	inspectB, err := exec.Command("docker", "inspect", br.Products[0].ID).Output()
	suite.NoError(err)
	inspected := string(inspectB)
	suite.Regexp(`APP_VERSION=1[.]2[.]3`, inspected)
	suite.Regexp(`APP_REVISION=cabbagedeadbeef`, inspected)

}
