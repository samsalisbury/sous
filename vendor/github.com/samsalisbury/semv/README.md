# semv

semv is a semantic versioning (semver) library for go

## Ethos

This library is designed to be friendly to common use-cases of semantic versioning, and follows [the semver v2.0.0 spec] closely, whilst being permissive to additional real-world usages of semver-like versions, for example partial versions like `1` or `5.3`. Whilst not valid semver versions, these are commonly seen in the wild, and I think a good library should allow us to work with these kinds of versions as well.

If you really care that only exact semver v2.0.0 versions are used, you can use `ParseExactSemver2` which will error if the string does not follow the spec.

[the semver v2.0.0 spec]: http://semver.org/spec/v2.0.0.html


## Example

```go
package main
import (
	"fmt"
	"os"

	"github.com/samsalisbury/semv"
)
func main() {
	r, err := semv.ParseRange("^3.0.1")	
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	v1, err := semv.Parse("1.0.0")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if !r.SatisfiedBy(v1) {
		fmt.Printf("%q is not satisfied by %q", r, v)
	} else {
		fmt.Printf("%q is satisfied by %q", r, v)
	}
}
// output: "^3.0.1" is not satisfied by "1.0.0"
```

Note that here we are using hard-coded versions, so using the full `ParseRange` and `Parse` functions is not necessary. Instead, the above would be neater written using the `MustParse` variations, which panic instead of returning an error.

```go
package main
import (
	"fmt"

	"github.com/samsalisbury/semv"
)
func main() {
	r := semv.MustParseRange("^3.0.1")	
	v1 := semv.MustParse("1.0.0")
	if !r.SatisfiedBy(v1) {
		fmt.Printf("%q is not satisfied by %q", r, v)
	} else {
		fmt.Printf("%q is satisfied by %q", r, v)
	}
}
// output: "^3.0.1" is not satisfied by "1.0.0"

```

### Version Parsing

Version string parsing using `Parse()` is permissive by default. All of the following will parse into a Version instance:

- `"1"` parses as `{Major: 1, Minor: 0, Patch: 0, Pre: "", Meta: ""}`
- `"1.2"` parses as `{Major: 1, Minor: 2, Patch: 0, Pre: "", Meta: ""}`
- `"1.2.3"` parses as `{Major: 1, Minor: 2, Patch: 3, Pre: "", Meta: ""}`
- `"1-beta"` parses as `{Major: 1, Minor: 0, Patch: 0, Pre: "beta", Meta: ""}`
- `"1.2+abc"` parses as `{Major: 1, Minor: 2, Patch: 0, Pre: "", Meta: "abc"}`
- `"1.2-beta+abc"` parses as `{Major: 1, Minor: 2, Patch: 0, Pre: "beta", Meta: "abc"}`

Note that whilst permissive, parse will still return errors for the following conditions:

- `UnexpectedCharacter` when encountering anything other than:
  - `[0-9\.]` in a major or minor field,
  - `[0-9\-]+`, in a patch field,
  - `[0-9a-zA-Z\-]` in a prerelease or meta field.
- `ZeroLengthNumeric` when encountering 2 dots together in the major, minor,
  patch fields.

If you want to validate that input is in exact semver 2.0.0 format, you should use `ParseExactSemver2` instead, which returns these additional errors:
- `VersionIncomplete` when either the minor or patch fields are missing
- `LeadingZero` when a major, minor, or patch contains an erroneous preceding
  zero character.

### Range Parsing

Range parsing  using `ParseRange` and `MustParseRange` allows common range specifiers like `>`, `>=`, `<`, `<=`, as well as modern range shortcuts as used in npm and other tools: `^` and `~`.

Currently, only single-version ranges are supported, so the only way to parse a range with both an upper and a lower limit is by using the `^` and `~` characters.

- `^1.2.3 == >=1.2.3 and <2.0.0`
- `~1.2.3 == >=1.2.3 and <1.3.0`

### VersionList

The `VersionList` type is a slice of versions. It implements `sort.Interface` so you can order arbitrary lists of versions.

You can find the greatest version in a `VersionList` satisfying a `Range` using the `GreatestSatisfying` method, e.g.:

```go
v, ok := MustParseList("1.0.0", "0.9.1", "1.2.0").GreatestSatisfying(MustParseRange("^1.0.0"))
if !ok {
	fmt.Println("No matching version found.")
} else {
	fmt.Println(v)
}
// Output:
// 1.2.0
```

### Version.String()

Simply calling `.String()` on a version created using one of the `New(Version|MajorMinorPatch)` funcs will print the full version string, omitting the optional prerelease and/or metadata sections depending on if they contain any data.

If a version is created by parsing a string, its original format is recorded with the version. In this case, calling `.String()` will print the version it its original format. E.g.:

- `MustParse("1").String() == "1"`
- `MustParse("1.0").String() == "1.0"`
- `MustParse("1.0.0").String() == "1.0.0"`
- `MustParse("1.0.0-beta").String() == "1.0.0-beta"`
- `MustParse("1.0.0-beta+abc").String() == "1.0.0-beta+abc"`
- `MustParse("1.0.0+abc").String() == "1.0.0+abc"`

This feature is useful when dealing with partial versions, as used by many popular projects, e.g. [Go], [NPM], and others. You can always use `.Format()` to print an exact format.

[Go]: https://golang.org
[NPM]: https://www.npmjs.com

### Version.Format()

If you want to print your version in a specific format, you can use `.Format()` with a format string, e.g.:

- `MustParse("1.2.3-beta.1+abc-def.2").Format("M") == "1"`
- `MustParse("1.2.3-beta.1+abc-def.2").Format("M.m") == "1.2"`
- `MustParse("1.2.3-beta.1+abc-def.2").Format("M.m.p") == "1.2.3"`
- `MustParse("1.2.3-beta.1+abc-def.2").Format("M.m.p-?") == "1.2.3-beta.1"`
- `MustParse("1.2.3-beta.1+abc-def.2").Format("M.m.p+?") == "1.2.3+abc-def.2"`
- `MustParse("1.2.3-beta.1+abc-def.2").Format("M.m.p-?+?") == "1.2.3-beta.1+abc-def.2"`


