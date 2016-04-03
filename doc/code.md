# sous code

Sous code is always formatted with [gofmt],
and closely follows the conventions outlined in [effective go].

You can use this [git pre-commit hook] to help you avoid committing incorrectly formatted code.
The `bin/check-gofmt` script is run in CI to ensure badly formatted code fails the build.

[gofmt]: https://golang.org/cmd/gofmt/
[effective go]: https://golang.org/doc/effective_go.html
[git pre-commit hook]: https://golang.org/misc/git/pre-commit

## Managing dependencies

Sous must always have all of its dependencies vendored into the `/vendor` directory,
this helps keep the builds repeatable, and enables offline work.
The `bin/safe-build` script performs a build using only the vendored dependencies.

You should use a recent version of [godep] to manage the dependencies in the vendor directory, e.g.

```sh
$ godep save ./...   # to add new dependencies, or
$ godep update ./... # to update existing dependencies
$ git add vendor/ Godeps/
$ git commit "Describe the dependencies updated or added"
```

**Warning: never use `godep save` without adding `./...`
because that will delete dependencies of the sub-packages, which is probably not what you want.**

[godep]: https://github.com/tools/godep
