package sous

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/samsalisbury/semv"
)

func TestHTTPNameInserter(t *testing.T) {

	reqd := false
	h := func(rw http.ResponseWriter, r *http.Request) {
		if meth := r.Method; strings.ToUpper(meth) != "PUT" {
			t.Errorf("Method should be PUT was: %s", meth)
		}
		if path := r.URL.Path; path != "/artifact" {
			t.Errorf("Path should be '/artifact' but was: %s", path)
		}
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Error(err)
		}
		if len(body) <= 0 {
			t.Errorf("Empty body")
		}
		rw.WriteHeader(200)
		reqd = true
	}

	srv := httptest.NewServer(http.HandlerFunc(h))

	hni, err := NewHTTPNameInserter(srv.URL)
	if err != nil {
		t.Error(err)
	}
	err = hni.Insert(
		SourceID{Location: SourceLocation{Repo: "a-repo", Dir: "offset"}, Version: semv.MustParse("5.5.5")},
		"dockerthin.com/repo/latest",
		"",
		[]Quality{},
	)
	if err != nil {
		t.Error(err)
	}
	if !reqd {
		t.Errorf("No request issued")
	}
}
