package restful

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
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
	//ls := logging.NewLogSet(semv.MustParse("0.0.0"), "dummy", "", ioutil.Discard)
	lt, ctrl := logging.NewLogSinkSpy()

	s := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		rw.Write([]byte("{}"))
	}))

	c, err := NewClient(s.URL, lt, map[string]string{})
	require.NoError(t, err)
	body := map[string]interface{}{}

	up, err := c.Retrieve("/path", map[string]string{"query": "present"}, &body, map[string]string{})

	require.NoError(t, err)
	logCalls := ctrl.CallsTo("LogMessage")
	//	require.Len(t, logCalls, 3)
	assert.Contains(up.(*resourceState).qparms, "query")

	logLvl := logCalls[0].PassedArgs().Get(0).(logging.Level)
	msg := logCalls[0].PassedArgs().Get(1).(logging.LogMessage)

	assert.Equal(logLvl, logging.DebugLevel)
	assert.Contains(msg.Message(), "Sending GET")

	//	fixedFields := map[string]interface{}{
	//		"@loglov3-otl": "sous-http-v1",
	//	}

	//	logging.AssertMessageFields(t, msg, append(logging.StandardVariableFields,
	//		logging.HTTPVariableFields...), fixedFields)

	//	logging.AssertMessageFields(t, msg, logging.StandardVariableFields, fixedFields)
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
