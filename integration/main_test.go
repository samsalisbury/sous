// +build integration
package integration

import (
	"flag"
	"log"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	log.SetFlags(log.Flags() | log.Lshortfile)
	flag.Parse()
	os.Exit(WrapCompose(m, "./test-registry"))
}
