// Package actions contains the actions that the CLI can take.
// cmdr.Commands get user input, and then drive the actions.
package actions

// An Action is used to Do things.
type Action interface {
	Do() error
}
