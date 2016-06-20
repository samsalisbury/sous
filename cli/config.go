package cli

import (
	"fmt"
	"io/ioutil"
	"os"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/configloader"
	"github.com/opentable/sous/util/whitespace"
	"github.com/opentable/sous/util/yaml"
)

func newConfig(u *User) (*sous.Config, error) {
	config := u.DefaultConfig()

	err := os.MkdirAll(u.ConfigDir(), os.ModeDir|0755)
	if err != nil {
		return nil, err
	}

	return &config, configloader.New().Load(&config, u.ConfigFile())
}

// Save the configuration to the configuration path (by default:
// $HOME/.config/sous/config)
func (c *LocalSousConfig) Save(u *User) error {
	return ioutil.WriteFile(u.ConfigFile(), c.Bytes(), 0600)
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
func (c *LocalSousConfig) SetValue(user *User, name, value string) error {
	if err := configloader.New().SetValue(c.Config, name, value); err != nil {
		return err
	}
	return c.Save(user)
}

func (c *LocalSousConfig) String() string {
	// yaml marshaller adds a trailing newline
	return whitespace.Trim(string(c.Bytes()))
}
