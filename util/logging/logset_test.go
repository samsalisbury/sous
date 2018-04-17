package logging

import (
	"io/ioutil"
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogSetContext(t *testing.T) {
	ls := NewLogSet(semv.MustParse("0.0.0"), "test", "test", ioutil.Discard)
	kafka, ctrl := newKafkaSinkSpy()

	ls.replaceKafka(kafka)

	child := ls.Child("child", KV("child-value", 1), KV("override", 2))
	grandchild := child.Child("grandchild", KV("gc-value", 10), KV("override", 20))

	Deliver(grandchild, KV("logged", 100), KV("logged", "extra"))
	sent := ctrl.CallsTo("send")
	require.Len(t, sent, 1)
	msg := sent[0].PassedArgs().Get(1)
	entry := msg.(*logrus.Entry)
	data := entry.Data
	assert.Equal(t, 100, data["logged"])
	assert.Equal(t, 10, data["gc-value"])
	assert.Equal(t, 20, data["override"])
	assert.Equal(t, 1, data["child-value"])
	assert.Equal(t, "test.child.grandchild", data["logger-name"])
	assert.Equal(t, `{"message":{"redundant":{"logged":["extra"]}}}`, data["json-value"])
}
