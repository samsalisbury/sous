// +build integration

package integration

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"testing"
	"time"

	sing "github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	"github.com/opentable/sous/ext/singularity"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/swaggering"
	"github.com/satori/go.uuid"
)

var ip net.IP
var registryName string

// SingularityURL captures the URL discovered during docker-compose for Singularity
var SingularityURL string

var successfulBuildRE = regexp.MustCompile(`Successfully built (\w+)`)

// WrapCompose is used to set up the docker/singularity testing environment.
// Use like this:
//  func TestMain(m *testing.M) {
//  	flag.Parse()
//  	os.Exit(WrapCompose(m))
//  }
// Importantly, WrapCompose handles panics so that defers will still happen
// (including shutting down singularity)
func WrapCompose(m *testing.M, composeDir string) (resultCode int) {
	if testing.Short() {
		return 0
	}

	defer func() {
		if err := recover(); err != nil {
			log.Print("Panic: ", err)
			resultCode = 1
		}
	}()

	descPath := os.Getenv("SOUS_QA_DESC")

	var envDesc desc.EnvDesc

	if descPath == "" {
		panic("SOUS_QA_DESC is unset! Integration tests now require a description file generated by sous_qa_setup.")
	}
	descReader, err := os.Open(descPath)
	if err != nil {
		panic(err)
	}
	dec := json.NewDecoder(descReader)
	err = dec.Decode(&envDesc)
	if err != nil {
		panic(err)
	}

	ip = envDesc.AgentIP
	registryName = envDesc.RegistryName()
	SingularityURL = envDesc.SingularityURL()

	log.Print("   *** Beginning tests... ***\n\n")
	resultCode = m.Run()
	return
}

// ResetSingularity clears out the state from the integration singularity service
// Call it (with and extra call deferred) anywhere integration tests use Singularity
func ResetSingularity() {
	const pollLimit = 30
	const retryLimit = 3
	log.Print("Resetting Singularity...")
	singClient := sing.NewClient(SingularityURL)

	reqList, err := singClient.GetRequests(false)
	if err != nil {
		panic(err)
	}

	// Singularity is sometimes not actually deleting a request until the second attempt...
	for j := retryLimit; j >= 0; j-- {
		for _, r := range reqList {
			_, err := singClient.DeleteRequest(r.Request.Id, nil)
			if err != nil {
				panic(err)
			}
		}

		log.Printf("Singularity resetting: Issued deletes for %d requests. Awaiting confirmation they've quit.", len(reqList))

		for i := pollLimit; i > 0; i-- {
			reqList, err = singClient.GetRequests(false)
			if err != nil {
				panic(err)
			}
			if len(reqList) == 0 {
				log.Printf("Singularity successfully reset.")
				return
			}
			time.Sleep(time.Second)
		}
	}
	for n, req := range reqList {
		log.Printf("Singularity reset failure: stubborn request: #%d/%d %#v", n+1, len(reqList), req)
	}
	panic(fmt.Errorf("singularity not reset after %d * %d tries - %d requests remain", retryLimit, pollLimit, len(reqList)))
}

// BuildImageName constructs a simple image name rooted at the SingularityURL
func BuildImageName(reponame, tag string) string {
	return fmt.Sprintf("%s/%s:%s", registryName, reponame, tag)
}

func registerAndDeploy(ip net.IP, clusterName, reponame, sourceRepo, dir, tag string, ports []int32, startup sous.Startup) error {
	imageName := BuildImageName(reponame, tag)
	if err := BuildAndPushContainer(dir, imageName); err != nil {
		panic(fmt.Errorf("building test container failed: %s", err))
	}
	if err := startInstance(SingularityURL, clusterName, imageName, sourceRepo, ports, startup); err != nil {
		panic(fmt.Errorf("starting a singularity instance failed: %s", err))
	}

	return nil
}

