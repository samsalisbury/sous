package graph

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/opentable/sous/config"
	"github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/configloader"
	"github.com/opentable/sous/util/whitespace"
	"github.com/opentable/sous/util/yaml"
)

// PossiblyInvalidConfig is a config that has not been validated.
// This is necessary for the 'sous config' command that should still work with
// invalid configs.
type PossiblyInvalidConfig struct{ *config.Config }

// DefaultConfig is the default config.
type DefaultConfig struct{ *config.Config }

func newPossiblyInvalidConfig(path string, defaultConfig DefaultConfig) (PossiblyInvalidConfig, error) {
	config := defaultConfig

	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, os.ModeDir|0755); err != nil {
		return PossiblyInvalidConfig{}, err
	}

	cl := configloader.New()
	sous.SetupLogging(&cl)
	var writeDefault bool
	defer func() {
		if !writeDefault {
			return
		}
		lsc := &LocalSousConfig{config.Config}
		lsc.Save(path)
		sous.Log.Info.Println("initialised config file: " + path)
	}()
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = nil
		writeDefault = true
	}
	if err != nil {
		return PossiblyInvalidConfig{}, err
	}

	return PossiblyInvalidConfig{Config: config.Config}, cl.Load(config.Config, path)
}

// Save the configuration to the configuration path (by default:
// $HOME/.config/sous/config)
func (c *LocalSousConfig) Save(path string) error {
	return ioutil.WriteFile(path, c.Bytes(), 0600)
}

// Bytes marshals the config to a []byte
func (c *LocalSousConfig) Bytes() []byte {
	y, err := yaml.Marshal(c.Config)
	if err != nil {
		panic("error marshalling config as yaml:" + err.Error())
	}
	return y
}

// GetValue retreives and returns a particular value from the configuration
func (c *LocalSousConfig) GetValue(name string) (string, error) {
	v, err := configloader.New().GetValue(c.Config, name)
	return fmt.Sprint(v), err
}

// SetValue stores a particular value on the config
func (c *LocalSousConfig) SetValue(path, name, value string) error {
	if err := configloader.New().SetValue(c.Config, name, value); err != nil {
		return err
	}
	return c.Save(path)
}

func (c *LocalSousConfig) String() string {
	// yaml marshaller adds a trailing newline
	return whitespace.Trim(string(c.Bytes()))
}
