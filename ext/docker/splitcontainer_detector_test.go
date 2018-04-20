package docker

import (
	"testing"

	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

func Test_parseImageOutput(t *testing.T) {
	inputEnv := "[ENV BUILD_ROOT /app ENV PRODUCT $BUILD_ROOT/product ENV SOUS_RUN_IMAGE_SPEC_OUTPUT=$PRODUCT/run_spec.json WORKDIR $BUILD_ROOT COPY ./ ./ CMD mvn -B deploy][PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/bin APP_BASE=/srv APP_REL=app CONFIG_REL=config BOOTSTRAP_REL=bootstrap JAR_REL=app/main.jar APP_DIR=/srv/app CONFIG_DIR=/srv/config BOOTSTRAP_DIR=/srv/bootstrap OT_JAR=/srv/app/main.jar GNUPGHOME=/srv/config/gnupg SOUS_RUN_IMAGE_SPEC=/run_spec.json]"
	envs := parseImageOutput(inputEnv)
	assert.Equal(t, len(envs), 13)
	assert.Equal(t, "/run_spec.json", envs[SOUS_RUN_IMAGE_SPEC])
	assert.Equal(t, "$PRODUCT/run_spec.json", envs["SOUS_RUN_IMAGE_SPEC_OUTPUT"])

	inputEnv = "[PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/bin APP_BASE=/srv APP_REL=app CONFIG_REL=config BOOTSTRAP_REL=bootstrap JAR_REL=app/main.jar APP_DIR=/srv/app CONFIG_DIR=/srv/config BOOTSTRAP_DIR=/srv/bootstrap OT_JAR=/srv/app/main.jar GNUPGHOME=/srv/config/gnupg SOUS_RUN_IMAGE_SPEC=/run_spec.json]"
	envs = parseImageOutput(inputEnv)
	assert.Equal(t, 12, len(envs))
	assert.Equal(t, "/run_spec.json", envs[SOUS_RUN_IMAGE_SPEC])

	inputEnv = ""
	envs = parseImageOutput(inputEnv)
	assert.Equal(t, 0, len(envs))
}

func Test_inspectImage(t *testing.T) {
	sh, _ := shell.Default()
	imageEnv := inspectImage(sh, "docker.otenv.com/sous-otj-autobuild:local")
	assert.True(t, len(imageEnv) > 0)
}

func Test_inspectImage_not_found(t *testing.T) {
	sh, _ := shell.Default()
	imageEnv := inspectImage(sh, "docker.otenv.com/sous-otj-autobuild:bogus")
	assert.Equal(t, "", imageEnv)
}

func Test_inspectImageForOnBuild(t *testing.T) {
	sh, _ := shell.Default()
	imageOnBuild := inspectImage(sh, "docker.otenv.com/sous-otj-autobuild:local")
	envs := parseImageOutput(imageOnBuild)
	assert.Equal(t, 13, len(envs))
}
