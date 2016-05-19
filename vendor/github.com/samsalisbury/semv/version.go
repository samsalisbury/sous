package semv

import (
	"fmt"
	"strconv"
	"strings"
)

type (
	// Version is a semver version
	Version struct {
		Major, Minor, Patch      int
		Pre, Meta, DefaultFormat string
	}
	// VersionIncomplete is an error returned by ParseExactSemver2
	// when a version is missing either minor or patch parts.
	VersionIncomplete struct {
		MissingPart string
	}
	// UnexpectedCharacter is an error returned by Parse and ParseExactSemver2
	// when they contain unexpected characters at a particular location.
	UnexpectedCharacter struct {
		Char rune
		Pos  int
	}
	// ZeroLengthNumeric is an error returned when either major, minor, or
	// patch is zero length. That is, when parsing a string containing two
	// consecutive dots. E.g. "1..3" or "..1"
	ZeroLengthNumeric struct {
		ZeroLengthPart string
	}
	// PrecedingZero is an error returned when one of the major, minor, or
	// patch parts contains a preceding zero. This error is only returned
	// when using ParseExactSemver2_0_0, and this validation is ignored
	// otherwise.
	PrecedingZero struct {
		PrecedingZeroPart, InputString string
	}
	mode uint
)

// NewVersion returns a new version with all fields set.
func NewVersion(major, minor, patch int, pre, meta string) Version {
	return Version{major, minor, patch, pre, meta, ""}
}

// NewMajorMinorPatch returns a new version with just the major, minor, and patch
// fields set.
func NewMajorMinorPatch(major, minor, patch int) Version {
	return Version{major, minor, patch, "", "", ""}
}

func (err VersionIncomplete) Error() string {
	return fmt.Sprintf("version incomplete: missing %s component", err.MissingPart)
}

func (err UnexpectedCharacter) Error() string {
	return fmt.Sprintf("unexpected character '%s' at position %d", string(err.Char), err.Pos)
}

func (err ZeroLengthNumeric) Error() string {
	return fmt.Sprintf("unexpected zero-length %s component", err.ZeroLengthPart)
}

func (err PrecedingZero) Error() string {
	return fmt.Sprintf("unexpected preceding zero in %s component: %q",
		err.PrecedingZeroPart, err.InputString)
}

const (
	modeMajor mode = iota
	modeMinor
	modePatch
	modePre
	modeMeta
	digits               = "01234567890"
	validPreAndMetaChars = digits + ".-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// PreDelim is the character separating major.minor.patch from the prerelease field.
	PreDelim = "-"
	// MetaDelim is the character separating major.minor.patch[-pre] from the metadata field.
	MetaDelim = "+"
	// Formatting characters Major, Minor, Patch, Pre, PreRaw, Meta, MetaRaw are used by
	// Version.Format
	Major   = "M"
	Minor   = "m"
	Patch   = "p"
	Pre     = PreDelim + "?"
	PreRaw  = PreDelim + "!"
	Meta    = MetaDelim + "?"
	MetaRaw = MetaDelim + "!"
	// MajorMinor is a format string used to format a version to just its major and minor
	// components.
	MajorMinor = Major + "." + Minor
	// MajorMinorPatch is a format string used to format a version to just its major,
	// minor, and patch fields.
	MajorMinorPatch = MajorMinor + "." + Patch
	// MMMPre is a format string, like MajorMinorPatch, with the prerelease field added
	// if it is nonempty.
	MMPPre = MajorMinorPatch + Pre
	// Complete is a format string, like MMMPre, but it also includes the metadata field
	// if it is nonempty.
	Complete = MMPPre + Meta
)

// Validate ensures that none of the fields are negative.
func (v Version) Validate() error {
	if v.Major < 0 || v.Minor < 0 || v.Patch < 0 {
		return fmt.Errorf("major, minor, patch must all be positive")
	}
	return nil
}

// String prints the string representation of this version.
// If the version was originally parsed, then String will attempt
// to re-print it at the same level of detail as was originally parsed in.
//
// E.g. Parse("1").String() == "1"
//      Parse("1.2").String() == "1.2"
//      Parse("1.2.3").String() == "1.2.3"
//      Parse("1.2.3-beta").String() == "1.2.3-beta"
func (v Version) String() string {
	return v.Format(v.DefaultFormat)
}

// MajorMinorPatch returns a new version with the prerelease and meta fields
// set to the empty string, and major, minor, patch equalling the
// major, minor, patch of the version it was invoked on.
func (v Version) MajorMinorPatch() Version {
	v.Pre = ""
	v.Meta = ""
	return v
}

// IsPrerelease returns true if the prerelease field is not empty.
func (v Version) IsPrerelease() bool {
	return v.Pre != ""
}

