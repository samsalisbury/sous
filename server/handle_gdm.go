package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/dto"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

type (
	// GDMResource is the resource for the GDM
	GDMResource struct {
		userExtractor
		context ComponentLocator
	}

	// GETGDMHandler is an injectable request handler
	GETGDMHandler struct {
		logging.LogSink
		GDM *sous.State
	}

	// PUTGDMHandler is an injectable request handler
	PUTGDMHandler struct {
		*http.Request
		logging.LogSink
		//GDM          *sous.State
		StateManager sous.StateManager
		User         ClientUser
	}
)

func newGDMResource(ctx ComponentLocator) *GDMResource {
	return &GDMResource{context: ctx}
}

// Get implements Getable on GDMResource
func (gr *GDMResource) Get(_ *restful.RouteMap, ls logging.LogSink, writer http.ResponseWriter, _ *http.Request, _ httprouter.Params) restful.Exchanger {
	return &GETGDMHandler{
		LogSink: ls,
		GDM:     gr.context.liveState(),
	}
}

// Exchange implements the Handler interface
func (h *GETGDMHandler) Exchange() (interface{}, int) {
	reportDebugHandleGDMMessage(fmt.Sprintf("Get GDM Handler Exchange with GDM: %v", h.GDM), nil, nil, h.LogSink)

	data := dto.GDMWrapper{Deployments: make([]*sous.Deployment, 0)}
	deps, err := h.GDM.Deployments()
	if err != nil {
		return err, http.StatusInternalServerError
	}

	keys := sous.DeploymentIDSlice(deps.Keys())
	sort.Sort(keys)

	for _, k := range keys {
		d, has := deps.Get(k)
		if !has {
			return "Error serializing GDM", http.StatusInternalServerError
		}
		data.Deployments = append(data.Deployments, d)
	}

	return data, http.StatusOK
}

// Put implements Putable on GDMResource
func (gr *GDMResource) Put(_ *restful.RouteMap, ls logging.LogSink, _ http.ResponseWriter, req *http.Request, _ httprouter.Params) restful.Exchanger {
	return &PUTGDMHandler{
		Request: req,
		LogSink: ls,
		//GDM:          gr.context.liveState(),
		StateManager: gr.context.StateManager,
		User:         gr.GetUser(req),
	}
}

// Exchange implements the Handler interface
func (h *PUTGDMHandler) Exchange() (interface{}, int) {
	data := dto.GDMWrapper{}
	dec := json.NewDecoder(h.Request.Body)
	dec.Decode(&data)
	deps := sous.NewDeployments(data.Deployments...)

	reportDebugHandleGDMMessage(fmt.Sprintf("Put GDM Handler Exchange with Client deployments: %v", deps), nil, nil, h.LogSink)

	state, err := h.StateManager.ReadState()
	if err != nil {
		msg := "Error loading state from storage"
		reportHandleGDMMessage(msg, nil, err, h.LogSink)
		return msg, http.StatusInternalServerError
	}

	reportDebugHandleGDMMessage(fmt.Sprintf("Put GDM Handler Exchange with Server State: %v", state), nil, nil, h.LogSink)

	state.Manifests, err = deps.PutbackManifests(state.Defs, state.Manifests, h.LogSink)
	if err != nil {
		msg := "Error getting state"
		reportHandleGDMMessage(msg, nil, err, h.LogSink)
		return msg, http.StatusConflict
	}

	flaws := state.Validate()
	if len(flaws) > 0 {
		msg := "Invalid GDM"
		reportHandleGDMMessage(msg, flaws, nil, h.LogSink)
		return msg, http.StatusBadRequest
	}

	if _, got := h.Header["Etag"]; got {
		state.SetEtag(h.Header.Get("Etag"))
	}

	if err := h.StateManager.WriteState(state, sous.User(h.User)); err != nil {
		msg := "Error committing state"
		reportHandleGDMMessage(msg, flaws, err, h.LogSink)
		return msg, http.StatusInternalServerError
	}
	return "", http.StatusNoContent
}

type handleGDMMessage struct {
	logging.CallerInfo
	msg          string
	flawsMessage sous.FlawMessage
	err          error
	debug        bool
}

func reportDebugHandleGDMMessage(msg string, flaws []sous.Flaw, err error, log logging.LogSink) {
	reportHandleGDMMessage(msg, flaws, err, log, true)
}

func reportHandleGDMMessage(msg string, f []sous.Flaw, err error, log logging.LogSink, debug ...bool) {

	isDebug := false
	if len(debug) > 0 {
		isDebug = debug[0]
	}

	msgLog := handleGDMMessage{
		msg:        msg,
		CallerInfo: logging.GetCallerInfo(logging.NotHere()),
		err:        err,
		flawsMessage: sous.FlawMessage{
			Flaws: f,
		},
		debug: isDebug,
	}
	logging.Deliver(log, msgLog)
}

func (msg handleGDMMessage) DefaultLevel() logging.Level {
	level := logging.WarningLevel

	if msg.debug == true {
		level = logging.DebugLevel
	}

	return level
}

func (msg handleGDMMessage) Message() string {
	return msg.composeMsg()
}

func (msg handleGDMMessage) composeMsg() string {
	errMsg := "nil"
	if msg.err != nil {
		errMsg = msg.err.Error()
	}
	flaws := msg.flawsMessage.ReturnFlawMsg()
	if flaws == "" {
		flaws = "nil"
	}
	return fmt.Sprintf("Handle GDM Message %s: flaws {%s}, error {%s}", msg.msg, flaws, errMsg)
}

func (msg handleGDMMessage) EachField(f logging.FieldReportFn) {
	f("@loglov3-otl", logging.SousGenericV1)

	flaws := msg.flawsMessage.ReturnFlawMsg()

	if flaws != "" {
		f("sous-flaws", flaws)
	}

	if msg.err != nil {
		f("error", msg.err.Error())
	}
	msg.CallerInfo.EachField(f)
}
