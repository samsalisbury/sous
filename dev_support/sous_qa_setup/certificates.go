package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/nyarly/inlinefiles/templatestore"
	"github.com/opentable/sous/dev_support/sous_qa_setup/desc"
	"github.com/opentable/sous/util/test_with_docker"
)

//go:generate inlinefiles --vfs=Templates --package main templates vfs_template.go

func registryCerts(testAgent test_with_docker.Agent, composeDir string, desc desc.EnvDesc) error {
	registryCertName := "testing.crt"
	registryCertPath := filepath.Join(composeDir, "docker-registry", registryCertName)

	if err := generateRegistryCert(composeDir, desc.AgentIP, registryCertPath, testAgent); err != nil {
		return err
	}

	return testAgent.InstallRegistryCertificate(desc.RegistryName, composeDir, registryCertPath)
}

func generateRegistryCert(composeDir string, agentIP net.IP, registryCertPath string, testAgent test_with_docker.Agent) error {
	certIPs, err := getCertIPSans(registryCertPath)
	if err != nil {
		return err
	}

	haveIP := false

	for _, certIP := range certIPs {
		if certIP.Equal(agentIP) {
			haveIP = true
			break
		}
	}

	if !haveIP {
		log.Printf("Rebuilding the registry certificate to add %v", agentIP)
		certIPs = append(certIPs, agentIP)
		err = buildTestingKeypair(filepath.Dir(registryCertPath), certIPs)
		if err != nil {
			return fmt.Errorf("While building testing keypair: %s", err)
		}

		err = testAgent.RebuildService(composeDir, "registry")
		if err != nil {
			return fmt.Errorf("While rebuilding the registry service: %s", err)
		}
	}
	return nil
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

func buildTestingKeypair(dir string, certIPs []net.IP) (err error) {
	selfSignConf := "self-signed.conf"
	temp, err := templatestore.LoadText(Templates, "", "ssl-config.tmpl")
	if err != nil {
		return
	}

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

	return err
}
