// This package just provides default config for the external
// yaml library.
package yaml

// Awaiting merge of https://github.com/go-yaml/yaml/pull/149
// before going back to the upstream repo.
import y "github.com/samsalisbury/yaml"

func Marshal(in interface{}) ([]byte, error) {
	return y.Marshal(in, y.OPT_NOLOWERCASE)
}

func Unmarshal(in []byte, out interface{}) error {
	return y.Unmarshal(in, out, y.OPT_NOLOWERCASE)
}
