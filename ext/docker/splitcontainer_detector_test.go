package docker

import (
	"testing"

	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

var fullEnv = "[ENV BUILD_ROOT /app ENV PRODUCT $BUILD_ROOT/product ENV SOUS_RUN_IMAGE_SPEC_OUTPUT=$PRODUCT/run_spec.json WORKDIR $BUILD_ROOT COPY ./ ./ CMD mvn -B deploy][PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/bin APP_BASE=/srv APP_REL=app CONFIG_REL=config BOOTSTRAP_REL=bootstrap JAR_REL=app/main.jar APP_DIR=/srv/app CONFIG_DIR=/srv/config BOOTSTRAP_DIR=/srv/bootstrap OT_JAR=/srv/app/main.jar GNUPGHOME=/srv/config/gnupg SOUS_RUN_IMAGE_SPEC=/run_spec.json]"
var pathEnv = "[PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/bin APP_BASE=/srv APP_REL=app CONFIG_REL=config BOOTSTRAP_REL=bootstrap JAR_REL=app/main.jar APP_DIR=/srv/app CONFIG_DIR=/srv/config BOOTSTRAP_DIR=/srv/bootstrap OT_JAR=/srv/app/main.jar GNUPGHOME=/srv/config/gnupg SOUS_RUN_IMAGE_SPEC=/run_spec.json]"

func Test_parseImageOutput(t *testing.T) {
	inputEnv := fullEnv
	envs := parseImageOutput(inputEnv)
	assert.Len(t, envs, 13)
	assert.Equal(t, "/run_spec.json", envs[SOUS_RUN_IMAGE_SPEC])
	assert.Equal(t, "$PRODUCT/run_spec.json", envs["SOUS_RUN_IMAGE_SPEC_OUTPUT"])

	inputEnv = pathEnv
	envs = parseImageOutput(inputEnv)
	assert.Len(t, envs, 12)
	assert.Equal(t, "/run_spec.json", envs[SOUS_RUN_IMAGE_SPEC])

	inputEnv = ""
	envs = parseImageOutput(inputEnv)
	assert.Len(t, envs, 0)
}

func Test_inspectImage(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "image")
	cctl.ResultSuccess(fullEnv, "")

	imageEnv := inspectImage(sh, "docker.otenv.com/sous-otj-autobuild:local")
	assert.True(t, len(imageEnv) > 0)
}

func Test_inspectImage_not_found(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "image")
	cctl.ResultSuccess("", "")

	imageEnv := inspectImage(sh, "docker.otenv.com/sous-otj-autobuild:bogus")
	assert.Equal(t, "", imageEnv)
}

func Test_inspectImageForOnBuild(t *testing.T) {
	sh, ctl := shell.NewTestShell()
	_, cctl := ctl.CmdFor("docker", "image")
	cctl.ResultSuccess(fullEnv, "")

	imageOnBuild := inspectImage(sh, "docker.otenv.com/sous-otj-autobuild:local")
	envs := parseImageOutput(imageOnBuild)
	assert.Len(t, envs, 13)
}
