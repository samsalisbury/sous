package main

import (
	"fmt"
	"os"

	"github.com/opentable/sous/cli"
)

func main() {

	panicking := true
	defer handlePanic(&panicking)

	// Create the default CLI dependency graph.
	g, err := cli.BuildGraph()
	if err != nil {
		die(err)
	}

	// Create a CLI for Sous
	c := &cli.CLI{
		OutWriter: os.Stdout,
		ErrWriter: os.Stderr,
		Hooks: cli.Hooks{
			// Before Execute is called on any command, inject it with values
			// from the graph.
			PreExecute: func(c cli.Command) error { return g.Inject(c) },
		},
		// HelpCommand is shown to the user if they type something that looks
		// like they want help, but which isn't recognised by Sous properly. It
		// uses the standard flag.ErrHelp value to decide whether or not to show
		// this.
		HelpCommand: os.Args[0] + " help",
	}

	// Create a new Sous command
	s := &cli.Sous{Version: Version}

	// Add the CLI, and Sous itself to the graph, so they can be injected into
	// the commands.
	g.Fill(c, s)

	// Invoke Sous command, and let it handle exiting.
	result := c.Invoke(s, os.Args)
	panicking = false
	os.Exit(result.ExitCode())
}

// die is used to exit during very early initialisation, before sous itself only
// can be used to handle exiting.
func die(v ...interface{}) {
	fmt.Fprintln(os.Stderr, v...)
	os.Exit(70)
}

// handlePanic gives us one last chance to send a message to the user in case a
// panic leaks right up to the top of the program. You can disable this message
// for brevity of output by setting DEBUG=YES
func handlePanic(panicking *bool) {
	if !*panicking || os.Getenv("DEBUG") == "YES" {
		return
	}
	fmt.Println(panicMessage)
	fmt.Printf("Sous Version: %s\n\n", Version)
}

const panicMessage = `
################################################################################
#                                                                              #
#                                       OOPS                                   #
#                                                                              #
#        Sous has panicked, due to programmer error. Please report this        #
#        to the project maintainers at:                                        #
#                                                                              #
#                https://github.com/opentable/sous/issues                      #
#                                                                              #
#        Please include this entire message and the stack trace below          # 
#        and we will investigate and fix it as soon as possible.               #
#                                                                              #
#        Thanks for your help in improving Sous for all!                       #
#                                                                              #
#        - The OpenTable DevTools Team                                         #
#                                                                              #
################################################################################
`
