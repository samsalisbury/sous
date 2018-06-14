package server

import (
	"encoding/json"
	"net/http"

	"github.com/julienschmidt/httprouter"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/firsterr"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/samsalisbury/semv"
)

type (
	// ArtifactResource provides the /artifact endpoint
	ArtifactResource struct {
		restful.QueryParser
		context ComponentLocator
	}

	// PUTArtifactHandler handles PUT requests to /artifact
	PUTArtifactHandler struct {
		*http.Request
		restful.QueryValues
		sous.Inserter
	}
)

func newArtifactResource(ctx ComponentLocator) *ArtifactResource {
	return &ArtifactResource{context: ctx}
}

// Put implements Putable on ArtifactResource, which marks it as accepting PUT requests
func (ar *ArtifactResource) Put(_ *restful.RouteMap, _ logging.LogSink, _ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &PUTArtifactHandler{
		Request:     req,
		QueryValues: ar.ParseQuery(req),
		Inserter:    ar.context.Inserter,
	}
}

// Exchange implements Exchanger on PUTArtifactHandler
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

	err = pah.Inserter.Insert(sid, ba)
	if err != nil {
		return err, http.StatusNotAcceptable
	}

	return "{}", http.StatusOK
}

func sourceIDFromValues(qv restful.QueryValues) (sous.SourceID, error) {
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
