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

