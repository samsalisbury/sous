package cli

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/opentable/sous/lib"
)

type (
	// User represents the user environment of the account running Sous
	User struct {
		*user.User
	}
)

const (
	defaultConfigDir = "sous"
	xdgConfigDefault = ".config"
	configFileBase   = "config.yaml"
)

// DefaultConfig builds a default configuration for this user
func (u *User) DefaultConfig() sous.Config {
	c := sous.DefaultConfig()
	c.DatabaseConnection = filepath.Join(u.ConfigDir(), "data.db")
	return c
}

// ConfigDir returns the directory we should use to store Sous configuration data
func (u *User) ConfigDir() string {
	if sd := os.Getenv("SOUS_CONFIG_DIR"); sd != "" {
		return sd
	}
	xdgConfig := os.Getenv("XDG_CONFIG")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(u.HomeDir, xdgConfigDefault)
	}
	return filepath.Join(xdgConfig, defaultConfigDir)
}

// ConfigFile returns the path to the local Sous config file
func (u *User) ConfigFile() string {
	return filepath.Join(u.ConfigDir(), configFileBase)
}
