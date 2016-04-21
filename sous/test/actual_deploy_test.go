package test

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"text/template"

	"github.com/opentable/test_with_docker"
)

var ip string

func TestMain(m *testing.M) {
	os.Exit(wrapCompose(m))
}

func wrapCompose(m *testing.M) int {
	log.SetFlags(log.Flags() | log.Lshortfile)

	machineName := "default"
	composeDir := "test/test-registry"

	err := registryCerts(composeDir, machineName)
	if err != nil {
		log.Fatal(err)
	}

	ipstr, started, err := test_with_docker.ComposeServices("default", composeDir, map[string]uint{"Singularity": 7099, "Registry": 5000})
	ip = ipstr

	defer test_with_docker.Shutdown(started)

	if err != nil {
		log.Panic(err)
	}

	return m.Run()
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
		return err
	}

	// This is the openssl command to produce a (very weak) self-signed keypair based on the conf templated above.
	// Ultimately, this provides the bare minimum to use the docker registry on a "remote" server
	openssl := exec.Command("openssl", "req", "-newkey", "rsa:512", "-x509", "-days", "365", "-out", "testing.crt", "-config", selfSignConf, "-batch")
	openssl.Dir = dir
	out, err := openssl.CombinedOutput()
	log.Print(string(out))

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
		log.Fatal(err)
	}

	block, _ := pem.Decode(certBuf.Bytes())

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}

	return cert.IPAddresses, nil
}

// registryCerts makes sure that we'll be able to reach the test registry
// Find the docker-machine IP
// Get the SAN from the existing test cert
//   If different, template out a new openssl config
//   Regenerate the key/cert pair
//   docker-compose rebuild the registry service
func registryCerts(composeDir string, machineName string) error {
	ipstr, err := test_with_docker.MachineIP(machineName)
	if err != nil {
		return err
	}
	ip := net.ParseIP(ipstr)

	certIPs, err := getCertIPSans(filepath.Join(composeDir, "testing.crt"))
	if err != nil {
		return err
	}

	haveIP := false

	for _, certIP := range certIPs {
		if certIP.Equal(ip) {
			log.Print("Registry cert includes registry IP")
			haveIP = true
			break
		}
	}

	if !haveIP {
		log.Print("Rebuilding the registry certificate")
		certIPs = append(certIPs, ip)
		err = buildTestingKeypair(composeDir, certIPs)
		if err != nil {
			return err
		}

		err = test_with_docker.RebuildService(machineName, composeDir, "registry")
		if err != nil {
			return err
		}
	}
	return err
}
