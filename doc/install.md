# Getting Started

# Install Sous

Sous is distributed as a single binary, in a tarball also containing documentation.

# Linux and Mac tarballs

[Download the archive] and extract it. For example on Darwin (OS X):

```shell
$ VERSION=0.1.7 
# In the following curl command:
#     -L means follow HTTP redirects.
#     -O means save the file to the current directory  with the same name.
$ curl -LO https://github.com/opentable/sous/releases/download/v$VERSION/sous-darwin-amd64_$VERSION.tar.gz
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
