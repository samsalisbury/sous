package main

import (
	"runtime"

	"github.com/samsalisbury/semv"
)

var (
	// VersionString is the version of Sous.
	VersionString = "0.0.0-devbuild"
	// Version is the version of Sous.
	Version = semv.MustParse(VersionString + "+" + Revision)
	// OS is the OS this Sous is running on.
	OS = runtime.GOOS
	// Arch is the architecture this Sous is running on.
	Arch = runtime.GOARCH
	// GoVersion is the version of Go this sous was built with.
	GoVersion = runtime.Version()
	// Revision may be set by the build process using build flags.
	Revision string
)
