var ip, registryName, singularityURL string
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

	"github.com/opentable/test_with_docker"
)

var successfulBuildRE = regexp.MustCompile(`Successfully built (\w+)`)

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
	tag := exec.Command("docker", "tag", containerID, tagName)
	tag.Dir = containerDir
	output, err = tag.CombinedOutput()
	if err != nil {
		return err
	}

	push := exec.Command("docker", "push", tagName)
	push.Dir = containerDir
	output, err = push.CombinedOutput()
	if err != nil {
		return err
	}

	return nil
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
		log.Print("Rebuilding the registry certificate")
		certIPs = append(certIPs, ip)
		err = buildTestingKeypair(composeDir, certIPs)
		if err != nil {
			panic(err)
		}

		err = testAgent.RebuildService(composeDir, "registry")
		if err != nil {
			panic(err)
		}
	}

	differs, err := testAgent.DifferingFiles([]string{certPath, caPath})
	if err != nil {
		panic(err)
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
	log.Print(certIPs)
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
