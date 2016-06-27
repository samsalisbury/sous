package test

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

	"github.com/opentable/go-singularity"
	"github.com/opentable/go-singularity/dtos"
	"github.com/opentable/sous/util/test_with_docker"
	"github.com/satori/go.uuid"
)

var ip net.IP
var registryName, singularityURL string

var successfulBuildRE = regexp.MustCompile(`Successfully built (\w+)`)

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

	ip, err = testAgent.IP()
	if err != nil {
		panic(err)
	}
	if ip == nil {
		panic(fmt.Errorf("Test agent returned nil IP"))
	}

	composeDir := "test-registry"
	registryName = fmt.Sprintf("%s:%d", ip, 5000)
	singularityURL = fmt.Sprintf("http://%s:%d/singularity", ip, 7099)

	registryCerts(testAgent, composeDir)

	started, err := testAgent.ComposeServices(composeDir, map[string]uint{"Singularity": 7099, "Registry": 5000})
	defer testAgent.Shutdown(started)

	log.Print("   *** Beginning tests... ***\n\n")
	resultCode = m.Run()
	return
}

func resetSingularity() {
	sing := singularity.NewClient(singularityURL)
	//sing.Debug = true

	reqList, err := sing.GetRequests()
	if err != nil {
		panic(err)
	}

	for _, r := range reqList {
		_, err := sing.DeleteRequest(r.Request.Id, nil)
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

func buildImageName(reponame, tag string) string {
	return fmt.Sprintf("%s/%s:%s", registryName, reponame, tag)
}

func registerAndDeploy(ip net.IP, reponame, dir string, ports []int32) (err error) {
	imageName := buildImageName(reponame, "latest")
	err = buildAndPushContainer(dir, imageName)
	if err != nil {
		panic(fmt.Errorf("building test container failed: %s", err))
	}

	err = startInstance(singularityURL, imageName, ports)
	if err != nil {
		panic(fmt.Errorf("starting a singularity instance failed: %s", err))
	}

	return
}

func buildAndPushContainer(containerDir, tagName string) error {
	build := exec.Command("docker", "build", ".")
	build.Dir = containerDir
	output, err := build.CombinedOutput()
	if err != nil {
		log.Print("Problem building container: ", containerDir, "\n", string(output))
		return err
	}

	match := successfulBuildRE.FindStringSubmatch(string(output))
	if match == nil {
		return fmt.Errorf("Couldn't find container id in:\n%s", output)
	}

	containerID := match[1]
	tag := exec.Command("docker", "tag", "-f", containerID, tagName)
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

	for {
		_, err := sing.PostRequest(req)
		if err != nil {
			if rerr, ok := err.(*singularity.ReqError); ok && rerr.Status == 409 { //not done deleting the request
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

	_, err := sing.Deploy(depReq)
	if err != nil {
		return err
	}

	return nil
}
