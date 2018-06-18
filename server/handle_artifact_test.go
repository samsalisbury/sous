package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPUTArtifact(t *testing.T) {
	art := sous.BuildArtifact{
		VersionName:     "test.reg.com/repo/test:2.2",
		DigestReference: "test.reg.com/repo/test@sha256:123123123123123123123123123123123",
		Qualities:       sous.Qualities{},
	}
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.Encode(art)
	req, err := http.NewRequest("PUT", "", buf)
	if err != nil {
		t.Fatal("error building request", err)
	}

	q, err := url.ParseQuery("repo=github.com/opentable/test&offset=&version=1.2.3")
	if err != nil {
		t.Fatal("error parsing query", err)
	}

	ins, ctrl := sous.NewInserterSpy()

	pah := &PUTArtifactHandler{
		Request:     req,
		QueryValues: restful.QueryValues{Values: q},
		Inserter:    ins,
	}

	_, status := pah.Exchange()

	assert.Equal(t, 200, status, "status")

	require.Len(t, ctrl.CallsTo("Insert"), 1)
	ic := ctrl.CallsTo("Insert")[0]
	inSid := ic.PassedArgs().Get(0).(sous.SourceID)
	inBA := ic.PassedArgs().Get(1).(sous.BuildArtifact)

	assert.Equal(t, "github.com/opentable/test", inSid.Location.Repo, "source id repo")
	assert.Equal(t, art.DigestReference, inBA.DigestReference, "build artifact digest name")
	assert.Equal(t, art.VersionName, inBA.VersionName, "build artifact version name")
}
