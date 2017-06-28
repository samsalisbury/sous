package restful

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

const canarySuffix = `": "canary",`

func calcContentLength(buf *bytes.Buffer, etag string) uint {
	return uint(buf.Len() + len(etag) + len(canarySuffix) + 1) //1 for '"'
}

// InjectCanaryAttr injects an attribute into an io.Reader that contains JSON
func InjectCanaryAttr(json io.Reader, attrName string) io.Reader {
	var err error
	b := make([]byte, 1)

	for err == nil {
		_, err = json.Read(b)
		if b[0] == '{' {
			break
		}
	}

	// 'JSON' doesn't include a '{'
	if err != nil {
		panic(err)
	}

	prefix := bytes.NewReader([]byte(`{"` + attrName + canarySuffix))
	return io.MultiReader(prefix, json)
}

func (mh *MetaHandler) validCanaryAttr(w http.ResponseWriter, r *http.Request, etag string) bool {
	rbytes, err := ioutil.ReadAll(r.Body)

	if err != nil {
		mh.writeHeaders(http.StatusInternalServerError, w, r, fmt.Sprintf("Error getting body from request: %v", err))
		return false
	}

	rbody := bytes.NewReader(rbytes)
	dump := map[string]interface{}{}

	if err := json.NewDecoder(rbody).Decode(&dump); err != nil {
		mh.writeHeaders(http.StatusBadRequest, w, r, fmt.Sprintf("Error parsing JSON: %v\n\nIt is possible that your client is out of date. Try updating.", err))
		return false
	}

	if canary, has := dump[etag]; !has || canary != "canary" {
		mh.writeHeaders(http.StatusBadRequest, w, r, fmt.Sprintf("Absent or malformed canary field: %q\n\nIt is possible that your client is out of date. Try updating.", canary))
		return false
	}

	off, err := rbody.Seek(0, io.SeekStart)
	if off != 0 || err != nil {
		mh.writeHeaders(http.StatusInternalServerError, w, r, fmt.Sprintf("Error manipulating a buffer: offset %d, error %v", off, err))
		return false
	}
	r.Body = ioutil.NopCloser(rbody)
	return true
}
