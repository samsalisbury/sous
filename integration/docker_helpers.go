// +build integration

package integration

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"html/template"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	sing "github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/ext/singularity"
	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/test_with_docker"
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

	testAgent, err := test_with_docker.NewAgentWithTimeout(5 * time.Minute)
	if err != nil {
		panic(err)
	}
	defer func() {
		testAgent.Cleanup()
	}()

	ip, err = testAgent.IP()
	if err != nil {
		panic(err)
	}
	if ip == nil {
		panic(fmt.Errorf("Test agent returned nil IP"))
	}

	registryName = fmt.Sprintf("%s:%d", ip, 5000)
	SingularityURL = fmt.Sprintf("http://%s:%d/singularity", ip, 7099)

	registryCerts(testAgent, composeDir)

	started, err := testAgent.ComposeServices(composeDir, map[string]uint{"Singularity": 7099, "Registry": 5000})
	defer testAgent.Shutdown(started)

	log.Print("   *** Beginning tests... ***\n\n")
	resultCode = m.Run()
	return
}

// ResetSingularity clears out the state from the integration singularity service
// Call it (with and extra call deferred) anywhere integration tests use Singularity
func ResetSingularity() {
	singClient := sing.NewClient(SingularityURL)

	reqList, err := singClient.GetRequests()
	if err != nil {
		panic(err)
	}

	for _, r := range reqList {
		_, err := singClient.DeleteRequest(r.Request.Id, nil)
		if err != nil {
			panic(err)
		}
	}
}

func registryCerts(testAgent test_with_docker.Agent, composeDir string) {
	registryCertName := "testing.crt"
	certPath := filepath.Join(composeDir, registryCertName)
	caPath := fmt.Sprintf("/etc/docker/certs.d/%s/ca.crt", registryName)

	certIPs, err := getCertIPSans(filepath.Join(composeDir, registryCertName))
	if err != nil {
		panic(err)
	}

	haveIP := false

	for _, certIP := range certIPs {
		if certIP.Equal(ip) {
			haveIP = true
			break
		}
	}

	if !haveIP {
		log.Printf("Rebuilding the registry certificate to add %v", ip)
		certIPs = append(certIPs, ip)
		err = buildTestingKeypair(composeDir, certIPs)
		if err != nil {
			panic(fmt.Errorf("While building testing keypair: %s", err))
		}

		err = testAgent.RebuildService(composeDir, "registry")
		if err != nil {
			panic(fmt.Errorf("While rebuilding the registry service: %s", err))
		}
	}

	differs, err := testAgent.DifferingFiles([]string{certPath, caPath})
	if err != nil {
		panic(fmt.Errorf("While checking for differing certs: %s", err))
	}

	for _, diff := range differs {
		local, remote := diff[0], diff[1]
		log.Printf("Copying %q to %q\n", local, remote)
		err = testAgent.InstallFile(local, remote)

		if err != nil {
			panic(fmt.Errorf("installFile failed: %s", err))
		}
	}

	if len(differs) > 0 {
		err = testAgent.RestartDaemon()
		if err != nil {
			panic(fmt.Errorf("restarting docker machine's daemon failed: %s", err))
		}
	}
	return
}

func buildTestingKeypair(dir string, certIPs []net.IP) (err error) {
	selfSignConf := "self-signed.conf"
	temp := template.Must(template.New("req").Parse(`
{{- "" -}}
[ req ]
prompt = no
distinguished_name=req_distinguished_name
x509_extensions = va_c3
encrypt_key = no
default_keyfile=testing.key
default_md = sha256

[ va_c3 ]
basicConstraints=critical,CA:true,pathlen:1
{{range . -}}
subjectAltName = IP:{{.}}
{{end}}
[ req_distinguished_name ]
CN=registry.test
{{"" -}}
		`))
	confPath := filepath.Join(dir, selfSignConf)
	reqFile, err := os.Create(confPath)
	if err != nil {
		return
	}
	err = temp.Execute(reqFile, certIPs)
	if err != nil {
		return
	}

	// This is the openssl command to produce a (very weak) self-signed keypair based on the conf templated above.
	// Ultimately, this provides the bare minimum to use the docker registry on a "remote" server
	openssl := exec.Command("openssl", "req", "-newkey", "rsa:512", "-x509", "-days", "365", "-out", "testing.crt", "-config", selfSignConf, "-batch")
	openssl.Dir = dir
	_, err = openssl.CombinedOutput()

	return
}

func getCertIPSans(certPath string) ([]net.IP, error) {
	certFile, err := os.Open(certPath)
	if _, ok := err.(*os.PathError); ok {
		return make([]net.IP, 0), nil
	}
	if err != nil {
		return nil, err
	}

	certBuf := bytes.Buffer{}
	_, err = certBuf.ReadFrom(certFile)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(certBuf.Bytes())

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return cert.IPAddresses, nil
}

// BuildImageName constructs a simple image name rooted at the SingularityURL
func BuildImageName(reponame, tag string) string {
	return fmt.Sprintf("%s/%s:%s", registryName, reponame, tag)
}

func registerAndDeploy(ip net.IP, clusterName, reponame, dir string, ports []int32) (err error) {
	imageName := BuildImageName(reponame, "latest")
	err = BuildAndPushContainer(dir, imageName)
	if err != nil {
		panic(fmt.Errorf("building test container failed: %s", err))
	}

	err = startInstance(SingularityURL, clusterName, imageName, reponame, ports)
	if err != nil {
		panic(fmt.Errorf("starting a singularity instance failed: %s", err))
	}

	return
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

func startInstance(url, clusterName, imageName, repoName string, ports []int32) error {
	did := sous.DeployID{
		ManifestID: sous.ManifestID{
			Source: sous.SourceLocation{
				Repo: repoName,
			},
		},
		Cluster: clusterName,
	}
	reqID := singularity.MakeRequestID(did)

	sing := sing.NewClient(url)

	req := loadMap(&dtos.SingularityRequest{}, map[string]interface{}{
		"Id":          reqID,
		"RequestType": dtos.SingularityRequestRequestTypeSERVICE,
		"Instances":   int32(1),
	}).(*dtos.SingularityRequest)

	for {
		_, err := sing.PostRequest(req)
		if err != nil {
			if rerr, ok := err.(*swaggering.ReqError); ok && rerr.Status == 409 { //not done deleting the request
				continue
			}

			return err
		}
		break
	}

	dockerInfo := loadMap(&dtos.SingularityDockerInfo{}, dtoMap{
		"Image": imageName,
	}).(*dtos.SingularityDockerInfo)

	depReq := loadMap(&dtos.SingularityDeployRequest{}, dtoMap{
		"Deploy": loadMap(&dtos.SingularityDeploy{}, dtoMap{
			"Id":        singularity.MakeDeployID(uuid.NewV4().String()),
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

	_, err := sing.Deploy(depReq)
	if err != nil {
		return err
	}

	return nil
}
