# sous code

Sous code is always formatted with [gofmt],
and closely follows the conventions outlined in [effective go].

You can use this [git pre-commit hook] to help you avoid committing incorrectly formatted code.
The `bin/check-gofmt` script is run in CI to ensure badly formatted code fails the build.

[gofmt]: https://golang.org/cmd/gofmt/
[effective go]: https://golang.org/doc/effective_go.html
[git pre-commit hook]: https://golang.org/misc/git/pre-commit

## Managing dependencies

Sous' dependencies are managed using [govendor]. You can install it by running

    $ go get -u github.com/kardianos/govendor

Sous must always have all of its dependencies vendored into the `/vendor` directory,
this helps keep the builds repeatable, and simplifies offline work.
The `bin/safe-build` script performs a build using only the vendored dependencies.

[govendor]: https://github.com/kardianos/govendor

## Organisation of this repo

In order to keep the repo organised, we have a number of top-level directories for specific
parts of the code-base:

- **bin**: standalone scripts and binary utils used to build, test, experiment with sous
- **cli**: the Sous command line interface, using util/cmdr
- **doc**: documentation
- **ext**: libraries talking to external services, e.g. the filesystem, network, shell etc. Packages in ext will typically talk to these services to construct or act upon structures defined in lib.
- **server**: the sous HTTP server library.
- **lib**: the main sous library providing core sous functionality. MUST NOT reference ext, cli, server.
- **util**: completely standalone, generic utility libraries. MUST NOT reference anything outside of util, vendor. Util libs can be consumed from anywhere else in the code base.
- **vendor**: standard vendor directory.


