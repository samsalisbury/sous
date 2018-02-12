package restful

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPutbackJSON(t *testing.T) {
	origB := bytes.NewBufferString(`{
		"a": 7,
		"b": "b",
		"c": [1,2,3],
		"d": {
			"z": 1,
			"y": [{"q":"q", "z": "o"}],
			"x": "x"
		},
		"e": null
	}`)

	baseB := bytes.NewBufferString(`{
		"b": "b",
		"c": [1,2,3],
		"d": {
			"y": [{"q":"q"}],
			"x": "x"
		},
		"e": null
	}`)

	updatedB := bytes.NewBufferString(`{
		"b": "y",
		"c": [2,3,1],
		"d": {
			"y": [{"q":"w"}, {"zx": "w"}],
			"x": "c"
		},
		"e": {"a": "a"}
	}`)

	outB := putbackJSON(origB, baseB, updatedB)

	// Comparing orginal to output
	mapped := map[string]interface{}{}

	b, err := ioutil.ReadAll(outB)
	assert.NoError(t, err)
	json.Unmarshal(b, &mapped)
	assert.Equal(t, 7.0, mapped["a"]) //missing from base, therefore untouched
	assert.Equal(t, "y", mapped["b"])
	assert.Equal(t, "w", dig(mapped, "d", "y", 0, "q"))
	assert.Equal(t, "w", dig(mapped, "d", "y", 1, "zx"))
	assert.Equal(t, float64(1), dig(mapped, "d", "z"))
	assert.Equal(t, "a", dig(mapped, "e", "a"))
}

func TestClientRetrieve(t *testing.T) {
	assert := assert.New(t)
	lt, ctrl := logging.NewLogSinkSpy()

	s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("{}"))
	}))

	c, err := NewClient(s.URL, lt, map[string]string{})
	require.NoError(t, err)
	body := map[string]interface{}{}

	up, err := c.Retrieve("/path", map[string]string{"query": "present"}, &body, map[string]string{})
	ctrl.DumpLogs(t)
	require.NoError(t, err)

	logCalls := ctrl.CallsTo("LogMessage")
	assert.Contains(up.(*resourceState).qparms, "query")

	var testIndex int
	for i, v := range logCalls {
		logmsg := v.PassedArgs().Get(1).(logging.LogMessage)

		msg := logmsg.Message()

		if strings.Contains(msg, "Client <- body:") {
			testIndex = i
			break
		}
	}

	logLvl := logCalls[testIndex].PassedArgs().Get(0).(logging.Level)
	tstmsg := logCalls[testIndex].PassedArgs().Get(1).(logging.LogMessage)

	assert.Equal(logLvl, logging.InformationLevel)

	fixedFields := map[string]interface{}{
		"@loglov3-otl":    "sous-http-v1",
		"body-size":       int64(0),
		"incoming":        true,
		"method":          "GET",
		"resource-family": "",
		"response-size":   int64(2),
		"status":          200,
		"url-pathname":    "/path",
		"url-querystring": "query=present",
	}

	var variableFields []string
	variableFields = append(
		logging.StandardVariableFields,
		"url",          // depends on the httptest server's random port
		"url-hostname", // likewise
		"duration",
		"call-stack-function", // live log entries are squirrelly
	)
	logging.AssertMessageFields(t, tstmsg, variableFields, fixedFields)
}

func dig(m interface{}, index ...interface{}) interface{} {
	var res interface{}
	has := true
	switch idx := index[0].(type) {
	case string:
		res, has = m.(map[string]interface{})[idx]
	case int:
		res = m.([]interface{})[idx]
	}

	if !has {
		panic("lazarus!")
	}

	if len(index) > 1 {
		return dig(res, index[1:]...)
	}
	return res
}
