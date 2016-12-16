# Contributing

In the interest of streamlining the process of going from zero to contributions,
here's a quick guide to getting going with [Go.]

[Go.](golang.org)

## Setting up

First, you'll need to install Go itself.
The official web page is at https://golang.org/doc/install.

Linux distros tend to include a modern version of golang in their repositories. Try `apt-get golang` or similar.

On Macs, assuming you've got Homebrew already installed, it should be as easy as

     $ brew install golang

From there, you need to ensure that you've set up some environment variables correctly. Add the following to `~/.profile`:
```bash
export GOPATH=$HOME/work
export PATH=$PATH:$GOPATH/bin
```
(assuming that you use bash - for other shells, you'll need to make adjustments.)

As a one time thing, `source ~/.profile` to be sure that the environment variables are set up.

That's it: you should now have a functional go environment.

## Getting Sous

     $ go get github.com/opentable/sous

This will not only install the `sous` executable in `$GOPATH/bin` (and therefore in your PATH),
but it will also fetch the source code into `$GOPATH/src/github.com/opentable/sous`.
You're already ready to branch, hack, and pull-request.

## Running tests

Use `./bin/test` to run normal unit tests.

Use `./bin/dev-integration` to run full integration tests.

In general, run `bin/test` in the course of normal development,
reserving `bin/dev-integration` for just before pushing a pull request.

## Workflow

We've adopted a pull-request, git-flow-y model of development for Sous.
Start by forking the project, then
commit changes on a branch and issue pull-requests with your changes.
We would appreciate tests for new code,
(and new tests for old code...)

We've adopted the use of [Travis CI](https://travis-ci.org)
and [CodeCov](https://codecov.io)
to help maintain and improve the code quality on Sous.
Note that PRs will normally be checked by these services before acceptance.
For the most part, this means that you'll need to ensure
that there are tests for your contributions.
