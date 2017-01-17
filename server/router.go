package server

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"

	"github.com/julienschmidt/httprouter"
	"github.com/opentable/sous/lib"
	"github.com/samsalisbury/psyringe"
)

type (
	// The MetaHandler collects common behavior for route handlers.
	MetaHandler struct {
		router                     *httprouter.Router
		processGraph, requestGraph Injector
		statusHandler              *StatusMiddleware
	}

	// ResponseWriter wraps the the http.ResponseWriter interface.
	// XXX This is a workaround for Psyringe
	ResponseWriter struct {
		http.ResponseWriter
	}

	// ExchangeLogger wraps and logs the exchange.
	ExchangeLogger struct {
		Exchanger Exchanger
		*sous.LogSet
		*http.Request
		httprouter.Params
	}

	// Injector is an interface for DI systems.
	Injector interface {
		Inject(...interface{}) error
		MustInject(...interface{})
		Add(...interface{})
	}
	// A GraphFactory builds a SousGraph.
	GraphFactory func() Injector
)

// Exchange implements Exchanger on ExchangeLogger.
func (xlog *ExchangeLogger) Exchange() (data interface{}, status int) {
	xlog.Vomit.Printf("Server: <- %s %s params: %v", xlog.Method, xlog.URL.String(), xlog.Params)
	data, status = xlog.Exchanger.Exchange()
	xlog.Vomit.Printf("Server: -> %d: %#v", status, data)
	return
}

// GetHandling handles Get requests.
func (mh *MetaHandler) GetHandling(factory ExchangeFactory) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		h := mh.injectedHandler(factory, w, r, p)
		data, status := h.Exchange()
		mh.renderData(status, w, r, data)
	}
}

// DeleteHandling handles Delete requests.
func (mh *MetaHandler) DeleteHandling(factory ExchangeFactory) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		h := mh.injectedHandler(factory, w, r, p)
		_, status := h.Exchange()
		mh.renderData(status, w, r, nil)
	}
}

// HeadHandling handles Head requests.
func (mh *MetaHandler) HeadHandling(factory ExchangeFactory) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		h := mh.injectedHandler(factory, w, r, p)
		_, status := h.Exchange()
		mh.writeHeaders(status, w, r, nil)
	}
}

// PutHandling handles PUT requests.
func (mh *MetaHandler) PutHandling(factory ExchangeFactory) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		if r.Header.Get("If-Match") == "" && r.Header.Get("If-None-Match") == "" {
			mh.writeHeaders(http.StatusPreconditionRequired, w, r, "")
			return
		}

		gr := copyRequest(r)
		gr.Method = "GET"
		grez := mh.synthResponse(gr)

		if r.Header.Get("If-None-Match") == "*" && grez.StatusCode != 404 {
			mh.writeHeaders(http.StatusPreconditionFailed, w, r, "resource present for If-None-Match=*!")
			return
		}
		if etag := r.Header.Get("If-Match"); etag != "" {
			grezEtag := grez.Header.Get("Etag")
			if grezEtag != etag {
				reqBody, _ := ioutil.ReadAll(r.Body)
				rezBody, _ := ioutil.ReadAll(grez.Body)
				mh.writeHeaders(http.StatusPreconditionFailed, w, r,
					fmt.Sprintf("Etag mismatch: provided %q != existing %q\n%s\n\n%s",
						etag, grezEtag, string(reqBody), string(rezBody)))
				return
			}
		}
		h := mh.injectedHandler(factory, w, r, p)
		data, status := h.Exchange()
		mh.renderData(status, w, r, data)
	}
}

// InstallPanicHandler installs an panic handler into the router.
func (mh *MetaHandler) InstallPanicHandler() {
	mh.processGraph.Inject(mh.statusHandler)
	mh.router.PanicHandler = func(w http.ResponseWriter, r *http.Request, recovered interface{}) {
		mh.statusHandler.HandlePanic(w, r, recovered)
	}

}

// ExchangeGraph returns a per-request Injector.
func (mh *MetaHandler) ExchangeGraph(w http.ResponseWriter, r *http.Request, p httprouter.Params) Injector {
	// This Clone is very important so we get a fresh injector for each request.
	// This type assertion is nasty, going away soon. - Sam
	requestGraph := mh.requestGraph.(*psyringe.Psyringe).Clone()
	requestGraph.Add(&ResponseWriter{ResponseWriter: w}, r, p, parseQueryValues)
	return requestGraph
}

func (mh *MetaHandler) injectedHandler(factory ExchangeFactory, w http.ResponseWriter, r *http.Request, p httprouter.Params) Exchanger {
	h := factory()

	exGraph := mh.ExchangeGraph(w, r, p)
	// Inject process-scoped stuff.
	mh.processGraph.MustInject(h)
	// Inject request-scoped stuff.
	exGraph.MustInject(h)

	logger := &ExchangeLogger{}
	mh.processGraph.MustInject(logger)
	exGraph.MustInject(logger)
	logger.Exchanger = h

	return logger
}

func (mh *MetaHandler) writeHeaders(status int, w http.ResponseWriter, r *http.Request, data interface{}) {
	mh.statusHandler.HandleResponse(status, r, w, data)
}

func (mh *MetaHandler) renderData(status int, w http.ResponseWriter, r *http.Request, data interface{}) {
	if data == nil || status >= 300 {
		mh.writeHeaders(status, w, r, data)
		return
	}

	buf := &bytes.Buffer{}
	digest := md5.New()
	// xxx conneg
	e := json.NewEncoder(io.MultiWriter(buf, digest))
	e.Encode(data)
	w.Header().Add("Content-Type", "application/json")
	w.Header().Add("Content-Length", fmt.Sprintf("%d", buf.Len()))
	w.Header().Add("Etag", base64.URLEncoding.EncodeToString(digest.Sum(nil)))
	mh.writeHeaders(status, w, r, data)
	buf.WriteTo(w)
}

func emptyBody() io.ReadCloser {
	return ioutil.NopCloser(&bytes.Buffer{})
}
func copyRequest(req *http.Request) *http.Request {
	nr := &http.Request{}
	*nr = *req
	if req.URL != nil {
		nr.URL = &url.URL{}
		*nr.URL = *req.URL
	}
	nr.Body = emptyBody() //users must copy body themselves
	return nr
}

func (mh *MetaHandler) synthResponse(req *http.Request) *http.Response {
	rw := httptest.NewRecorder()
	mh.router.ServeHTTP(rw, req)
	res := &http.Response{
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		StatusCode: rw.Code,
		Header:     rw.HeaderMap,
	}
	if res.StatusCode == 0 {
		res.StatusCode = 200
	}
	res.Status = http.StatusText(res.StatusCode)
	if rw.Body != nil {
		res.Body = ioutil.NopCloser(bytes.NewReader(rw.Body.Bytes()))
	}

	if trailers, ok := res.Header["Trailer"]; ok {
		res.Trailer = make(http.Header, len(trailers))
		for _, k := range trailers {
			// TODO: use http2.ValidTrailerHeader, but we can't
			// get at it easily because it's bundled into net/http
			// unexported. This is good enough for now:
			switch k {
			case "Transfer-Encoding", "Content-Length", "Trailer":
				// Ignore since forbidden by RFC 2616 14.40.
				continue
			}
			k = http.CanonicalHeaderKey(k)
			vv, ok := rw.HeaderMap[k]
			if !ok {
				continue
			}
			vv2 := make([]string, len(vv))
			copy(vv2, vv)
			res.Trailer[k] = vv2
		}
	}
	return res
}
