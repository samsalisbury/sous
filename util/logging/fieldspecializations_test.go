package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResourceField(t *testing.T) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.Encode(map[string]interface{}{"test": resourceField(2)})

	t.Log(buf.String())
	assert.Regexp(t, `2[.]0`, buf.String())
}
