package restful

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/util/readdebugger"
	"github.com/samsalisbury/psyringe"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
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

func TestRenderDataCanaries(t *testing.T) {
	rr := httptest.NewRecorder()
	ph := &StatusMiddleware{
		logSet: &silentLogSet{},
	}
	mh := &MetaHandler{
		//graphFac:      grf,
		//router:        r,
		statusHandler: ph,
	}
	rq := httptest.NewRequest("GET", "/somewhere", nil)
	data := map[string]string{"a": "b"}

	mh.renderData(200, rr, rq, data)

	rz := rr.Result()
	bodyB, err := ioutil.ReadAll(rz.Body)
	assert.NoError(t, err)

	dump := map[string]interface{}{}
	assert.NoError(t, json.Unmarshal(bodyB, &dump))

	etag := rz.Header.Get("Etag")
	assert.NotZero(t, etag)

	assert.Contains(t, dump, etag)
	assert.Equal(t, dump[etag].(string), "canary")
}

type PutConditionalsSuite struct {
	suite.Suite
	server *httptest.Server
	client *http.Client
}

func (t *PutConditionalsSuite) SetupTest() {
	dif := func() Injector { return psyringe.New() }
	t.server = httptest.NewServer(testRouteMap().BuildRouter(dif, &fallbackLogger{}))

	t.client = &http.Client{}
}

func (t *PutConditionalsSuite) TeardownTest() {
	t.server.Close()
}

func (t *PutConditionalsSuite) testReq(method, path string, data interface{}) *http.Request {
	body := justBytes(json.Marshal(data))
	t.Require().NotNil(body)
	req, err := http.NewRequest(method, t.server.URL+path, body)
	t.NoError(err)
	return req
}

func (t *PutConditionalsSuite) TestOptionsAllowCORS() {
	req := t.testReq("OPTIONS", "/test/one", nil)
	req.Header.Add("Origin", "test-client.example.com")
	res, err := t.client.Do(req)
	t.NoError(err)
	t.Equal(res.Status, "200 OK")
	t.Equal(res.Header.Get("Access-Control-Allow-Origin"), "test-client.example.com")
	t.T().Log(res.Header)
	methods := res.Header.Get("Access-Control-Allow-Methods")
	t.Regexp("GET", methods)
	t.Regexp("HEAD", methods)
	t.Regexp("PUT", methods)
	t.Regexp("OPTIONS", methods)
}

func (t *PutConditionalsSuite) TestGetAllowCORS() {
	req := t.testReq("GET", "/test/one", nil)
	req.Header.Add("Origin", "test-client.example.com")
	res, err := t.client.Do(req)
	t.NoError(err)
	t.Equal("200 OK", res.Status)
	bb, _ := ioutil.ReadAll(res.Body)
	t.T().Log("Response body: ", string(bb))
	t.Equal("*", res.Header.Get("Access-Control-Allow-Origin"))
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
	dec := json.NewDecoder(readdebugger.New(res.Body, func(b []byte, n int, e error) {
		//spew.Dump(b, n, e)
	}))
	t.NoError(dec.Decode(&td))
	res.Body.Close()
	t.Equal(TestData{"base", "one", "two"}, td)
	etag := res.Header.Get("Etag")

	req := t.testReq("PUT", "/test/one?extra=two", map[string]interface{}{
		etag:    "canary",
		"Data":  "changed",
		"Name":  "one",
		"Extra": "two",
	})
	req.Header.Add("If-Match", etag)
	res, err = t.client.Do(req)
	t.NoError(err)
	t.Equal(res.Status, "200 OK")
}

func (t *PutConditionalsSuite) TestPutConditionalsWithoutCanaryIsRejected() {
	res, err := http.Get(t.server.URL + "/test/one?extra=two")
	t.NoError(err)
	var td TestData
	dec := json.NewDecoder(readdebugger.New(res.Body, func(b []byte, n int, e error) {
		//spew.Dump(b, n, e)
	}))
	t.NoError(dec.Decode(&td))
	res.Body.Close()
	t.Equal(TestData{"base", "one", "two"}, td)
	etag := res.Header.Get("Etag")

	req := t.testReq("PUT", "/test/one?extra=two", map[string]interface{}{
		"Data":  "changed",
		"Name":  "one",
		"Extra": "two",
	})
	req.Header.Add("If-Match", etag)
	res, err = t.client.Do(req)
	t.NoError(err)
	t.Equal(res.StatusCode, 400)
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
