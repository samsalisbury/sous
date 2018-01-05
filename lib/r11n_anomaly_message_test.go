package sous

import (
	"fmt"
	"testing"
)

func TestR11nAnomalyMessage(t *testing.T) {
	testCases := []struct {
		r           *Rectification
		a           r11nAnomaly
		wantMessage string
		wantDesc    ResolutionType
	}{
		{
			&Rectification{},
			r11nDroppedQueueNotEmpty,
			"rectification dropped: queue not empty",
			"not created (not attempted)",
		},
		{
			&Rectification{},
			r11nDroppedQueueFull,
			"rectification dropped: queue full",
			"not created (not attempted)",
		},
		{
			&Rectification{},
			r11nWentMissing,
			"rectification went missing: reason unknown",
			"not created (went missing)",
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_%s", tc.wantMessage, tc.wantDesc), func(t *testing.T) {
			m := newR11nAnomalyMessage(tc.r, tc.a)
			gotMessage, gotDesc := m.Message(), m.resolution.Desc

			if gotMessage != tc.wantMessage {
				t.Errorf("got message %q; want %q", gotMessage, tc.wantMessage)
			}
			if gotDesc != tc.wantDesc {
				t.Errorf("got desc %q; want %q", gotDesc, tc.wantDesc)
			}
		})
	}
}
