package logging

import (
	"fmt"
	"testing"
)

func TestKafkaSinkShouldSend(t *testing.T) {
	testcase := func(msg, sink Level, should bool) {
		t.Run(fmt.Sprintf("Sink %s Msg %s -> %t", msg, sink, should), func(t *testing.T) {
			s := liveKafkaSink{level: sink}
			actual := s.shouldSend(msg)
			if actual != should {
				t.Errorf("s.shouldSend(%s) -> %t, not %t", msg, actual, should)
			}
		})
	}

	testcase(InformationLevel, WarningLevel, false)
	testcase(WarningLevel, WarningLevel, true)
	testcase(CriticalLevel, WarningLevel, true)
}
