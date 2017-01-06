package sous

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPStateManager_Create(t *testing.T) {
	reqd := false
	h := func(rw http.ResponseWriter, r *http.Request) {
		if meth := r.Method; strings.ToUpper(meth) != "PUT" {
			t.Errorf("Method should be PUT was: %s", meth)
		}
		if inm := r.Header.Get("If-None-Match"); inm != "*" {
			t.Errorf("If-None-Match header should be '*', was %s", inm)
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

	cl, err := NewClient(srv.URL)
	if err != nil {
		t.Error(err)
	}
	hsm := NewHTTPStateManager(cl)
	hsm.create(&Manifest{})
	if !reqd {
		t.Errorf("No request issued")
	}
}

func TestHTTPStateManager_Delete(t *testing.T) {
	reqd := false
	etag := "w/sauce"
	h := func(rw http.ResponseWriter, r *http.Request) {
		method := strings.ToUpper(r.Method)
		switch method {
		default:
			t.Errorf("Method should be GET or DELETE was: %s", method)
		case "DELETE":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if len(body) != 0 {
				t.Errorf("Non-empty body: %q", body)
			}
			if det := r.Header.Get("If-Match"); det != etag {
				t.Errorf("DELETE without If-Match")
			}
			rw.WriteHeader(200)
			reqd = true
		case "GET":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if len(body) != 0 {
				t.Errorf("Non-empty body: %q", body)
			}
			rw.Header().Add("Etag", etag)
			rw.WriteHeader(200)
			reqd = true
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(h))

	cl, err := NewClient(srv.URL)
	if err != nil {
		t.Error(err)
	}
	hsm := NewHTTPStateManager(cl)
	hsm.del(&Manifest{})
	if !reqd {
		t.Errorf("No request issued")
	}
}

func TestHTTPStateManager_Modify(t *testing.T) {
	//Log.Vomit.SetOutput(os.Stderr)
	reqd := false
	etag := "w/sauce"
	h := func(rw http.ResponseWriter, r *http.Request) {
		method := strings.ToUpper(r.Method)
		switch method {
		default:
			t.Errorf("Method should be GET or PUT was: %s", method)
		case "PUT":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if len(body) <= 0 {
				t.Errorf("Empty body")
			}
			if det := r.Header.Get("If-Match"); det != etag {
				t.Errorf("DELETE without If-Match")
			}
			rw.WriteHeader(200)
			reqd = true
		case "GET":
			body, err := ioutil.ReadAll(r.Body)
			if err != nil {
				t.Error(err)
			}
			if len(body) != 0 {
				t.Errorf("Non-empty body: %q", body)
			}
			rw.Header().Add("Etag", etag)
			rw.Write([]byte("{}"))
			rw.WriteHeader(200)
		}
	}

	srv := httptest.NewServer(http.HandlerFunc(h))

	cl, err := NewClient(srv.URL)
	if err != nil {
		t.Error(err)
	}
	hsm := NewHTTPStateManager(cl)
	err = hsm.modify(&ManifestPair{
		Prior: &Manifest{},
		Post:  &Manifest{},
	})
	if err != nil {
		t.Errorf("Received error: %+v", err)
	}
	if !reqd {
		t.Errorf("No request issued")
	}
}
