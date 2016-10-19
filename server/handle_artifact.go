package server

import (
	"encoding/json"
	"net/http"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/firsterr"
	"github.com/samsalisbury/semv"
)

type (
	ArtifactResource struct{}

	PUTArtifactHandler struct {
		*http.Request
		*QueryValues
		sous.Inserter
	}
)

func (ar *ArtifactResource) Put() *PUTArtifactHandler { return &PUTArtifactHandler{} }

func (pah *PUTArtifactHandler) Exchange() (interface{}, int) {
	ba := sous.BuildArtifact{}
	dec := json.NewDecoder(pah.Request.Body)
	err := dec.Decode(&ba)
	if err != nil {
		return err, http.StatusNotAcceptable
	}

	sid, err := sourceIDFromValues(pah.QueryValues)
	if err != nil {
		return err, http.StatusNotAcceptable
	}

	err = pah.Inserter.Insert(sid, ba.Name, "", ba.Qualities)
	if err != nil {
		return err, http.StatusNotAcceptable
	}

	return "", http.StatusOK
}

func sourceIDFromValues(qv *QueryValues) (sous.SourceID, error) {
	var r, o, vs string
	var v semv.Version
	var err error
	var sid sous.SourceID

	return sid, firsterr.Returned(
		func() error { r, err = qv.Single("repo"); return err },
		func() error { o, err = qv.Single("offset", ""); return err },
		func() error { vs, err = qv.Single("version", ""); return err },
		func() error { v, err = semv.Parse(vs); return err },
		func() error {
			sid = sous.SourceID{
				Location: sous.SourceLocation{
					Repo: r,
					Dir:  o,
				},
				Version: v,
			}
			return nil
		},
	)
}
