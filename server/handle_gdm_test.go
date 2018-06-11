package server

import (
	"testing"

	"github.com/opentable/sous/dto"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
)

func TestHandlesGDMGet(t *testing.T) {
	assert := assert.New(t)
	ls, _ := logging.NewLogSinkSpy()

	th := &GETGDMHandler{
		LogSink: ls,
		GDM:     sous.NewState(),
	}

	data, status := th.Exchange()
	//assert.Equal(w.Header().Get("Etag"), etag)
	assert.Equal(status, 200)
	assert.Len(data.(dto.GDMWrapper).Deployments, 0)
}

func TestReturnFlawMsg_nil_flaws(t *testing.T) {
	assert := assert.New(t)

	hmsg := handleGDMMessage{
		CallerInfo:   logging.GetCallerInfo(logging.NotHere()),
		msg:          "test",
		flawsMessage: sous.FlawMessage{Flaws: nil},
		err:          nil,
	}

	//   a.NotPanics(func(){
	//     RemainCalm()
	//   },

	assert.NotPanics(func() {
		hmsg.flawsMessage.ReturnFlawMsg()
	}, "Calling returnFlawMsg should not panic with flaws is nil")

}

func TestReturnFlawMsg(t *testing.T) {

	empty := make(sous.Resources)

	flaws := empty.Validate()
	assert.Len(t, flaws, 3)

	hmsg := handleGDMMessage{
		CallerInfo:   logging.GetCallerInfo(logging.NotHere()),
		msg:          "test",
		flawsMessage: sous.FlawMessage{Flaws: flaws},
		err:          nil,
	}

	flawsMsg := hmsg.flawsMessage.ReturnFlawMsg()

	assert.Contains(t, flawsMsg, "Missing resource")

}
