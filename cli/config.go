package cli

import (
	"github.com/opentable/sous/sous"
	"github.com/opentable/sous/util/configloader"
)

func newDefaultConfig(u *User) (*sous.Config, error) {
	var config sous.Config
	return &config, configloader.New().Load(&config, u.ConfigDir())
}
