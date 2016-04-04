# releasing sous

Sous uses [Travis CI] to build all pushes and the master branch.
It also publishes pre-compiled binaries into [sous GitHub releases], if you provide a suitable tag.

Note that only users with push access are able to create releases; it is not possible from a pull request.

[Travis CI]: https://travis-ci.org/opentable/sous
[sous GitHub releases]: https://github.com/opentable/sous/releases

## How to release sous

The `bin/publish` script takes care of parsing the tag, and creating the GitHub release.
In order to trigger publishing the binaries, you need to add a [semver]-compatible [annotated tag]:

```shell
$ git tag -am "Description of the release" v1.0.0-rc
$ git push <remote> v0.0.1-rc
```

Note that if you include a [pre-release field] in the version, e.g. `-rc` or `-beta` etc, then the release will automatically be tagged as a pre-release.
If you do not specify a pre-release field in your semver tag, the release will be considered to be the latest usable version by GitHub,
therefore... **Always specify a pre-release field unless you really want all our users to upgrade!**

Travis will build the tagged version, and then invoke the `bin/publish` script.

[semver]: http://semver.org
[annotated tag]: https://git-scm.com/book/en/v2/Git-Basics-Tagging#Annotated-Tags
[pre-release field]: http://semver.org/#spec-item-9
