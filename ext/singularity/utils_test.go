package singularity

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"io"
	"log"
	"net/url"
	"strings"

	"github.com/opentable/swaggering"
)

// Requester copied from swaggering for reference.

// Requester defines the interface that Swaggering uses to
// make actual HTTP requests of the API server
type Requester interface {
	// DTORequest performs an HTTP request and populates a DTO based on the response
	DTORequest(dto swaggering.DTO, method, path string, pathParams, queryParams swaggering.URLParams, body ...swaggering.DTO) error

	// Request performs an HTTP request and returns the body of the response
	Request(method, path string, pathParams, queryParams swaggering.URLParams, body ...swaggering.DTO) (io.ReadCloser, error)
}

type TestGETRequester map[string]swaggering.DTO

func (tgr TestGETRequester) Request(method, path string, pathParams, queryParams swaggering.URLParams, body ...swaggering.DTO) (io.ReadCloser, error) {
	panic("Not implemented.")
}

func (tgr TestGETRequester) DTORequest(dto swaggering.DTO, method, path string, pathParams, queryParams swaggering.URLParams, body ...swaggering.DTO) error {

	// Turn path into a text/template string.
	path = strings.Replace(path, "{", "{{.", -1)
	path = strings.Replace(path, "}", "}}", -1)
	// Populate it with pathParams.
	var t = template.Must(template.New("url").Parse(path))
	pathWriter := &bytes.Buffer{}
	t.Execute(pathWriter, pathParams)
	path = pathWriter.String()

	// Do matching against URL unescaped strings.
	cleanPath, err := url.QueryUnescape(path)
	if err != nil {
		log.Fatal(err)
	}
	cleanPath = html.UnescapeString(cleanPath)
	for p, d := range tgr {
		cleanP, err := url.QueryUnescape(p)
		if err != nil {
			log.Fatal(err)
		}
		cleanP = html.UnescapeString(cleanP)
		if cleanP == cleanPath {
			return dto.Absorb(d)
		}
		log.Printf("No match: %q != %q", cleanP, cleanPath)
	}
	return fmt.Errorf("no DTO available at %s", path)
}

func (tgr *TestGETRequester) RegisterDTO(dto swaggering.DTO, pathFormat string, a ...interface{}) {
	pathSegments := make([]interface{}, len(a))
	for i, a := range a {
		pathSegments[i] = url.QueryEscape(fmt.Sprint(a))
	}
	path := fmt.Sprintf(pathFormat, pathSegments...)
	(*tgr)[path] = dto
	log.Printf("Registered a %T at %s", dto, path)
}
