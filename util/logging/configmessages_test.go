package logging

import (
	"net"
	"testing"
	"time"

	graphite "github.com/cyberdelia/go-metrics-graphite"
	"github.com/opentable/sous/util/logging/constants"
)

func TestGraphiteConfigMessage(t *testing.T) {
	msg := graphiteConfigMessage{
		CallerInfo: GetCallerInfo(),
		cfg: &graphite.Config{
			Addr: &net.TCPAddr{
				IP:   net.IP{169, 169, 13, 13},
				Port: 3636,
				Zone: "",
			},
			FlushInterval: 30 * time.Second,
		},
	}

	AssertMessageFields(t, msg, StandardVariableFields, map[string]interface{}{
		"graphite-flush-interval":    int64(30000000),
		"graphite-server-address":    "169.169.13.13:3636",
		"@loglov3-otl":               constants.SousGraphiteConfigV1,
		"sous-successful-connection": true,
	})

}
