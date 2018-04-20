package docker

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/util/shell"
	"github.com/stretchr/testify/assert"
)

func Test_parsePartialEnv(t *testing.T) {
	inputEnv := "[PATH=/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/bin APP_BASE=/srv APP_REL=app CONFIG_REL=config BOOTSTRAP_REL=bootstrap JAR_REL=app/main.jar APP_DIR=/srv/app CONFIG_DIR=/srv/config BOOTSTRAP_DIR=/srv/bootstrap OT_JAR=/srv/app/main.jar GNUPGHOME=/srv/config/gnupg SOUS_RUN_IMAGE_SPEC=/run_spec.json]"

	envs := parsePartialEnv(inputEnv)
	assert.Equal(t, len(envs), 12)

	assert.Equal(t, envs[SOUS_RUN_IMAGE_SPEC], "/run_spec.json")
}

func Test_inspectImageForEnv(t *testing.T) {
	sh, _ := shell.Default()
	imageEnv := inspectImageForEnv(sh, "docker.otenv.com/sous-otj-autobuild:latest")
	fmt.Println("output : ", imageEnv)
	assert.FailNow(t, "fail")
}
