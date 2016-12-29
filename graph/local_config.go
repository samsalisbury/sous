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
	"github.com/pkg/errors"
)

type (
	// PossiblyInvalidConfig is a config that has not been validated.
	// This is necessary for the 'sous config' command that should still work with
	// invalid configs.
	PossiblyInvalidConfig struct{ *config.Config }

	// DefaultConfig is the default config.
	DefaultConfig struct{ *config.Config }

	// ConfigLoader wraps the configloader.ConfigLoader interface
	ConfigLoader struct{ configloader.ConfigLoader }
)

func newPossiblyInvalidLocalSousConfig(u LocalUser, defaultConfig DefaultConfig, gcl *ConfigLoader) (PossiblyInvalidConfig, error) {
	v, err := newPossiblyInvalidConfig(u.ConfigFile(), defaultConfig, gcl)
	return v, initErr(err, "getting configuration")
}

func newLocalSousConfig(pic PossiblyInvalidConfig) (v LocalSousConfig, err error) {
	v.Config, err = pic.Config, pic.Validate()
	if err != nil {
		err = errors.Wrapf(err, "tip: run 'sous config' to see and manipulate your configuration")
	}
	return v, initErr(err, "validating configuration")
}

func newConfigLoader() *ConfigLoader {
	cl := configloader.New()
	sous.SetupLogging(cl)
	return &ConfigLoader{ConfigLoader: cl}
}

func newPossiblyInvalidConfig(path string, defaultConfig DefaultConfig, gcl *ConfigLoader) (PossiblyInvalidConfig, error) {
	cl := gcl.ConfigLoader

	config := defaultConfig

	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, os.ModeDir|0755); err != nil {
		return PossiblyInvalidConfig{}, err
	}

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
