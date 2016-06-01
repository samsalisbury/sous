package main

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"

	"github.com/docopt/docopt-go"
)

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	parsed, err := docopt.Parse(`Usage: cert_san <cert-path>`, nil, true, "", false)

	if err != nil {
		log.Fatal(err)
	}

	certPath := parsed["<cert-path>"].(string)

	certFile, err := os.Open(certPath)
	if err != nil {
		log.Fatal(err)
	}

	certBuf := bytes.Buffer{}
	_, err = certBuf.ReadFrom(certFile)
	if err != nil {
		log.Fatal(err)
	}

	log.Print("\n", certBuf.String())
	block, _ := pem.Decode(certBuf.Bytes())

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Cert good for addresses: %+v", cert.IPAddresses)
}
