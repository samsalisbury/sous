package graph

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/opentable/sous/config"
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

	written, err := newPossiblyInvalidConfig(path, DefaultConfig{&config.Config{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ioutil.ReadFile(path); err != nil {
		t.Fatal("Config file not created:", path, ":", err)
	}

	read, err := newPossiblyInvalidConfig(path, DefaultConfig{&config.Config{}})
	if err != nil {
		t.Fatal(err)
	}

	if *read.Config != *written.Config {
		t.Log("READ:\n\n", read)
		t.Log("WRITTEN:\n\n", written)
		t.Error("Read and written configs were different.")
	}
}
