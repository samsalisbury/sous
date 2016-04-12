package cli

import (
	"fmt"
	"io/ioutil"

	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/configloader"
	"github.com/opentable/sous/util/yaml"
)

func newDefaultConfig(u *User) (*sous.Config, error) {
	var config sous.Config
	return &config, configloader.New().Load(&config, u.ConfigFile())
}

func (c *LocalSousConfig) Save(u *User) error {
	return ioutil.WriteFile(u.ConfigFile(), c.Bytes(), 600)
}

func (c *LocalSousConfig) Bytes() []byte {
	y, err := yaml.Marshal(c)
	if err != nil {
		panic("error marshalling config as yaml:" + err.Error())
	}
	return y
}

func (c *LocalSousConfig) GetValue(name string) (string, error) {
	v, err := configloader.New().GetFieldValue(c.Config, name)
	return fmt.Sprint(v), err
}

func (c *LocalSousConfig) String() string {
	return string(c.Bytes())
}
