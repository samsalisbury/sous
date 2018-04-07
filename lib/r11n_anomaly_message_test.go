package sous

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/util/logging"
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

			// See also diff_rez_message_test.go which tests these same
			// fields including errors and deployment/manifest IDs.
			logging.AssertMessageFields(t, m, logging.StandardVariableFields, map[string]interface{}{
				"@loglov3-otl":                 logging.SousDiffResolution,
				"sous-resolution-errortype":    "",
				"sous-resolution-errormessage": "",
				"sous-deployment-id":           ":",
				"sous-manifest-id":             "",
				"sous-diff-source-type":        "global rectifier",
				"sous-diff-source-user":        "unknown",
				"sous-resolution-description":  string(tc.wantDesc),
			})
		})
	}
}
