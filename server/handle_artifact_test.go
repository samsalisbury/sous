package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/restful"
)

type artifactTestInserter struct {
	insFunc func(sous.SourceID, string, string, []sous.Quality) error
}

func (ati *artifactTestInserter) Insert(sid sous.SourceID, in, etag string, qs []sous.Quality) error {
	return ati.insFunc(sid, in, etag, qs)
}

func TestPUTArtifact(t *testing.T) {
	art := sous.NewBuildArtifact("test.reg.com/repo/test", sous.Strpairs{})
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

	var inSid sous.SourceID
	var inName string

	pah := &PUTArtifactHandler{
		Request:     req,
		QueryValues: restful.QueryValues{Values: q},
		Inserter: &artifactTestInserter{
			insFunc: func(s sous.SourceID, in, et string, qz []sous.Quality) error {
				inSid = s
				inName = in
				return nil
			},
		},
	}

	_, status := pah.Exchange()
	if status != 200 {
		t.Errorf("status should be 200, was %d", status)
	}
	if inSid.Location.Repo != "github.com/opentable/test" {
		t.Errorf("inserted SID repo was %s, should be github.com/opentable/test", inSid.Location.Repo)
	}
	if inName != "test.reg.com/repo/test" {
		t.Errorf("inserted artifact name was %s, should be test.reg.com/repo/test", inName)
	}
}
