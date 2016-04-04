package main

import "github.com/samsalisbury/semv"

var (
	// These variables are set by build flags.
	VersionString, OS, Arch, GoVersion string
	// Revision should be set by the build process using build flags.
	Revision string
	// Version is the current version of Sous
	Version semv.Version
)

func init() {
	if VersionString == "" {
		VersionString = "0.0.0-unversioned"
	}
	Version = semv.MustParse(VersionString)
}
