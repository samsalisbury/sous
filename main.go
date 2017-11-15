package main

import (
	"fmt"
	"log"
	"os"

	"github.com/opentable/sous/cli"
	"github.com/opentable/sous/config"
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

func sniffVerbosity(cliArgs []string) graph.VerbosityOverride {
	var s, q, v, d bool
	// We want to give higher verbosity precedence in the case more than
	// one flag is set, so first check for them all then return the most
	// verbose one detected.
	for _, a := range cliArgs {
		if a == "-d" {
			d = true
		}
		if a == "-v" {
			v = true
		}
		if a == "-q" {
			q = true
		}
		if a == "-s" {
			s = true
		}
	}
	if d {
		return graph.VerbosityOverride{
			Overridden: true,
			Value:      &config.Verbosity{Debug: true},
		}
	}
	if v {
		return graph.VerbosityOverride{
			Overridden: true,
			Value:      &config.Verbosity{Loud: true},
		}
	}
	if q {
		return graph.VerbosityOverride{
			Overridden: true,
			Value:      &config.Verbosity{Quiet: true},
		}
	}
	if s {
		return graph.VerbosityOverride{
			Overridden: true,
			Value:      &config.Verbosity{Silent: true},
		}
	}
	return graph.VerbosityOverride{}
}

func action() int {
	log.SetFlags(log.Flags() | log.Lshortfile)

	di := graph.BuildGraph(Sous.Version, os.Stdin, os.Stdout, os.Stderr)

	// We can't call flag.Parse yet because haven't defined our
	// flags. However, we want to guess at verbosity from the command line.
	// In addition, by cloning the graph here, we allow later configuration
	// of verbosity and logging in general.
	initializationDI := di.Clone()
	initializationDI.Add(sniffVerbosity(os.Args))

	type logSetScoop struct {
		*logging.LogSet
	}
	lss := &logSetScoop{}
	if err := initializationDI.Inject(lss); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return InitializationFailedExitCode
	}

	defer func() {
		// Gracefully shut down the logs created at initialization.
		lss.LogSet.AtExit()
		// Grab the LogSet used by the main di graph to AtExit on that too.
		// If we fail to get it here, don't worry as that means it was never
		// instantiated by the main graph. This can happen if e.g. a bad flag
		// is passed to the CLI which causes an exit prior to the Parsed event
		// firing on the CLI.
		if err := di.Inject(lss); err == nil {
			lss.LogSet.AtExit()
		}
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
