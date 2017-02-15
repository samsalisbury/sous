package restful

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type (
	// QueryValues wrap url.Values to keep them needing to be re-exported
	QueryValues struct {
		url.Values
	}
)

func parseQueryValues(req *http.Request) (*QueryValues, error) {
	v, err := url.ParseQuery(req.URL.RawQuery)
	return &QueryValues{v}, err
}

// Single returns a singular value for a query field, or an error if the field
// is missing (and no default is supplied) or if multiple values are supplied
func (qv *QueryValues) Single(field string, def ...string) (string, error) {
	values, ok := qv.Values[field]
	if !ok {
		if len(def) == 0 {
			return "", errors.Errorf("No %s given", field)
		}
		return def[0], nil
	}
	if len(values) > 1 {
		return "", errors.Errorf("Multiples of %s given", field)
	}
	return values[0], nil
}
