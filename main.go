package main

import (
	"fmt"
	"log"
	"os"

	"github.com/opentable/sous/cli"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/util/logging"
)

// Sous is the Sous CLI root command.
var Sous = &cli.Sous{
	Version:   Version,
	OS:        OS,
	Arch:      Arch,
	GoVersion: GoVersion,
}

func main() {
	// handlePanic will only happen if os.Exit does not get called.
	// The only way for os.Exit not to get called is if action() panics,
	// so handlePanic here is panic-specific behaviour.
	// The reason action is its own method is to allow other deferred calls in
	// there to also happen in spite of panic or return, and definitely before
	// os.Exit can be called.
	defer handlePanic()
	os.Exit(action())
}

// InitializationFailedExitCode (70) is returned when early initialization of
// Sous fails (e.g. when we can't set up logging or other elementary issues).
const InitializationFailedExitCode = 70

func action() int {
	log.SetFlags(log.Flags() | log.Lshortfile)

	di := graph.BuildGraph(Sous.Version, os.Stdin, os.Stdout, os.Stderr)
	type logSetScoop struct {
		*logging.LogSet
	}
	lss := &logSetScoop{}
	if err := di.Inject(lss); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return InitializationFailedExitCode
	}
	defer func() {
		lss.LogSet.AtExit()
	}()

	c, err := cli.NewSousCLI(di, Sous, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return InitializationFailedExitCode
	}
	c.LogSink = lss.LogSet

	return c.Invoke(os.Args).ExitCode()
}

// handlePanic gives us one last chance to send a message to the user in case a
// panic leaks right up to the top of the program. You can disable this message
// for brevity of output by setting DEBUG=YES
func handlePanic() {
	// It is important that this method does not call recover() because we want
	// the runtime to handle spitting out the default stack trace after the
	// message is printed.
	if os.Getenv("DEBUG") == "YES" {
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
