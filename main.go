package main

import (
	"flag"
	"fmt"
	"io/ioutil"
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

func getLogSet(graph *graph.SousGraph) (*logging.LogSet, error) {
	type logSetScoop struct {
		*logging.LogSet
	}
	lss := &logSetScoop{}
	if err := graph.Inject(lss); err != nil {
		return nil, err
	}
	return lss.LogSet, nil
}

func action() int {
	log.SetFlags(log.Flags() | log.Lshortfile)

	mainGraph := graph.BuildGraph(Sous.Version, os.Stdin, os.Stdout, os.Stderr)

	// Clone the graph so we can add early verbosity override from flags.
	preParseGraph := &graph.SousGraph{Psyringe: mainGraph.Clone()}
	preParseGraph.Add(earlyLoggingVerbosityOverride(os.Args))
	preParseLogSet, err := getLogSet(preParseGraph)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return InitializationFailedExitCode
	}

	defer func() {
		// Gracefully shut down the logs created at initialization.
		preParseLogSet.AtExit()
		// Grab the LogSet used by the main graph to AtExit on that too.
		// If we fail to get it here, don't worry as that means it was never
		// instantiated by the main graph. This can happen if e.g. a bad flag
		// is passed to the CLI which causes an exit prior to the Parsed event
		// firing on the CLI.
		if mainLogSet, err := getLogSet(mainGraph); err == nil {
			mainLogSet.AtExit()
		}
	}()

	c, err := cli.NewSousCLI(mainGraph, Sous, os.Stdout, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return InitializationFailedExitCode
	}
	// This LogSink is replaced after the Parsed event fires on the CLI.
	c.LogSink = preParseLogSet

	return c.Invoke(os.Args).ExitCode()
}

func earlyLoggingVerbosityOverride(cliArgs []string) graph.VerbosityOverride {
	globalFS := flag.NewFlagSet("verbosity", flag.ContinueOnError)
	globalFS.SetOutput(ioutil.Discard)
	var verbosity config.Verbosity
	cli.AddVerbosityFlags(&verbosity)(globalFS)
	// Explicitly ignore this error because we expect there to be other flags
	// that are not yet defined.
	// This behaviour tested by https://play.golang.org/p/kEy-mtM3H0
	//
	// NOTE: It is possible that a flag taking an argument that is validly "-d"
	// or one of the other verbosity flags will confuse this method into setting
	// the verbosity incorrectly. This is why we only use this for
	// initialisation logging. Later the entire command line is parsed again
	// (once we know which command is being run), and a new LogSet is created
	// based on the result of that, which is therefore guaranteed to be
	// as accurate as the FlagSet.Parse method.
	_ = globalFS.Parse(os.Args)
	// Nothing was overridden by flags as verbosity == zero verbosity.
	if (verbosity == config.Verbosity{}) {
		// The zero verbosity override means we defer to config or defaults.
		return graph.VerbosityOverride{}
	}
	// At least one verbosity flag was present, so verbosity is overridden.
	return graph.VerbosityOverride{
		Overridden: true,
		Value:      &verbosity,
	}
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
