package sensonet

// Copied from https://github.com/evcc-io/evcc

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

var (
	JSONContent = "application/json"
	// JSONEncoding specifies application/json
	JSONEncoding = map[string]string{
		"Content-Type": JSONContent,
		"Accept":       JSONContent,
	}
	// AcceptJSON accepting application/json
	AcceptJSON = map[string]string{
		"Accept": JSONContent,
	}
)

// Helper provides utility primitives
type Helper struct {
	httpDoer
}

type httpDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// NewHelper creates http helper for simplified PUT GET logic
func NewHelper(client httpDoer) *Helper {
	return &Helper{
		httpDoer: client,
	}
}

// DoBody executes HTTP request and returns the response body
func (r *Helper) DoBody(req *http.Request) ([]byte, error) {
	resp, err := r.Do(req)
	var body []byte
	if err == nil {
		body, err = ReadBody(resp)
	}
	return body, err
}

// decodeJSON reads HTTP response and decodes JSON body if error is nil
func decodeJSON(resp *http.Response, res interface{}) error {
	if err := ResponseError(resp); err != nil {
		_ = json.NewDecoder(resp.Body).Decode(&res)
		return err
	}
	return json.NewDecoder(resp.Body).Decode(&res)
}

// DoJSON executes HTTP request and decodes JSON response.
// It returns a StatusError on response codes other than HTTP 2xx.
func (r *Helper) DoJSON(req *http.Request, res interface{}) error {
	resp, err := r.Do(req)
	if err == nil {
		defer resp.Body.Close()
		err = decodeJSON(resp, &res)
	}
	return err
}

// StatusError indicates unsuccessful http response
type StatusError struct {
	resp *http.Response
}

func (e StatusError) Error() string {
	return fmt.Sprintf("unexpected status: %d (%s)", e.resp.StatusCode, http.StatusText(e.resp.StatusCode))
}

// Response returns the response with the unexpected error
func (e StatusError) Response() *http.Response {
	return e.resp
}

// StatusCode returns the response's status code
func (e StatusError) StatusCode() int {
	return e.resp.StatusCode
}

// ResponseError turns an HTTP status code into an error
func ResponseError(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return StatusError{resp: resp}
	}
	return nil
}

// ReadBody reads HTTP response and returns error on response codes other than HTTP 2xx. It closes the request body after reading.
func ReadBody(resp *http.Response) ([]byte, error) {
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return []byte{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return b, StatusError{resp: resp}
	}
	return b, nil
}
