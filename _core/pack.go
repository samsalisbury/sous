package core

import "fmt"

// Pack describes a project type based on a particular dev stack.
// It is guaranteed that Detect() will be called before any of
// Problems(), ProjectDesc(), and Targets(). Therefore you can
// use the Detect() step to store internal state inside the pack
// if that is useful.
type Pack interface {
	fmt.Stringer
	// Name returns a short constant string naming the pack
	Name() string
	// Desc returns a longer constant string describing the pack
	Desc() string
	// Detect is called to check if the current project is of
	// the type this pack knows how to build. It should return
	// a descriptive error if this pack does not think it can
	// work with the current project.
	Detect() error
	// Problems is called to do a more thorough check on the
	// current project to highlight any potential problems
	// with running the various targets against it.
	Problems() ErrorCollection
	// ProjectVersion returns a semver-compatible version string
	// representing the version of the application described by
	// the source code, or an empty string if that is not
	// available.
	AppVersion() string
	// ProjectDesc returns a description of the current project.
	// It should include important information such as stack
	// name, runtime version, application version, etc.
	AppDesc() string
	// Targets returns a slice of all targets this pack is able
	// to build.
	Targets() []Target
}
