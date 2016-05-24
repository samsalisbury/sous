package test

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/opentable/singularity"
	"github.com/opentable/singularity/dtos"
	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/test_with_docker"
	"github.com/opentable/sous/util/docker_registry"
	"github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

var ip, registryName, imageName, singularityURL string

func TestGetLabels(t *testing.T) {
	assert := assert.New(t)
	cl := docker_registry.NewClient()
	cl.BecomeFoolishlyTrusting()

	labels, err := cl.LabelsForImageName(imageName)

	assert.Nil(err)
	assert.Contains(labels, sous.DockerRepoLabel)
}

func TestGetRunningDeploymentSet(t *testing.T) {
	assert := assert.New(t)

	deps, err := sous.GetRunningDeploymentSet([]string{singularityURL})
	assert.Nil(err)
	assert.Equal(3, len(deps))
	var grafana *sous.Deployment
	for i := range deps {
		if deps[i].SourceVersion.RepoURL == "https://github.com/opentable/docker-grafana.git" {
			grafana = deps[i]
		}
	}
	if !assert.NotNil(grafana) {
		assert.FailNow("If deployment is nil, other tests will crash")
	}
	assert.Equal(singularityURL, grafana.Cluster)
	assert.Regexp("^0\\.1", grafana.Resources["cpus"])    // XXX strings and floats...
	assert.Regexp("^100\\.", grafana.Resources["memory"]) // XXX strings and floats...
	assert.Equal("1", grafana.Resources["ports"])         // XXX strings and floats...
	assert.Equal(17, grafana.SourceVersion.Patch)
	assert.Equal("91495f1b1630084e301241100ecf2e775f6b672c", grafana.SourceVersion.Meta)
	assert.Equal(1, grafana.NumInstances)
	assert.Equal(sous.ManifestKindService, grafana.Kind)
}

func TestMain(m *testing.M) {
	flag.Parse()
	os.Exit(wrapCompose(m))
}

func wrapCompose(m *testing.M) (resultCode int) {
	log.SetFlags(log.Flags() | log.Lshortfile)

	if testing.Short() {
		return 0
	}

	defer func() {
		log.Println("Cleaning up...")
		if err := recover(); err != nil {
			log.Print("Panic: ", err)
			resultCode = 1
		}
	}()

	testAgent, err := test_with_docker.NewAgentWithTimeout(5 * time.Minute)
	if err != nil {
		panic(err)
	}

	ip, err := testAgent.IP()
	if err != nil {
		panic(err)
	}

	composeDir := "test-registry"
	registryCertName := "testing.crt"
	registryName = fmt.Sprintf("%s:%d", ip, 5000)

	err = registryCerts(testAgent, composeDir, registryCertName, ip)
	if err != nil {
		panic(fmt.Errorf("building registry certs failed: %s", err))
	}

	started, err := testAgent.ComposeServices(composeDir, map[string]uint{"Singularity": 7099, "Registry": 5000})
	defer testAgent.Shutdown(started)

	registerAndDeploy(ip, "hello-labels", "hello-labels", []int32{})
	registerAndDeploy(ip, "hello-server-labels", "hello-server-labels", []int32{80})
	registerAndDeploy(ip, "grafana-repo", "grafana-labels", []int32{})
	imageName = fmt.Sprintf("%s/%s:%s", registryName, "grafana-repo", "latest")

	log.Print("   *** Beginning tests... ***\n\n")
	resultCode = m.Run()
	return
}

func registerAndDeploy(ip net.IP, reponame, dir string, ports []int32) (err error) {
	imageName := fmt.Sprintf("%s/%s:%s", registryName, reponame, "latest")
	err = buildAndPushContainer(dir, imageName)
	if err != nil {
		panic(fmt.Errorf("building test container failed: %s", err))
	}

	singularityURL = fmt.Sprintf("http://%s:%d/singularity", ip, 7099)
	err = startInstance(singularityURL, imageName, ports)
	if err != nil {
		panic(fmt.Errorf("starting a singularity instance failed: %s", err))
	}

	return
}

type dtoMap map[string]interface{}

func loadMap(fielder dtos.Fielder, m dtoMap) dtos.Fielder {
	_, err := dtos.LoadMap(fielder, m)
	if err != nil {
		log.Fatal(err)
	}

	return fielder
}

var notInIDre = regexp.MustCompile(`[-/]`)

func idify(in string) string {
	return notInIDre.ReplaceAllString(in, "")
}

func startInstance(url, imageName string, ports []int32) error {
	reqID := idify(imageName)

	sing := singularity.NewClient(url)

	req := loadMap(&dtos.SingularityRequest{}, map[string]interface{}{
		"Id":          reqID,
		"RequestType": dtos.SingularityRequestRequestTypeSERVICE,
		"Instances":   int32(1),
	}).(*dtos.SingularityRequest)

	_, err := sing.PostRequest(req)
	if err != nil {
		return err
	}

	dockerInfo := loadMap(&dtos.SingularityDockerInfo{}, dtoMap{
		"Image": imageName,
	}).(*dtos.SingularityDockerInfo)

	depReq := loadMap(&dtos.SingularityDeployRequest{}, dtoMap{
		"Deploy": loadMap(&dtos.SingularityDeploy{}, dtoMap{
			"Id":        idify(uuid.NewV4().String()),
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
		}),
	}).(*dtos.SingularityDeployRequest)

	_, err = sing.Deploy(depReq)
	if err != nil {
		return err
	}

	return nil
}
