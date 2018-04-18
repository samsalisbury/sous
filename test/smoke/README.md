# Running smoke tests...

How to run smoke tests from the root of this repo.

```
$ make install-dev && go test -v -count 1 ./test/smoke
```

You must run install-dev since you need an up to date sous binary in your path, built with the `integration` tag. (It also turns on the race detector.)

If install-dev isn't working for you, you can build sous like this:

```
go install -tags=integration -race ./
```

Make sure that is the sous in your path before running smoke tests!
