package semv

import "fmt"

type (
	// Range is a semver range.
	Range struct {
		Min, MinEqual, Max, MaxEqual *Version
	}
)

// ParseRange attempts to parse the passed string as a semver range. It
// allows the caret ^ and tilde ~ prefixes, as used by NPM, and also
// >, >=, <, <= as prefixes to indicate greater than, greater than or
// equal to, less than, and less than or equal to, respectively.
func ParseRange(s string) (Range, error) {
	if len(s) == 0 {
		return Range{}, fmt.Errorf("cannot parse range from empty string")
	}
	v, err := ParseAny(s)
	if err != nil {
		return Range{}, err
	}
	if len(s) >= 2 {
		switch s[:2] {
		case "==":
			return EqualTo(v), nil
		case ">=":
			return GreaterThanOrEqualTo(v), nil
		case "<=":
			return LessThanOrEqualTo(v), nil
		}
	}
	switch s[0] {
	case '=', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return EqualTo(v), nil
	case '>':
		return GreaterThan(v), nil
	case '<':
		return LessThan(v), nil
	case '~':
		max := v.IncrementMinor()
		return GreaterThanOrEqualToAndLessThan(v, max), nil
	case '^':
		max := v.IncrementMajor()
		return GreaterThanOrEqualToAndLessThan(v, max), nil
	}
	return Range{}, fmt.Errorf("unable to parse version range %q", s)
}

// MustParseRange is similar to ParseRange except that it panics instead
// of returning an error.
func MustParseRange(s string) Range {
	r, err := ParseRange(s)
	if err != nil {
		panic(err)
	}
	return r
}

// GreaterThan returns a range satisfied by any version greater than
// the version passed in, according to semver 2.0.0 precedence rules.
func GreaterThan(v Version) Range {
	return Range{Min: &v}
}

// LessThan returns a range satisfied by any version less than the
// version passed in, according to semver 2.0.0 precedence rules.
func LessThan(v Version) Range {
	return Range{Max: &v}
}

// EqualTo returns a range satisfied only by the specific version passed
// in, according to semver 2.0.0 precedence rules.
func EqualTo(v Version) Range {
	return Range{MinEqual: &v, MaxEqual: &v}
}

// GreaterThanOrEqualTo is similar to GreaterThan, but the Range returned
// is additionally satisfied by versions exactly equal to the version
// passed in, according to semver 2.0.0 precedence rules.
func GreaterThanOrEqualTo(v Version) Range {
	return Range{MinEqual: &v}
}

// LessThanOrEqualTo is similar to LessThan, but the Range returned
// is additionally satisfied by versions exactly equal to the version
// passed in, according to semver 2.0.0 precedence rules.
func LessThanOrEqualTo(v Version) Range {
	return Range{MaxEqual: &v}
}

// GreaterThanOrEqualToAndLessThan returns Range that is satisfied by
// versions greater than or equal to the first, and less than the second,
// of the versions passed in, according to semver 2.0.0 precedence rules.
func GreaterThanOrEqualToAndLessThan(min, lessThan Version) Range {
	return Range{MinEqual: &min, Max: &lessThan}
}

// SatisfiedBy returns true if the version passed in fits inside the range
// the method is invoked on.
func (r Range) SatisfiedBy(v Version) bool {
	// TODO: This passes tests but is pretty horrible. Refactor it for readability.
	if r.Min != nil {
		if !r.Min.Less(v) {
			return false
		} else if v.IsPrerelease() && (!r.Min.IsPrerelease() || r.Min.MMPLess(v)) {
			return false
		}
	}
	if r.MinEqual != nil {
		if !v.Equals(*r.MinEqual) && !r.MinEqual.Less(v) {
			return false
		} else if v.IsPrerelease() && (!r.MinEqual.IsPrerelease() && !r.MinEqual.MMPLess(v) && !r.MinEqual.MMPEqual(v)) {
			return false
		}
	}

	if r.Max != nil {
		if !v.Less(*r.Max) {
			return false
		} else if v.IsPrerelease() && (!r.Max.IsPrerelease() || !v.MMPLess(*r.Max)) {
			return false
		}
	}
	if r.MaxEqual != nil {
		if !v.Equals(*r.MaxEqual) && !v.Less(*r.MaxEqual) {
			return false
		} else if v.IsPrerelease() && !r.MaxEqual.IsPrerelease() && !v.MMPLess(*r.MaxEqual) && !v.MMPEqual(*r.MaxEqual) {
			return false
		}
	}

	return true
}

// String returns the minimal string representation of this range. For example,
// the range ">=1.0.0 <2.0.0" is compressed to "^1.0.0"
func (r Range) String() string {
	// Special case for exact equality range
	if r.MinEqual != nil && r.MaxEqual != nil && r.MaxEqual.Equals(*r.MinEqual) {
		return r.MinEqual.String()
	}
	// Special case for tilde and caret ranges
	if r.MinEqual != nil && r.Max != nil {
		if r.Max.Equals(r.MinEqual.IncrementMajor()) {
			return "^" + r.MinEqual.String()
		}
		if r.Max.Equals(r.MinEqual.IncrementMinor()) {
			return "~" + r.MinEqual.String()
		}
	}
	// All other cases
	out := ""
	if r.Min != nil {
		out = ">" + r.Min.String()
	} else if r.MinEqual != nil {
		out = ">=" + r.MinEqual.String()
	}
	if r.Max != nil {
		if out != "" {
			out += " "
		}
		out += "<" + r.Max.String()
	} else if r.MaxEqual != nil {
		if out != "" {
			out += " "
		}
		out += "<=" + r.MaxEqual.String()
	}
	return out
}

// Equals returns true if the range passed in is semantically equivalent to the
// range it is invoked on. (That is, if the same set of versions satisfies each
// range.)
func (r Range) Equals(other Range) bool {
	return r.Min.ValueEquals(other.Min) &&
		r.Max.ValueEquals(other.Max) &&
		r.MinEqual.ValueEquals(other.MinEqual) &&
		r.MaxEqual.ValueEquals(other.MaxEqual)
}
