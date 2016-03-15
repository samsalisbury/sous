package main

import (
	"fmt"

	"github.com/samsalisbury/semv"
)

const version = "1.0.0-alpha"

var (
	// Revision should be set by the build process using build flags.
	Revision string
	// Version is the current version of Sous
	Version = semv.MustParse(fmt.Sprintf("%s+%s", version, Revision))
)
