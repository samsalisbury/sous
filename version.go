package main

import "github.com/samsalisbury/semv"

var (
	Revision string
	Version  semv.Version
)

func init() {
	if Revision == "" {
		Revision = "unknown-revision"
	}
	Version = semv.MustParse("1.0.0-alpha+" + Revision)
}
