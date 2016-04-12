package cli

import (
	"os"
	"os/user"
	"path/filepath"
)

type (
	User struct {
		*user.User
	}
)

const DefaultConfigDir = ".sous"

func (u *User) ConfigDir() string {
	if sd := os.Getenv("SOUS_CONFIG_DIR"); sd != "" {
		return sd
	}
	return filepath.Join(u.HomeDir, DefaultConfigDir)
}

func (u *User) ConfigFile() string {
	return filepath.Join(u.ConfigDir(), "config")
}
