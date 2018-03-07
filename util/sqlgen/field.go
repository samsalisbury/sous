package sqlgen

type (
	row map[string]field

	field struct {
		column *coldef
		values []interface{}
	}
)

func (r row) dupes(other row) bool {
	hasCandidateField := false
	for n, v := range r {
		if !v.column.candidate {
			continue
		}
		hasCandidateField = true
		if !other[n].equal(v) {
			return false
		}
	}
	for n, v := range other {
		if !v.column.candidate {
			continue
		}
		hasCandidateField = true
		if !r[n].equal(v) {
			return false
		}
	}
	return hasCandidateField
}

func (f field) equal(other field) bool {
	if len(f.values) != len(other.values) {
		return false
	}

	for i, v := range f.values {
		if other.values[i] != v {
			return false
		}
	}

	return true
}
