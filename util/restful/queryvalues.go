package restful

import (
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

type (
	// QueryValues wrap url.Values to provide a few convenience methods
	QueryValues struct {
		url.Values
	}

	// QueryParser is a convenience embed for parsing query values
	QueryParser struct{}
)

// ParseQuery parses query values out of a url, returning an empty list of values in the event of an error
func (QueryParser) ParseQuery(req *http.Request) QueryValues {
	v, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		v = url.Values{}
	}
	return QueryValues{v}
}

// Single returns a singular value for a query field, or an error if the field
// is missing (and no default is supplied) or if multiple values are supplied
func (qv QueryValues) Single(field string, def ...string) (string, error) {
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
