package main

import (
	"os"
	"os/user"
	"path/filepath"
)

type Config struct {
	SousSettingsDir string
}

const defaultSettingsDirName = ".sous"

func newDefaultConfig(u *user.User) (Config, error) {
	var settingsDir string
	if sd := os.Getenv("SOUS_SETTINGS_DIR"); sd != "" {
		settingsDir = sd
	} else {
		settingsDir = defaultSettingsDir(u)
	}
	return Config{
		SousSettingsDir: settingsDir,
	}, nil
}

func defaultSettingsDir(u *user.User) string {
	return filepath.Join(u.HomeDir, defaultSettingsDirName)
}
