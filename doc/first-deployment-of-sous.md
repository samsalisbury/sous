# Deploying Sous for the first time

The typical use for Sous is to deploy software with it.
But before that happens,
you have to have to deploy Sous itself.

# Required Infrastructure

Sous has some services that it depends on
existing in your environment.

## Global Deployment Manifest Hosting

Sous uses a centralized git repository to store
the source of truth as to what's deployed in the environment.
A private Github repo can work just fine for this purpose,
and so, probably would a little gitolite running inside
your datacenter.

## Docker Registry

Sous relies on a Docker registry to store and locate images.
The assumption is that you'll use a private registry for this,
since the images we're talking about will be services running
inside your infrastructure.

## Singularity and Mesos

Sous targets deploying to the Singlurity Mesos scheduler,
without which you'll get little mileage out of it.

## The Service Wrapper

In the `/examples` directory,
you'll find
a `main.go` service wrapper,
a `sous-server.yaml` manifest,
and a `Dockerfile`
that are the start of a Mesos hosted Sous server.
The particulars of your specific microservices environment
will greatly influence how this needs to function,
so we can't provide a complete one-size-fits-all solution.
For instance, you'll likely need to arrange for
some kind of service discovery and name announcement;
the particulars of which are up to you.

So, you'll start a new git repo for the service itself,
fill out the details as appropriate,
and push it out.

Once you get that sorted out,
you can deploy Sous with Sous though.

## The First Deploy

You'll also want to check out your GDM locally.
We need to do a little system-wide configuration.
```bash
> git clone <gdm repo remote> `sous config StateLocation`
> cd `sous config StateLocation`
```

Now, you'll want to edit `defs.yaml` and add this to it:
```
CONTENTS TBD
```
Now, we're going to run `sous` locally,
using your checkout of the repository.
This is **not** the normal mode of operation for Sous.
Normally, Sous servers manage the GDM repo exclusively,
but in order to bootstrap the server,
we'll need have it do its thing from our workstation.

```bash
> export SOUS_SERVER=''
> cd <service project dir>
> git tag -a '0.0.1' && git push --tags
> sous build
> sous deploy -cluster <name> -tag '0.0.1'
```

Now, you should have your Sous service wrapper running in Singularity,
and can distribute it's URL to your users in order for them to access it.
