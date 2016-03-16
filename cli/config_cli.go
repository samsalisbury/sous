package cli

import (
	"os"
	"os/user"
	"path/filepath"

	"github.com/opentable/sous2/sous"
)

const defaultSettingsDirName = ".sous"

func newDefaultConfig(u *user.User) (sous.Config, error) {
	var settingsDir string
	if sd := os.Getenv("SOUS_SETTINGS_DIR"); sd != "" {
		settingsDir = sd
	} else {
		settingsDir = defaultSettingsDir(u)
	}
	return sous.Config{
		SettingsDir: settingsDir,
	}, nil
}

func defaultSettingsDir(u *user.User) string {
	return filepath.Join(u.HomeDir, defaultSettingsDirName)
}
