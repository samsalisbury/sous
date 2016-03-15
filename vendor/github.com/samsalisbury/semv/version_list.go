package semv

import "sort"

// VersionList is a slice of Versions. It implements sort.Interface, so you can use
// that to do in-place sorting of the versions (default order is lowest to highest).
// It also provides some features for version selection (see GreatestSatisfying), and
// provides copy semantics via Clone, Sorted, and SortedDesc.
type VersionList []Version

// ParseList takes a slice of strings and attempts to parse each
// one using Parse, adding each successfully parsed version to the resultant
// VersionList. If any of them fail to parse, the failure error is
// returned along with an empty list.
func ParseList(versions ...string) (VersionList, error) {
	return parseList(Parse, versions)
}

// ParseExactSemver2List is similar to ParseList, except it uses the
// ParseExactSemver2 func to parse each version string.
func ParseExactSemver2List(versions ...string) (VersionList, error) {
	return parseList(ParseExactSemver2, versions)
}

// MustParseList is similar to ParseList, except that it panics on parse errors.
func MustParseList(versions ...string) VersionList {
	vl, err := ParseList(versions...)
	if err != nil {
		panic(err)
	}
	return vl
}

// MustParseExactSemver2List is similar to ParseExactSemver2List, except that
// it panics on parse errors.
func MustParseExactSemver2List(versions ...string) VersionList {
	vl, err := ParseExactSemver2List(versions...)
	if err != nil {
		panic(err)
	}
	return vl
}

// parseList is used internally to provide generic version list parsing.
func parseList(parseFunc func(string) (Version, error), versions []string) (VersionList, error) {
	vl := make(VersionList, len(versions))
	for i, s := range versions {
		v, err := parseFunc(s)
		if err != nil {
			return nil, err
		}
		vl[i] = v
	}
	return vl, nil
}

// Len returns the number of versions in this list.
func (vl VersionList) Len() int { return len(vl) }

// Swap is used by sort to swap elements in the list.
func (vl VersionList) Swap(i, j int) { vl[i], vl[j] = vl[j], vl[i] }

// Less indicates whether or not the Version at index i is less than that at
// index j.
func (vl VersionList) Less(i, j int) bool { return vl[i].Less(vl[j]) }

// Sorted is similar to Clone, but it additionally sorts the copy of the list
// from lowest to highest versions.
func (vl VersionList) Sorted() VersionList {
	newVL := vl.Clone()
	sort.Sort(newVL)
	return newVL
}

// Clone returns an itemwise copy of this VersionList
func (vl VersionList) Clone() VersionList {
	newVL := make(VersionList, len(vl))
	for i := range vl {
		newVL[i] = vl[i]
	}
	return newVL
}

// SortedDesc is similar to Sorted, except the returned copy of the list is sorted in
// the opposite direction.
func (vl VersionList) SortedDesc() VersionList {
	newVL := vl.Sorted()
	sort.Sort(sort.Reverse(newVL))
	return newVL
}

// GreatestSatisfying returns the greatest (highest) version contained in the
// VersionList, which satisfies the passed Range. If none are found that satisfy
// the range, the second return value is false, otherwise it is true.
func (vl VersionList) GreatestSatisfying(r Range) (Version, bool) {
	sortedCopy := vl.SortedDesc()
	for _, v := range sortedCopy {
		if r.SatisfiedBy(v) {
			return v, true
		}
	}
	return Version{}, false
}
