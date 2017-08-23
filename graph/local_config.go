package graph

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh/terminal"

	"github.com/davecgh/go-spew/spew"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/util/configloader"
	"github.com/opentable/sous/util/logging"
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

func newSousConfig(lsc LocalSousConfig) *config.Config {
	return lsc.Config
}

func newPossiblyInvalidLocalSousConfig(u config.LocalUser, defaultConfig DefaultConfig, gcl *ConfigLoader) (PossiblyInvalidConfig, error) {
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
	spew.Dump("live config loader")
	cl := configloader.New()
	logging.SetupLogging(cl)
	return &ConfigLoader{ConfigLoader: cl}
}

func newPossiblyInvalidConfig(path string, defaultConfig DefaultConfig, gcl *ConfigLoader) (PossiblyInvalidConfig, error) {
	spew.Dump(gcl)
	cl := gcl.ConfigLoader

	pic := defaultConfig

	configDir := filepath.Dir(path)
	if err := os.MkdirAll(configDir, os.ModeDir|0755); err != nil {
		return PossiblyInvalidConfig{}, err
	}

	var writeDefault bool
	defer func() {
		if !writeDefault {
			return
		}
		// Since this is initialisation, let's get the user to confirm some stuff...
		userInitConfig(pic.Config)
		if err := pic.Validate(); err != nil {
			// If the config is invalid, warn but write it anyway and allow the
			// user to correct it themselves.
			logging.Log.Warn.Printf("Newly initialised config file is invalid: %s", err)
			logging.Log.Warn.Printf("Please correct the issue by editing %s", path)
		}
		lsc := &LocalSousConfig{pic.Config}
		lsc.Save(path)
		logging.Log.Info.Println("initialised config file: " + path)
	}()
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		err = nil
		writeDefault = true
	}
	if err != nil {
		return PossiblyInvalidConfig{}, err
	}

	return PossiblyInvalidConfig{Config: pic.Config}, cl.Load(pic.Config, path)
}

func userInput(prompt, vDefault, eg string, v *string) {
	if vDefault == "" {
		fmt.Printf("%s (e.g. %q): ", prompt, eg)
	} else {
		fmt.Printf("%s (e.g. %q): (enter for %q)", prompt, eg, vDefault)
	}
	reader := bufio.NewReader(os.Stdin)
	in, err := reader.ReadString('\n')
	if err != nil {
		logging.Log.Warn.Printf("Failed to read input: %s", err)
		return
	}
	// Strip the newline and any other whitespace.
	in = strings.TrimSpace(in)
	if in == "" {
		in = vDefault
	}
	*v = in
}

func userInitConfig(c *config.Config) {
	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		logging.Log.Warn.Println("Unable to run interactive configuration; stdout isn't attached to a terminal.")
		return
	}
	if !terminal.IsTerminal(int(os.Stdin.Fd())) {
		logging.Log.Warn.Println("Unable to run interactive configuration; stdin isn't attached to a terminall.")
		return
	}
	if os.Getenv("TASK_HOST") != "" { // XXX This is terrible, but the terminal check fails (which breaks the Mesos servers)
		logging.Log.Warn.Println("Refusing to run interactive configuration; TASK_HOST is set.")
		return
	}
	fmt.Println(`
	It looks like you haven't used Sous before (there's no config file).
	Please provide the following details to configure Sous...
	If you don't know some of the answers don't worry, you can use 'sous config'
	on the command line to change them later.
	`)
	userInput("Your email address", c.User.Email, "name@mycompany.com", &c.User.Email)
	userInput("Your full name", c.User.Name, "Alfie Noakes", &c.User.Name)
	userInput("Your company's primary sous server URL", c.Server, "http://sous.mycompany.com", &c.Server)

	fmt.Println("All done, thanks!")
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
