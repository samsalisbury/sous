package restful

import (
	"bytes"
	"encoding/json"
	"io"
)

type jsonMap map[string]interface{}

func putbackJSON(originalBuf, baseBuf, changedBuf io.Reader) *bytes.Buffer {
	var original, base, changed jsonMap
	if err := mapDecode(originalBuf, &original); err != nil {
		panic(err)
	}
	if err := mapDecode(baseBuf, &base); err != nil {
		panic(err)
	}

	if err := mapDecode(changedBuf, &changed); err != nil {
		panic(err)
	}
	original = applyChanges(base, changed, original)
	return encodeJSON(original)
}

// mutates base
func applyChanges(base, changed, target map[string]interface{}) map[string]interface{} {
	if target == nil {
		panic("nil target for applyChanges")
	}
	for k, v := range changed {
		switch v := v.(type) {
		default:
			if b, old := base[k]; !old {
				target[k] = v //created
			} else {
				delete(base, k)
				if !same(b, v) { // changed
					target[k] = v
				}
			}
		case map[string]interface{}:
			if b, old := base[k]; !old {
				target[k] = v //created
			} else {
				delete(base, k)
				// Unchecked cast: if base[k] isn't also a map, we have bigger problems.
				// If target[k] isn't a map, then the server has changed the type under us, and we should crash

				tsub := target[k]
				if tsub == nil {
					tsub = map[string]interface{}{}
				}

				newMap := applyChanges(b.(map[string]interface{}), v, tsub.(map[string]interface{}))

				target[k] = newMap
			}
		}
	}

	// the remaining fields were deleted
	for k := range base {
		delete(target, k)
	}

	return target
}

// same does a kind of limited deep equality over loosly typed values (e.g. map[string]interface{})
func same(left, right interface{}) bool {
	switch left := left.(type) {
	default:
		return left == right
	case map[string]interface{}:
		r, is := right.(map[string]interface{})
		if !is {
			return false
		}
		for lk := range left {
			rv, has := r[lk]
			if !has {
				return false
			}
			if !same(left[lk], rv) {
				return false
			}
		}
		for rk := range r {
			lv, has := left[rk]
			if !has {
				return false
			}
			if !same(lv, r[rk]) {
				return false
			}
		}
		return true
	case []interface{}:
		r, is := right.([]interface{})
		if !is {
			return false
		}
		if len(left) != len(r) {
			return false
		}
		for n := range left {
			if !same(left[n], r[n]) {
				return false
			}
		}

		return true
	}
}

func mapDecode(buf io.Reader, into *jsonMap) error {
	return json.NewDecoder(buf).Decode(into)
}

func encodeJSON(from interface{}) *bytes.Buffer {
	buf := &bytes.Buffer{}
	if err := json.NewEncoder(buf).Encode(from); err != nil {
		panic(err)
	}
	return buf
}
