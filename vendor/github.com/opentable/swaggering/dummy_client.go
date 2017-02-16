package swaggering

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
)

type (
	dummyDTOResponse struct {
		dto DTO
		err error
	}

	dummySimpleResponse struct {
		body string
		err  error
	}

	// DummyClient stands in for the generic part of a swaggering client for testing
	DummyClient struct {
		NextDTO    func(method, path string, pathParams, queryParams urlParams, body ...DTO) (DTO, error)
		NextSimple func(method, path string, pathParams, queryParams urlParams, body ...DTO) (string, error)
		ReplaceDTO func(dummyDTOResponse)
	}

	// DummyControl is a tool for controlling a particular kind of DummyClient
	DummyControl struct {
		dtos    chan dummyDTOResponse
		simples chan dummySimpleResponse
	}

	// StarvedChannelError means that a channel needed a value but didn't have one
	StarvedChannelError struct {
		m, p, kind, bodyT string
		pp, qp            urlParams
	}
)

func makeStarvedChannelError(kind, m, p string, pp, qp urlParams, b ...DTO) *StarvedChannelError {
	bodyT := "<empty>"
	if len(b) > 0 {
		bodyT = fmt.Sprintf("%T", b[0])
	}
	return &StarvedChannelError{
		m: m, p: p, pp: pp, qp: qp,
		kind:  kind,
		bodyT: bodyT,
	}
}

func (e *StarvedChannelError) Error() string {
	return fmt.Sprintf("swaggering.DummyClient: no %s response for %s %s params: %v %v body: %s", e.kind, e.m, e.p, e.pp, e.qp, e.bodyT)
}

// NewChannelDummy returns a pair of a DummyClient and a DummyControl.
// Responses fed to DummyControl will be returned by the DummyClient
func NewChannelDummy() (DummyClient, DummyControl) {
	ctrl := DummyControl{
		dtos:    make(chan dummyDTOResponse, 15),
		simples: make(chan dummySimpleResponse, 15),
	}

	clnt := DummyClient{
		NextDTO: func(m, p string, pp, qp urlParams, b ...DTO) (DTO, error) {
			select {
			case dr := <-ctrl.dtos:
				return dr.dto, dr.err
			default:
				return nil, makeStarvedChannelError("dto", m, p, pp, qp, b...)
			}
		},
		NextSimple: func(m, p string, pp, qp urlParams, b ...DTO) (string, error) {
			select {
			case sr := <-ctrl.simples:
				return sr.body, sr.err
			default:
				return "", makeStarvedChannelError("dto", m, p, pp, qp, b...)
			}
		},
		ReplaceDTO: func(d dummyDTOResponse) {
			ctrl.dtos <- d
		},
	}

	return clnt, ctrl
}

// FeedDTO pushes a DTO (and an error) into the queue for the paired
// DummyClient to return
func (c DummyControl) FeedDTO(dto DTO, err error) {
	c.dtos <- dummyDTOResponse{dto: dto, err: err}
}

// FeedSimple pushes a body string (and an error) into the queue for the paired
// DummyClient to return
func (c DummyControl) FeedSimple(body string, err error) {
	c.simples <- dummySimpleResponse{body: body, err: err}
}

// DTORequest performs an HTTP request and populates a DTO based on the response
func (dc *DummyClient) DTORequest(pop DTO, m, p string, pp, qp urlParams, b ...DTO) error {

	unusedDTOs := make(chan dummyDTOResponse, 100)
	defer func() {
		// Put back the unused DTOs.
		log.Printf("UNUSED DTOS: %d", len(unusedDTOs))
		close(unusedDTOs) // the "close" goes in the buffer like any value.
		for d := range unusedDTOs {
			panic("HEY")
			dc.ReplaceDTO(d)
		}
	}()
	var errors []error
	for dto, err := dc.NextDTO(m, p, pp, qp, b...); ; {
		if err != nil {
			if _, ok := err.(*StarvedChannelError); ok {
				if len(errors) == 0 {
					return err
				}
				return fmt.Errorf("no acceptable DTOs, looked at %d, got errors: % #v",
					len(errors), errors)
			}
			if dto != nil {
				unusedDTOs <- dummyDTOResponse{dto, err}
			}
			return err
		}
		if err := pop.Absorb(dto); err != nil {
			unusedDTOs <- dummyDTOResponse{dto, nil}
			errors = append(errors, err)
			continue
		} else {
			return nil
		}
	}
	//return fmt.Errorf("errors: % #v", errors)
}

// Request performs an HTTP request and returns the body of the response
func (dc *DummyClient) Request(m, p string, pp, qp urlParams, b ...DTO) (io.ReadCloser, error) {
	body, err := dc.NextSimple(m, p, pp, qp, b...)
	if err != nil {
		return nil, err
	}

	return ioutil.NopCloser(bytes.NewBufferString(body)), nil
}
