package sqlgen

type (
	row map[string]field

	field struct {
		column *coldef
		values []interface{}
	}
)

func (r row) equal(other row) bool {
	for n, v := range r {
		if !other[n].equal(v) {
			return false
		}
	}
	for n, v := range other {
		if !r[n].equal(v) {
			return false
		}
	}
	return true
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
