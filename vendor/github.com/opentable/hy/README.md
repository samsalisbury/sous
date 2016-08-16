# hy: hierarchical files

[![CircleCI](https://circleci.com/gh/opentable/hy.svg?style=svg)](https://circleci.com/gh/opentable/hy)
[![codecov](https://codecov.io/gh/opentable/hy/branch/master/graph/badge.svg)](https://codecov.io/gh/opentable/hy)
[![Go Report Card](https://goreportcard.com/badge/github.com/opentable/hy)](https://goreportcard.com/report/github.com/opentable/hy)
[![GoDoc](https://godoc.org/github.com/opentable/hy?status.svg)](https://godoc.org/github.com/opentable/hy)
[![License MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

## TODO
- Improve memory efficiency (currently loads everything eagerly in-memory, should use readers and (de)coders rather than []byte and marshalers).
- Add test cases for all failure modes.
  - Attempt to specify a directory for a struct field.
  - SpecialMap has wrong method signatures.
- Make analysis concurrent.
- Make writing concurrent.
- Make reading concurrent.
- Add options for default path names:
  - lowerCamelCase
  - CamelCase
  - snake_case
  - lowercaseonly
- On write, need to pick:
  - Fail if ID field not matching key or index?
  - Overwrite ID with current key or index?
  - Elide ID field from output altogether? (This should be the default, so
    it only matters in memory.)
  - Other?
