/*
The semv package provides user-friendly, idiomatic Go semantic version parsing
and range checking. It is based on the semantic versioning v2.0.0 specification
at http://semver.org/spec/v2.0.0.html

This library is designed to be conducive to common use-cases of semantic
versioning, and follows the semver v2.0.0 spec closely, whilst being permissive
to additional real-world usages of semver-like versions, for example partial
versions like 1 or 5.3. Whilst not complete semver versions, these are commonly
seen in the wild, and I think a good library should allow us to work with these
kinds of versions as well.

If you really care that only exact semver v2.0.0 versions are used, you can use
ParseExactSemver2 which will error if the string does not follow the spec.
*/
package semv
