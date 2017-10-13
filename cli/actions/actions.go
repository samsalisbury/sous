// Package actions contains the actions that the CLI can take.
// cmdr.Commands get user input, and then drive the actions.
package actions

type injector interface {
	Add(interface{})
	Inject(interface{})
}

// An Action is used to Do things.
type Action interface {
	Do() error
}

var guards = map[string]bool{}

func guardedAdd(di injector, guardName string, value interface{}) {
	if guards[guardName] {
		return
	}
	guards[guardName] = true
	di.Add(value)
}
