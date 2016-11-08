package server

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/nyarly/testify/assert"
	"github.com/nyarly/testify/suite"
	"github.com/opentable/sous/config"
	"github.com/opentable/sous/graph"
	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/psyringe"
)

type (
	TestResource struct {
		Data string
	}

	TestGetExchanger struct {
		*TestResource

		httprouter.Params
		*QueryValues
	}
	TestPutExchanger struct {
		*TestResource

		*http.Request
		httprouter.Params
		*QueryValues
	}

	TestData struct {
		Data, Name, Extra string
	}
)

func (tr *TestResource) Get() Exchanger { return &TestGetExchanger{TestResource: tr} }
func (tr *TestResource) Put() Exchanger { return &TestPutExchanger{TestResource: tr} }

func (ge *TestGetExchanger) Exchange() (interface{}, int) {
	p := ge.Params.ByName("param")
	if p == "missing" {
		return TestData{}, 404
	}
	return TestData{ge.TestResource.Data, p, ge.QueryValues.Get("extra")}, 200
}

func (ge *TestPutExchanger) Exchange() (interface{}, int) {
	var data TestData
	dec := json.NewDecoder(ge.Request.Body)
	if err := dec.Decode(&data); err != nil {
		return err, http.StatusBadRequest
	}
	ge.TestResource.Data = data.Data

	return struct{ Data, Name, Extra string }{
		ge.TestResource.Data,
		ge.Params.ByName("param"),
		ge.QueryValues.Get("extra"),
	}, 200
}

func testRouteMap() *RouteMap {
	return &RouteMap{
		{"test", "/test/:param", &TestResource{"base"}},
	}
}

func justBytes(b []byte, e error) io.ReadCloser {
	if e != nil {
		return nil
	}
	return ioutil.NopCloser(bytes.NewBuffer(b))
}

func TestOverallRouter(t *testing.T) {
	assert := assert.New(t)

	gf := func() Injector {
		g := graph.BuildGraph(&bytes.Buffer{}, os.Stdout, os.Stdout)
		g.Add(&config.Verbosity{})
		return g
	}
	ts := httptest.NewServer(SousRouteMap.BuildRouter(gf))
	defer ts.Close()

	res, err := http.Get(ts.URL + "/gdm")
	assert.NoError(err)
	gdm, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	assert.NoError(err)
	assert.Regexp(`"Deployments"`, string(gdm))
	assert.NotEqual(res.Header.Get("Etag"), "")
}

type PutConditionalsSuite struct {
	suite.Suite
	server *httptest.Server
	client *http.Client
}

func (t *PutConditionalsSuite) SetupTest() {
	dif := func() Injector { return psyringe.New(sous.SilentLogSet) }
	t.server = httptest.NewServer(testRouteMap().BuildRouter(dif))

	t.client = &http.Client{}
}

func (t *PutConditionalsSuite) TeardownTest() {
	t.server.Close()
}

func (t *PutConditionalsSuite) testReq(method, path string, data interface{}) *http.Request {
	body := justBytes(json.Marshal(data))
	t.Require().NotNil(body)
	req, err := http.NewRequest("PUT", t.server.URL+path, body)
	t.NoError(err)
	return req
}

func (t *PutConditionalsSuite) TestPutConditionalsNoneMatch() {
	req := t.testReq("PUT", "/test/missing?extra=two", TestData{"new", "zebra", "two"})
	req.Header.Add("If-None-Match", "*")
	res, err := t.client.Do(req)
	t.NoError(err)
	t.Equal(res.Status, "200 OK")
}

func (t *PutConditionalsSuite) TestPutConditionalsNoneMatchRejected() {
	req := t.testReq("PUT", "/test/one?extra=two", TestData{"new", "zebra", "two"})
	req.Header.Add("If-None-Match", "*")
	res, err := t.client.Do(req)
	t.NoError(err)
	t.Equal(res.Status, "412 Precondition Failed")
}

func (t *PutConditionalsSuite) TestPutConditionals() {
	var td TestData

	res, err := http.Get(t.server.URL + "/test/one?extra=two")
	t.NoError(err)
	dec := json.NewDecoder(res.Body)
	t.NoError(dec.Decode(&td))
	res.Body.Close()

	t.Equal(TestData{"base", "one", "two"}, td)
	etag := res.Header.Get("Etag")
	t.NotEqual(etag, "")

	req := t.testReq("PUT", "/test/one?extra=two", TestData{"changed", "one", "two"})
	res, err = t.client.Do(req)
	t.NoError(err)
	t.Equal(res.Status, "428 Precondition Required")
}

func (t *PutConditionalsSuite) TestPutConditionalsMatched() {
	res, err := http.Get(t.server.URL + "/test/one?extra=two")
	t.NoError(err)
	var td TestData
	dec := json.NewDecoder(res.Body)
	t.NoError(dec.Decode(&td))
	res.Body.Close()
	t.Equal(TestData{"base", "one", "two"}, td)
	etag := res.Header.Get("Etag")

	req := t.testReq("PUT", "/test/one?extra=two", TestData{"changed", "one", "two"})
	req.Header.Add("If-Match", etag)
	res, err = t.client.Do(req)
	t.NoError(err)
	t.Equal(res.Status, "200 OK")
}

func (t *PutConditionalsSuite) TestPutConditionalsMatchedRejected() {
	req := t.testReq("PUT", "/test/one?extra=two", TestData{"changed", "one", "two"})
	req.Header.Add("If-Match", "blarglearglebarg")
	res, err := t.client.Do(req)
	t.NoError(err)
	t.Equal(res.Status, "412 Precondition Failed")
}

func TestPutConditionals(t *testing.T) {
	suite.Run(t, new(PutConditionalsSuite))
}
