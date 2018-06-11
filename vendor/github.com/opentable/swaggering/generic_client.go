package swaggering

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/opentable/sous/util/logging"
	"github.com/opentable/sous/util/logging/messages"
)

type (
	// Requester defines the interface that Swaggering uses to
	// make actual HTTP requests of the API server
	Requester interface {
		// DTORequest performs an HTTP request and populates a DTO based on the response
		DTORequest(resourceName string, dto DTO, method, path string, pathParams, queryParams UrlParams, body ...DTO) error

		// Request performs an HTTP request and returns the body of the response
		Request(resourceName string, method string, path string, pathParams UrlParams, queryParams UrlParams, body ...DTO) (io.ReadCloser, error)
	}

	// GenericClient is a generic client for Swagger described services
	GenericClient struct {
		BaseURL string
		Logger  logging.LogSink
		HTTP    http.Client
	}

	// ReqError represents failures from requests
	ReqError struct {
		Method, Path string
		Message      string
		Status       int
		Body         bytes.Buffer
	}

	UrlParams map[string]interface{}
)

func (e *ReqError) Error() string {
	return fmt.Sprintf("%s %s => %s: \n%s", e.Method, e.Path, e.Message, e.Body.String())
}

// DTORequest performs an HTTP request and populates a DTO based on the response
func (gc *GenericClient) DTORequest(resourceName string, pop DTO, method, path string, pathParams, queryParams UrlParams, body ...DTO) (err error) {
	resBody, err := gc.Request(resourceName, method, path, pathParams, queryParams, body...)
	if err != nil {
		return
	}
	err = pop.Populate(resBody)
	return
}

// Request performs an HTTP request and returns the body of the response
func (gc *GenericClient) Request(resourceName, method, path string, pathParams, queryParams UrlParams, body ...DTO) (resBody io.ReadCloser, err error) {
	req, err := gc.buildRequest(method, path, pathParams, queryParams, body...)
	if err != nil {
		return
	}
	start := time.Now()
	res, err := gc.HTTP.Do(req)
	if err != nil {
		return
	}

	messages.ReportClientHTTPResponse(gc.Logger, "GenericClient", res, resourceName, time.Now().Sub(start))

	if res.StatusCode > 299 {
		rerr := &ReqError{
			Status:  res.StatusCode,
			Message: res.Status,
			Method:  method,
			Path:    path,
			Body:    bytes.Buffer{},
		}
		rerr.Body.ReadFrom(res.Body)
		res.Body.Close()
		err = rerr
		return
	}
	return res.Body, nil
}

func (gc *GenericClient) buildRequest(method, path string, pathParams, queryParams UrlParams, bodies ...DTO) (req *http.Request, err error) {
	url, err := url.Parse(gc.BaseURL)
	if err != nil {
		return
	}

	path, err = pathRender(path, pathParams)
	if err != nil {
		return
	}
	url.Path = strings.Join([]string{strings.TrimRight(url.Path, "/"), strings.TrimLeft(path, "/")}, "/")

	q := url.Query()
	for k, v := range queryParams {
		q.Set(k, fmt.Sprintf("%v", v))
	}
	url.RawQuery = q.Encode()

	if len(bodies) > 0 {
		req, err = gc.buildBodyRequest(method, url.String(), bodies[0])
	} else {
		req, err = gc.buildBodilessRequest(method, url.String())
	}
	return
}

func (gc *GenericClient) buildBodilessRequest(method, path string) (req *http.Request, err error) {
	req, err = http.NewRequest(method, path, nil)
	return
}

func (gc *GenericClient) buildBodyRequest(method, path string, bodyObj DTO) (req *http.Request, err error) {
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	enc.Encode(bodyObj) // XXX Here, consider a goroutine and a PipeWriter
	req, err = http.NewRequest(method, path, &buf)
	req.Header.Add("Content-Type", "application/json")
	return
}

func pathRender(path string, params UrlParams) (res string, err error) {
	parmRE := regexp.MustCompile(`{([^}]+)}`)
	building := make([]byte, 0, 150)
	pathBytes := []byte(path)

	start := 0
	indices := parmRE.FindAllSubmatchIndex(pathBytes, -1)
	for _, matches := range indices {
		building = append(building, pathBytes[start:matches[0]]...)
		start = matches[len(matches)-1] + 1
		parmName := pathBytes[matches[2]:matches[3]]
		val, ok := params[string(parmName)]
		if !ok {
			err = fmt.Errorf("Path parameter %q not provided for %s", parmName, path)
			return
		}
		building = append(building, fmt.Sprintf("%v", val)...)
	}
	building = append(building, pathBytes[start:]...)
	res = string(building)

	return
}

func (gc *GenericClient) buildURL(subpath string) (url string) {
	return strings.Join([]string{gc.BaseURL, subpath}, "/")
}
