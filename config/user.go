package config

import (
	"os"
	"os/user"
	"path/filepath"
)

// LocalUser represents the OS user running Sous.
type LocalUser struct {
	*user.User
}

const (
	defaultConfigDir = "sous"
	xdgConfigDefault = ".config"
	configFileBase   = "config.yaml"
)

// DefaultConfig builds a default configuration for this user
func (u *LocalUser) DefaultConfig() Config {
	c := DefaultConfig()
	c.Docker.DatabaseConnection = filepath.Join(u.ConfigDir(), "data.db")
	return c
}

// ConfigDir returns the directory we should use to store Sous configuration data
func (u *LocalUser) ConfigDir() string {
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
func (u *LocalUser) ConfigFile() string {
	return filepath.Join(u.ConfigDir(), configFileBase)
}
