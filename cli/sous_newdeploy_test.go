package cli

import (
	"fmt"
	"testing"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/restful"
	"github.com/stretchr/testify/assert"
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

func Test_PollDeployQueueBrokenURL(t *testing.T) {
	brokenLocation := "http://never-going2.wrk/test"

	result := PollDeployQueue(brokenLocation, client, log)
	fmt.Printf("result : %v", result)
	assert.Equal(t, 70, result.ExitCode(), "should map to internal error")
}

func Test_PollDeployQueueLocalLocation(t *testing.T) {
	result := PollDeployQueue(location, client, log)
	fmt.Printf("result : %v \n", result)
	assert.Equal(t, 0, result.ExitCode(), "should return success")
}
