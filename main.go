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
