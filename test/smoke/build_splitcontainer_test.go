//+build smoke

package smoke

import (
	"testing"

	"github.com/opentable/sous/util/filemap"
)

func simpleServerSplitContainer() filemap.FileMap {
	return filemap.FileMap{
		"Dockerfile": `
			FROM alpine:3.2
			ENV SOUS_RUN_IMAGE_SPEC=/image-spec.json
			COPY image-spec.json /
			RUN mkdir /server
			COPY server.sh /server/
			`,
		"image-spec.json": `
			{
			  "image": {
			    "type": "Docker",
				"from": "alpine:3.2"
			  },
			  "files": [
			    {
				  "source": {"dir": "/server"},
			      "dest": {"dir": "/"}
			    }
			  ],
			  "exec": ["/server/server.sh"]
			}
			`,
		"server.sh": `#!/usr/bin/env sh
			echo "Listening on :$PORT0"
			while true; do
			  echo -e "HTTP/1.1 200 OK\n\n$(date)" | nc -l -p $PORT0
			done
			`,
	}
}

func TestSplitContainer(t *testing.T) {

	pf := pfs.newParallelTestFixture(t)

	fixtureConfigs := []fixtureConfig{
		{dbPrimary: false},
		{dbPrimary: true},
	}

	pf.RunMatrix(fixtureConfigs,
		PTest{Name: "simple-splitcontainer", Test: func(t *testing.T, f *TestFixture) {
			client := setupProject(t, f, simpleServerSplitContainer())
			client.MustRun(t, "init", nil, "-kind", "http-service")
			client.MustRun(t, "build", nil, "-tag", "1")
			client.MustRun(t, "deploy", nil, "-cluster", "cluster1", "-tag", "1")
		}},
	)
}
