package swaggering

import (
	"bytes"
	"io"
	"io/ioutil"
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
	}

	// DummyControl is a tool for controlling a particular kind of DummyClient
	DummyControl struct {
		dtos    chan dummyDTOResponse
		simples chan dummySimpleResponse
	}
)

// NewChannelDummy returns a pair of a DummyClient and a DummyControl.
// Responses fed to DummyControl will be returned by the DummyClient
func NewChannelDummy() (DummyClient, DummyControl) {
	ctrl := DummyControl{
		dtos:    make(chan dummyDTOResponse, 15),
		simples: make(chan dummySimpleResponse, 15),
	}

	clnt := DummyClient{
		NextDTO: func(m, p string, pp, qp urlParams, b ...DTO) (DTO, error) {
			dr := <-ctrl.dtos
			return dr.dto, dr.err
		},
		NextSimple: func(m, p string, pp, qp urlParams, b ...DTO) (string, error) {
			sr := <-ctrl.simples
			return sr.body, sr.err
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
	dto, err := dc.NextDTO(m, p, pp, qp, b...)
	if err != nil {
		return err
	}
	err = pop.Absorb(dto)
	if err != nil {
		return err
	}
	return nil
}

// Request performs an HTTP request and returns the body of the response
func (dc *DummyClient) Request(m, p string, pp, qp urlParams, b ...DTO) (io.ReadCloser, error) {
	body, err := dc.NextSimple(m, p, pp, qp, b...)
	if err != nil {
		return nil, err
	}

	return ioutil.NopCloser(bytes.NewBufferString(body)), nil
}
