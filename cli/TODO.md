# TODO

This package has been a stumbling block
for devs new to the project
and a home to bugs for some time.
A change to design has been contemplated,
but not yet implemented.

## Goal Design

Every cmdr.Command in the cli package
should have its fields reduced
to just its flags.
Their Execute method should call
e.g.
`graph.GimmeASousInitExectutor(...) *executors.SousInit`
and then call Run() on that.

Upshot is that the injection
for CLI
should get put into the graph package.
And the executors should be DI'd,
so that they can be more easily tested.

The cmdr.Command.Executor methods then
become responsible *only*
for UI concerns like
handling arguments
and outputting results.
