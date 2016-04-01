# sous code

Sous code is always formatted with [gofmt],
and closely follows the conventions outlined in [effective go].

[gofmt]: https://golang.org/cmd/gofmt/
[effective go]: https://golang.org/doc/effective_go.html

## Managing dependencies

Sous must always have all of its dependencies vendored into the `/vendor` directory,
this helps keep the builds repeatable, and enables offline work.
The `bin/safe-build` script performs a build using only the vendored dependencies.

You should use a recent version of [godep] to manage the dependencies in the vendor directory, e.g.

```sh
$ godep save ./...
$ godep update ./...
$ git add vendor Godeps
$ git commit "Describe the dependencies updated or added"
```

**Warning: never use `godep save` without adding `./...`
because that will delete dependencies of the sub-packages, which is probably not what you want.**

[godep]: https://github.com/tools/godep