// BuildAndPushContainer builds a container based on the source found in
// containerDir, and then pushes it to the integration docker registration
// under tagName
func BuildAndPushContainer(containerDir, tagName string) error {
	build := exec.Command("docker", "build", ".")
	build.Dir = containerDir
	output, err := build.CombinedOutput()
	if err != nil {
		log.Print("Problem building container: ", containerDir, "\n", string(output))
		log.Print(err)
		return err
	}

	match := successfulBuildRE.FindStringSubmatch(string(output))
	if match == nil {
		return fmt.Errorf("Couldn't find container id in:\n%s", output)
	}

	containerID := match[1]
	tag := exec.Command("docker", "tag", containerID, tagName)
	tag.Dir = containerDir
	output, err = tag.CombinedOutput()
	if err != nil {
		log.Print("Problem tagging container: ", containerDir, "\n", string(output))
		return err
	}

	push := exec.Command("docker", "push", tagName)
	push.Dir = containerDir
	output, err = push.CombinedOutput()
	if err != nil {
		log.Print("Problem pushing container: ", containerDir, "\n", string(output))
		return err
	}

	return nil
}

type dtoMap map[string]interface{}

func loadMap(fielder swaggering.Fielder, m dtoMap) swaggering.Fielder {
	_, err := swaggering.LoadMap(fielder, m)
	if err != nil {
		log.Fatal(err)
	}

	return fielder
}

var notInIDre = regexp.MustCompile(`[-/]`)

func startInstance(url, clusterName, imageName, repoName string, ports []int32, startup sous.Startup) error {
	did := sous.DeploymentID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: repoName,
			},
		},
		Cluster: clusterName,
	}
	log.Printf("%#v", did)
	reqID, err := singularity.MakeRequestID(did)
	if err != nil {
		return err
	}
	sing := sing.NewClient(url)

	req := loadMap(&dtos.SingularityRequest{}, map[string]interface{}{
		"Id":          reqID,
		"RequestType": dtos.SingularityRequestRequestTypeSERVICE,
		"Instances":   int32(1),
	}).(*dtos.SingularityRequest)

	for {
		log.Printf("Creating request %q", reqID)
		_, err := sing.PostRequest(req)
		if err != nil {
			log.Printf("PostRequest error:%#v", err)
			if rerr, ok := err.(*swaggering.ReqError); ok && rerr.Status == 409 { //not done deleting the request
				time.Sleep(time.Second)
				continue
			}

			return err
		}
		break
	}

	dockerInfo := loadMap(&dtos.SingularityDockerInfo{}, dtoMap{
		"Image": imageName,
	}).(*dtos.SingularityDockerInfo)

	deployID := "TESTGENERATED_" + singularity.StripDeployID(uuid.NewV4().String())
	depMap := dtoMap{
		"Metadata": map[string]string{
			"com.opentable.sous.clustername": clusterName,
		},
		"Id":        deployID,
		"RequestId": reqID,
		"Resources": loadMap(&dtos.Resources{}, dtoMap{
			"Cpus":     0.1,
			"MemoryMb": 100.0,
			"NumPorts": int32(1),
		}),
		"ContainerInfo": loadMap(&dtos.SingularityContainerInfo{}, dtoMap{
			"Type":   dtos.SingularityContainerInfoSingularityContainerTypeDOCKER,
			"Docker": dockerInfo,
		}),
	}

	err = singularity.MapStartupIntoHealthcheckOptions((*map[string]interface{})(&depMap), startup)
	if err != nil {
		return err
	}

	depReqMap := dtoMap{
		"Deploy": loadMap(&dtos.SingularityDeploy{}, depMap),
	}

	depReq := loadMap(&dtos.SingularityDeployRequest{}, depReqMap).(*dtos.SingularityDeployRequest)

	log.Printf("Constructed SingularityDeployRequest %#v", depReq)
	log.Printf("  containing SingularityDeploy %#v", *depReq.Deploy)

	_, err = sing.Deploy(depReq)
	if err != nil {
		return err
	}
	log.Printf("Started singularity deploy %q at request %q", deployID, reqID)

	return nil
}
