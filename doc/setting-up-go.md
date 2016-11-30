# Getting starting with Go

One of the really nice things about Go is that,
once you have a Go environment set up on your workstation,
it's really easy both to install Go software
and to build and distribute your own.

The first step is getting Go set up, though.
For this guide we'll assume that you're using OS X and `bash`
-- if this isn't you, we'll assume you're used to translating instructions
intended for Mac users.

First, you'll need to install Go itself:

```bash
> brew install go
```

Next, you'll need to set up an environment variable.
Open `~/.bash_profile` and add
```bash
export GOPATH='~/go'
```

Just the once, you'll want to reload your `~/.bash_profile`
by running
```bash
> source ~/.bash_profile
```

You should be all set!
Try it out by installing something:
```bash
> go get github.com/aquilax/number_crusher
> number_crusher
```
