package sous

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
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
		}
	}`)
	origCopy := *origB

	baseB := bytes.NewBufferString(`{
		"b": "b",
		"c": [1,2,3],
		"d": {
			"y": [{"q":"q"}],
			"x": "x"
		}
	}`)

	updatedB := bytes.NewBufferString(`{
		"b": "y",
		"c": [2,3,1],
		"d": {
			"y": [{"q":"w"}, {"zx", "w"}],
			"x": "c"
		}
	}`)

	outB := putbackJSON(origB, baseB, updatedB)
	o := map[string]interface{}{}
	spew.Dump(json.Unmarshal(origCopy.Bytes(), &o))
	b, _ := json.Marshal(o)
	spew.Dump(string(b))
	spew.Dump(outB.String())
	mapped := map[string]interface{}{}
	b, err := ioutil.ReadAll(outB)
	assert.NoError(t, err)
	json.Unmarshal(b, &mapped)
	assert.Equal(t, 7.0, mapped["a"]) //missing from update
	assert.Equal(t, "y", mapped["b"])
	assert.Equal(t, "w", dig(mapped, "d", "y", 0, "q"))
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
