package main

import (
	"fmt"
	"os"
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			unhandledError()
			panic(r)
		}
	}()

	deps, err := buildDeps()
	if err != nil {
		panic(err)
	}

	c := BuildCommand{}
	fatalInternalError(deps.Inject(&c))

	fmt.Println(c.Git)
}

// fatalInternalError does nothing if it is passed nil, otherwise it prints the
// error message, and exits with exit code 2. This should be used only where
// Sous itself is at fault, or fails to initialise. After initialisation, if
// something goes wrong with performing the user action, you should exit with
// exit code 1.
func fatalInternalError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}

type BuildCommand struct {
	Git   LocalGitContext
	Shell LocalWorkDirShell
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
