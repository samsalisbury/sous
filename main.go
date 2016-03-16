package main

import (
	"fmt"
	"os"

	"github.com/opentable/sous2/cli"
)

func main() {
	// wrap any panics which leak up to this level, to ask the user to report
	// errors.
	defer func() {
		if r := recover(); r != nil {
			unhandledError()
			panic(r)
		}
	}()

	c := cli.CLI{
		Version: Version,
	}

	// The CLI itself should manage exiting cleanly.
	c.Invoke(os.Args)

	// If it fails to exit due to programmer error, let the user know.
	fmt.Fprintf(os.Stderr, "error: sous did not exit cleanly")
	os.Exit(70)
}

func unhandledError() {
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
