package cli

import (
	"fmt"
	"io/ioutil"

	sous "github.com/opentable/sous/lib"
	"github.com/opentable/sous/util/configloader"
	"github.com/opentable/sous/util/whitespace"
	"github.com/opentable/sous/util/yaml"
)

func newDefaultConfig(u *User) (*sous.Config, error) {
	var config sous.Config
	return &config, configloader.New().Load(&config, u.ConfigFile())
}

func (c *LocalSousConfig) Save(u *User) error {
	return ioutil.WriteFile(u.ConfigFile(), c.Bytes(), 0600)
}

func (c *LocalSousConfig) Bytes() []byte {
	y, err := yaml.Marshal(c.Config)
	if err != nil {
		panic("error marshalling config as yaml:" + err.Error())
	}
	return y
}

func (c *LocalSousConfig) GetValue(name string) (string, error) {
	v, err := configloader.New().GetValue(c.Config, name)
	return fmt.Sprint(v), err
}

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
