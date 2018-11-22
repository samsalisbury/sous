package docker

import (
	"testing"

	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

var fullEnv = "ENV BUILD_ROOT /app\nENV PRODUCT $BUILD_ROOT/product\nENV SOUS_RUN_IMAGE_SPEC_OUTPUT=$PRODUCT/run_spec.json\nWORKDIR $BUILD_ROOT\nCOPY ./ ./\nCMD mvn -B deploy\nENV PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/bin\nENV APP_BASE=/srv\nENV APP_REL=app\nENV CONFIG_REL=config\nENV BOOTSTRAP_REL=bootstrap\nENV JAR_REL=app/main.jar\nENV APP_DIR=/srv/app\nENV CONFIG_DIR=/srv/config\nENV BOOTSTRAP_DIR=/srv/bootstrap\nENV OT_JAR=/srv/app/main.jar\nENV GNUPGHOME=/srv/config/gnupg\nENV SOUS_RUN_IMAGE_SPEC=/run_spec.json"
var pathEnv = "ENV PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/bin\nENV APP_BASE=/srv\nENV APP_REL=app\nENV CONFIG_REL=config\nENV BOOTSTRAP_REL=bootstrap\nENV JAR_REL=app/main.jar\nENV APP_DIR=/srv/app\nENV CONFIG_DIR=/srv/config\nENV BOOTSTRAP_DIR=/srv/bootstrap\nENV OT_JAR=/srv/app/main.jar\nENV GNUPGHOME=/srv/config/gnupg\nENV SOUS_RUN_IMAGE_SPEC=/run_spec.json"

func Test_parseImageOutput(t *testing.T) {
	envs := parseImageOutput(fullEnv)
	assert.Len(t, envs, 15)
	assert.Contains(t, envs, SOUS_RUN_IMAGE_SPEC)
	assert.Equal(t, envs[SOUS_RUN_IMAGE_SPEC], "/run_spec.json")
	assert.Equal(t, envs["SOUS_RUN_IMAGE_SPEC_OUTPUT"], "$PRODUCT/run_spec.json")

	envs = parseImageOutput(pathEnv)
	assert.Len(t, envs, 12)
	assert.Contains(t, envs, SOUS_RUN_IMAGE_SPEC)
	assert.Equal(t, envs[SOUS_RUN_IMAGE_SPEC], "/run_spec.json")

	envs = parseImageOutput("")
	assert.Len(t, envs, 0)
}

func Test_inspectImage(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "image")
	cctl.ResultSuccess(fullEnv, "")

	imageEnv, err := inspectImage(sh, "docker.otenv.com/sous-otj-autobuild:local")
	assert.NoError(t, err)
	assert.True(t, len(imageEnv) > 0)
}

func Test_inspectImage_not_found(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "image")
	cctl.ResultFailure("", "Image Not Found")

	imageEnv, err := inspectImage(sh, "docker.otenv.com/sous-otj-autobuild:bogus")
	assert.Error(t, err)
	assert.Equal(t, "", imageEnv)
}

func Test_inspectImageForOnBuild(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "image")
	cctl.ResultSuccess(fullEnv, "")

	imageOnBuild, err := inspectImage(sh, "docker.otenv.com/sous-otj-autobuild:local")
	assert.NoError(t, err)
	envs := parseImageOutput(imageOnBuild)
	assert.Len(t, envs, 15)
}