// Format takes a format string and outputs the version accordingly.
//
// You can use the following format strings (which are available as constants):
//
//     Major = "M", minor = "m", Patch = "p", Pre = "-?", Meta = "+?",
//     PreRaw = "-!", MetaRaw = "+!"
//
// Pre and Meta are replaced with the empty string when Pre or Meta are empty,
// respectively, or, with the prerelease version prefixed by '-' or the metadata
// prefixed with '+', if either are not empty.
//
// See other constants in this library for more. The empty string is treated
// equivalently to the format string "M.m.p-?+?".
func (v Version) Format(format string) string {
	if format == "" {
		format = Complete
	}
	replacements := map[string]interface{}{
		Major: v.Major,
		Minor: v.Minor,
		Patch: v.Patch,
	}
	formatted := replaceAll(format, replacements)
	if v.Pre != "" {
		formatted = strings.Replace(formatted, Pre, PreDelim+v.Pre, -1)
	} else {
		formatted = strings.Replace(formatted, Pre, "", -1)
	}
	if v.Meta != "" {
		formatted = strings.Replace(formatted, Meta, MetaDelim+v.Meta, -1)
	} else {
		formatted = strings.Replace(formatted, Meta, "", -1)
	}
	formatted = strings.Replace(formatted, PreRaw, v.Pre, -1)
	formatted = strings.Replace(formatted, MetaRaw, v.Meta, -1)
	return formatted
}

// Less returns true if the version it is invoked on is less than the version
// passed in, according to the precendence rules in semver 2.0.0
func (v Version) Less(than Version) bool {
	if v.Major < than.Major {
		return true
	}
	if v.Major > than.Major {
		return false
	}
	if v.Minor < than.Minor {
		return true
	}
	if v.Minor > than.Minor {
		return false
	}
	if v.Patch < than.Patch {
		return true
	}
	if v.Patch > than.Patch {
		return false
	}
	if v.Pre != "" && than.Pre == "" {
		return true
	}
	if v.Pre == "" && than.Pre != "" {
		return false
	}
	vPreParts := v.PreComponents()
	thanPreParts := than.PreComponents()
	for i := range thanPreParts {
		if vInt, err := strconv.Atoi(vPreParts[i]); err == nil {
			if tInt, err := strconv.Atoi(thanPreParts[i]); err == nil {
				if vInt < tInt {
					return true
				}
				if vInt > tInt {
					return false
				}
			} else {
				return true
			}
		}
		if vPreParts[i] < thanPreParts[i] {
			return true
		}
		if vPreParts[i] > thanPreParts[i] {
			return false
		}
		if len(vPreParts) == i+1 && len(thanPreParts) > len(vPreParts) {
			return true
		}
	}
	return false
}

// MMPLess returns true if the version it is invoked on's major, minor, patch
// triple is less than the passed in version's major, minor, patch triple.
func (v Version) MMPLess(other Version) bool {
	return v.MajorMinorPatch().Less(other.MajorMinorPatch())
}

// MMPEqual returns true if the major, minor, patch triple of the version it is
// invoked in is equal to the major, minor, patch triple of the passed in version.
func (v Version) MMPEqual(other Version) bool {
	return v.MajorMinorPatch().Equals(other.MajorMinorPatch())
}

// PreComponents returns the prerelease field split by . characters.
func (v Version) PreComponents() []string {
	return strings.Split(v.Pre, ".")
}

// Equals returns true if the versions are equal according to semver 2.0.0 precedence
// rules. If you want to test that the entire version is exactly equal, use the normal
// '=' operator.
func (v Version) Equals(other Version) bool {
	return !v.Less(other) && !other.Less(v)
}

// ValueEquals works on Version pointers, and checks that their values
// are equal. Also returns true if both are nil, and obviously false if
// one is nil and ther other one isn't.
func (v *Version) ValueEquals(other *Version) bool {
	if v == nil && other == nil {
		return true
	}
	if v == nil || other == nil {
		return false
	}
	return (*v).Equals(*other)
}

// Satisfies is a convenience function. v.Satisfies(r) == r.IsSatisfied(v)
func (v Version) Satisfies(r Range) bool {
	return r.SatisfiedBy(v)
}

// IncrementMajor returns a new Version with the major field incremented by 1
// and minor and patch set to zero.
func (v Version) IncrementMajor() Version {
	v.Major++
	v.Minor = 0
	v.Patch = 0
	return v
}

// IncrementMinor returns a new Version with the minor field incremented by 1
// and the patch set to zero.
func (v Version) IncrementMinor() Version {
	v.Minor++
	v.Patch = 0
	return v
}

// IncrementPatch returns a new Version with the patch field incremented by 1.
func (v Version) IncrementPatch() Version {
	v.Patch++
	return v
}

// SetPre returns a new Version with the prerelease field set to the provided
// string.
func (v Version) SetPre(s string) Version {
	v.Pre = s
	return v
}

// SetMeta returns a new Version with the meta field set to the provided value.
func (v Version) SetMeta(s string) Version {
	v.Meta = s
	return v
}

// MarshalYAML allows sensible YAML marshalling by gopkg.in/yaml.v2
func (v Version) MarshalYAML() (interface{}, error) {
	return v.String(), nil
}

// UnmarshalYAML allows sensible YAML unmarshalling by gopkg.in/yaml.v2
func (v *Version) UnmarshalYAML(f func(interface{}) error) (err error) {
	var s string
	if err = f(&s); err != nil {
		return
	}
	*v, err = Parse(s)
	return
}
