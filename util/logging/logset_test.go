package logging

import (
	"io/ioutil"
	"testing"

	"github.com/samsalisbury/semv"
	"github.com/stretchr/testify/assert"
)

func TestLogSetContext(t *testing.T) {
	ls := NewLogSet(semv.MustParse("0.0.0"), "test", "test", ioutil.Discard)
	kafka, ctrl := newKafkaSinkSpy()

	ls.replaceKafka(kafka)

	child := ls.Child("child", KV("child-value", 1), KV("override", 2))
	grandchild := child.Child("grandchild", KV("gc-value", 10), KV("override", 20))

	Deliver(grandchild, KV("logged", 100))
	sent := ctrl.CallsTo("send")
	if assert.Len(t, sent, 1) {
		msg := sent[0].PassedArgs().Get(1)
		assert.Equal(t, msg, "a carrot!")
	}
}
