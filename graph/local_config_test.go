package graph

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/util/configloader"
	"github.com/opentable/sous/util/logging"
)

func remove(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func TestNewConfig(t *testing.T) {
	path := "./testdata/testconfig.yaml"
	if err := remove(path); err != nil {
		t.Fatal("Test setup failed to remove file:", err)
	}
	defer func() {
		if err := remove(path); err != nil {
			t.Fatal("Test cleanup failed to remove file:", err)
		}
	}()

	gcl := newConfigLoader(silentLogSink)

	written, err := newPossiblyInvalidConfig(silentLogSink, path, DefaultConfig{&config.Config{}}, gcl)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ioutil.ReadFile(path); err != nil {
		t.Fatal("Config file not created:", path, ":", err)
	}

	read, err := newPossiblyInvalidConfig(silentLogSink, path, DefaultConfig{&config.Config{}}, gcl)
	if err != nil {
		t.Fatal(err)
	}

	if !read.Config.Equal(written.Config) {
		t.Log("READ:\n\n", read)
		t.Log("WRITTEN:\n\n", written)
		t.Error("Read and written configs were different.")
	}
}

func TestLoadConfig(t *testing.T) {
	path := "./testdata/config.yaml"

	cl := configloader.New(logging.SilentLogSet())
	config := config.Config{}
	err := cl.Load(&config, path)

	if err != nil {
		t.Fatalf("Err loading config: %s", err)
	}

	if len(config.SiblingURLs["ci-sf"]) == 0 {
		t.Error("Empty URL for ci-sf")
	}
}
