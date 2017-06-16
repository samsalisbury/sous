package sous

import (
	"bytes"
	"encoding/json"
	"io"
)

type jsonMap map[string]interface{}

func putbackJSON(originalBuf, baseBuf, changedBuf io.Reader) *bytes.Buffer {
	var original, base, changed jsonMap
	mapDecode(originalBuf, &original)
	mapDecode(baseBuf, &base)
	mapDecode(changedBuf, &changed)
	original = applyChanges(base, changed, original)
	return encodeJSON(original)
}

// mutates base
func applyChanges(base, changed, target map[string]interface{}) map[string]interface{} {
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
				newMap := applyChanges(b.(map[string]interface{}), v, target[k].(map[string]interface{}))

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

func same(left, right interface{}) bool {
	switch left := left.(type) {
	default:
		return left == right
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
	dec := json.NewDecoder(buf)
	return dec.Decode(into)
}

func encodeJSON(from interface{}) *bytes.Buffer {
	buf := &bytes.Buffer{}
	enc := json.NewEncoder(buf)
	enc.Encode(from)
	return buf
}
