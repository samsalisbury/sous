package logging

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCPUResourceField(t *testing.T) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.Encode(map[string]interface{}{"test": CPUResourceField(2)})

	t.Log(buf.String())
	assert.Regexp(t, `{"test":2[.]0`, buf.String())
}

func TestMemoryResourceField(t *testing.T) {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.Encode(map[string]interface{}{"test": MemResourceField(200)})

	t.Log(buf.String())
	assert.Regexp(t, `{"test":200}`, buf.String())
}
