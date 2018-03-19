package cli

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
)

var log, _ = logging.NewLogSinkSpy()
var client, _ = restful.NewClient("", log, nil)
var location = "127.0.0.1:8888/deploy-queue-item?action=bb836990-5ab2-4eab-9f52-ad3fd555539b&cluster=dev-ci-sf&flavor=&offset=&repo=github.com%2Fopentable%2Fsous-demo"

func TestDeployQueueItem(t *testing.T) {
	response := SingleDeployResponse{}

	location = "http://" + location

	updater, err := client.Retrieve(location, nil, &response, nil)
	fmt.Println("err : ", err)
	fmt.Println("updater : ", updater)
	fmt.Println("response : ", response)

}

func Test_PollDeployQueue(t *testing.T) {
	location = "http://" + location
	result := PollDeployQueue(location, log)
	assert(t, result.ExitCode, 1)
}
