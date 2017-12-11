# Getting Started

# Install Sous

Sous is distributed as a single binary, in a tarball also containing documentation.

# Homebrew
On macOS you can install via [homebrew](https://brew.sh):

```
$ brew install opentable/public/sous
==> Tapping opentable/public
🍺  /usr/local/Cellar/sous/0.5.1: 5 files, 15.8MB, built in 5 seconds
```

**NOTE: This formula has recently been migrated to a new tap at [opentable/public].**
If you previously installed sous from the old `opentable/osx-tools` tap,
please uninstall your current sous using `brew uninstall sous` and remove the old
tap by running `brew untap opentable/osx-tools`. Then follow the instructions above
to install sous properly, and have it upgraded by `brew upgrade` in future.

If you had ever installed the old head-only formula for Sous, you should definitely
uninstall it using the instructions above, as head-only formula block upgrades.
If you are not sure, then follow the instructions above to make sure you are on the
the latest version of Sous, and that you continue to receive updates in future.

[opentable/public]: https://github.com/opentable/homebrew-public 

# Linux and Mac tarballs

[Download the archive] and extract it. For example on Darwin (OS X):

```shell
$ VERSION=0.1.7 
# In the following curl command:
#     -L means follow HTTP redirects.
#     -O means save the file to the current directory  with the same name.
$ curl -LO https://github.com/opentable/sous/releases/download/$VERSION/sous-darwin-amd64_$VERSION.tar.gz
$ tar -vxzf sous-darwin-amd64_$VERSION.tar.gz
```

This will create a directory called sous-darwin-amd64-$VERSION. Next, you need to copy
the `sous` binary to somewhere in your path. We recommend `/usr/local/bin`. To do that:

```shell
# You will be prompted for your password after pressing enter.
$ sudo cp sous-darwin-amd64_$VERSION/sous /usr/local/bin
```

That's it, test your installation by running `sous version`:

```shell
$ sous version
sous version 0.1.7+f3dc702d216b7bdbab2b55dc6b91b8bee7a55abd (go1.7.3 darwin/amd64)
```
    
[Download the archive]: https://github.com/opentable/sous/releases

# Installing from HEAD

If you want to install the latest unstable development version of Sous, you will need to install
go 1.7.3 or later, set up your `GOPATH` and then use `go get` to install it. E.g.:

```shell
$ go get github.com/opentable/sous
```

Installing like this is not recommended unless you are a Sous developer. You can tell if Sous
was installed this way by running `sous version`, and it will give output like the following:

```shell
$ sous version
sous version 0.0.0-devbuild (go1.7.3 darwin/amd64)
This is an unsupported development build.
Get supported releases from https://github.com/opentable/sous/releases
```
