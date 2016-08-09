package semv

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

// Parse permissively parses a string as a semver value. The minimal string
// which will not error is a single digit, which will be interpreted as a major
// version, e.g. Parse("1").Format("M.m.p") == "1.0.0".
//
// All of the following will parse into a Version instance:
//
//     "1" parses as {Major: 1, Minor: 0, Patch: 0, Pre: "", Meta: ""}
//     "1.2" parses as {Major: 1, Minor: 2, Patch: 0, Pre: "", Meta: ""}
//     "1.2.3" parses as {Major: 1, Minor: 2, Patch: 3, Pre: "", Meta: ""}
//     "1-beta" parses as {Major: 1, Minor: 0, Patch: 0, Pre: "beta", Meta: ""}
//     "1.2+abc" parses as {Major: 1, Minor: 2, Patch: 0, Pre: "", Meta: "abc"}
//     "1.2-beta+abc" parses as {Major: 1, Minor: 2, Patch: 0, Pre: "beta", Meta: "abc"}
//
// Note that whilst permissive, parse will still return errors for the following
// conditions:
//
//     UnexpectedCharacter when encountering anything other than:
//
//         [0-9\.] in a major or minor field,
//         [0-9\-]+, in a patch field,
//         [0-9a-zA-Z\-] in a prerelease or meta field.
//
//     ZeroLengthNumeric when encountering 2 dots together in the major, minor,
//     patch fields.
//
// If you want to validate that input is in exact semver 2.0.0 format, you
// should use ParseExactSemver2 instead.
func Parse(s string) (Version, error) {
	v, errs := parse(s)
	// Skip nil, LeadingZero, and VersionIncomplete errors in this
	// permissive parse func.
	for _, err := range errs {
		if err == nil {
			continue
		}
		if _, ok := err.(LeadingZero); ok {
			continue
		}
		if _, ok := err.(VersionIncomplete); ok {
			continue
		}
		return v, err
	}
	return v, nil
}

// MustParse is like Parse, but panics on errors. This is useful when
// initialising versions in the global scope.
func MustParse(s string) Version {
	v, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return v
}

// ParseExactSemver2 returns an error, and an incomplete Version if the string
// passed in does not conform exactly to semver 2.0.0. It can return the same
// errors as Parse, plus these two additional ones:
//
//     VersionIncomplete when either the minor or patch fields are missing
//
//     LeadingZero when a major, minor, or patch contains an erroneous preceding zero character.
//
func ParseExactSemver2(s string) (Version, error) {
	v, errs := parse(s)
	return v, firstErr(errs...)
}

// MustParseExactSemver2 is like ParseExactSemver2, excapt that
// it panics on errors. This is useful in when initialising version in
// the global scope.
func MustParseExactSemver2(s string) Version {
	v, err := ParseExactSemver2(s)
	if err != nil {
		panic(err)
	}
	return v
}

// ParseAny tries to parse any version found in a string. It starts
// parsing at the first decimal digit [0-9], and stops when it finds
// an invalid character. It returns an error only if there are no
// digits found in the string.
func ParseAny(s string) (Version, error) {
	startIndex := strings.IndexAny(s, digits)
	if startIndex == -1 {
		return Version{}, fmt.Errorf("no version found in %q", s)
	}
	v, _ := Parse(s[startIndex:])
	return v, nil
}

func parse(s string) (Version, []error) {
	var parsedMinor, parsedPatch, parsedPre, parsedMeta bool
	var (
		major = &bytes.Buffer{}
		minor = &bytes.Buffer{}
		patch = &bytes.Buffer{}
		pre   = &bytes.Buffer{}
		meta  = &bytes.Buffer{}
	)
	targets := map[mode]*bytes.Buffer{
		modeMajor: major,
		modeMinor: minor,
		modePatch: patch,
		modePre:   pre,
		modeMeta:  meta,
	}
	m := modeMajor
	var i int
	var c rune
	// finalise takes the current buffers and tries to return a partial version
	finalise := func(knownErrors ...error) (Version, []error) {
		var err error
		v := Version{}
		v.DefaultFormat = Major
		majorString := major.String()
		if err := validateMMPFormat(majorString, "major"); err != nil {
			knownErrors = append(knownErrors, err)
		}
		if v.Major, err = strconv.Atoi(majorString); err != nil {
			return v, append(knownErrors, err)
		}
		if parsedMinor {
			v.DefaultFormat = MajorMinor
			minorString := minor.String()
			if err := validateMMPFormat(minorString, "minor"); err != nil {
				knownErrors = append(knownErrors, err)
			}
			if v.Minor, err = strconv.Atoi(minorString); err != nil {
				return v, append(knownErrors, err)
			}
		}
		if parsedPatch {
			v.DefaultFormat = MajorMinorPatch
			patchString := patch.String()
			if err := validateMMPFormat(patchString, "patch"); err != nil {
				knownErrors = append(knownErrors, err)
			}
			if v.Patch, err = strconv.Atoi(patchString); err != nil {
				return v, append(knownErrors, err)
			}

		}
		if parsedPre {
			v.DefaultFormat = v.DefaultFormat + "-?"
		}
		if parsedMeta {
			v.DefaultFormat = v.DefaultFormat + "+?"
		}
		v.Pre = pre.String()
		v.Meta = meta.String()
		return v, knownErrors
	}
	changeMode := func() (bool, error) {
		if (m == modePre || m == modeMeta) && c == '-' {
			return false, nil
		}
		if m == modeMeta && c == '+' {
			return false, UnexpectedCharacter{c, i}
		}
		if m == modePatch && c == '.' {
			return false, UnexpectedCharacter{c, i}
		}
		if (m == modeMajor || m == modeMinor) && c == '.' {
			m++
			return true, nil
		}
		switch c {
		default:
			return false, nil
		case '-':
			m = modePre
		case '+':
			m = modeMeta
		}
		return true, nil
	}
	for i, c = range s {
		if m == modeMinor {
			parsedMinor = true
		}
		if m == modePatch {
			parsedPatch = true
		}
		if m == modePre {
			parsedPre = true
		}
		if m == modeMeta {
			parsedMeta = true
		}
		switch c {
		case '.', '-', '+':
			changed, err := changeMode()
			if err != nil {
				return finalise(err)
			}
			if changed {
				continue
			}
		}
		switch m {
		case modeMajor, modeMinor, modePatch:
			if strings.ContainsRune(digits, c) {
				targets[m].WriteRune(c)
			} else {
				return finalise(UnexpectedCharacter{c, i})
			}
		case modePre, modeMeta:
			if strings.ContainsRune(validPreAndMetaChars, c) {
				targets[m].WriteRune(c)
			} else {
				return finalise(UnexpectedCharacter{c, i})
			}
		}
	}
	if !parsedMinor {
		return finalise(VersionIncomplete{"minor"})
	}
	if !parsedPatch {
		return finalise(VersionIncomplete{"patch"})
	}
	return finalise(nil)
}

func replaceAll(s string, replacements map[string]interface{}) string {
	for what, replacement := range replacements {
		s = strings.Replace(s, what, fmt.Sprint(replacement), -1)
	}
	return s
}

func firstErr(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func validateMMPFormat(s, name string) error {
	if len(s) == 0 {
		return ZeroLengthNumeric{name}
	}
	if len(s) > 1 && s[0] == '0' {
		return LeadingZero{name, s}
	}
	return nil
}
