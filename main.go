package main

import (
	"fmt"
	"log"

	"github.com/opentable/sous2/ext/git"
	"github.com/opentable/sous2/util/shell"
	"github.com/samsalisbury/psyringe"
)

type (
	WorkdirShell    *shell.Sh
	ScratchDirShell *shell.Sh
)

func main() {
	defer func() {
		if r := recover(); r != nil {
			unhandledError()
			panic(r)
		}
	}()
	s := psyringe.New()
	if err := s.Fill(
		func() (WorkdirShell, error) {
			return shell.Default()
		},
		func() (ScratchDirShell, error) {
			s, err := shell.Default()
			if err != nil {
				return nil, err
			}
			return s, s.CD("/tmp")
		},
		func(sh WorkdirShell) (*git.Client, error) {
			return git.NewClient(sh)
		},
		func(c *git.Client) (*git.Repo, error) {
			return c.OpenRepo(".")
		},
		func(r *git.Repo) (*git.Context, error) {
			return r.Context()
		},
	); err != nil {
		log.Fatal(err)
	}

	c := BuildCommand{}
	if err := s.Inject(&c); err != nil {
		log.Fatal(err)
	}

	fmt.Println(c.Git)
}

type BuildCommand struct {
	Git   *git.Context
	Shell WorkdirShell
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
