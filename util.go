package sensonet

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
	*http.Client
}

// NewHelper creates http helper for simplified PUT GET logic
func NewHelper(client *http.Client) *Helper {
	return &Helper{
		Client: client,
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

func GetDhwData(state SystemStatus, index int) *DhwData {
	// Extracting correct State.Dhw element
	if len(state.State.Dhw) == 0 {
		return nil
	}
	var dhwData DhwData
	for _, stateDhw := range state.State.Dhw {
		if stateDhw.Index == index || (stateDhw.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			dhwData.State = stateDhw
			break
		}
	}
	for _, propDhw := range state.Properties.Dhw {
		if propDhw.Index == index || (propDhw.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			dhwData.Properties = propDhw
			break
		}
	}
	for _, confDhw := range state.Configuration.Dhw {
		if confDhw.Index == index || (confDhw.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			dhwData.Configuration = confDhw
			break
		}
	}
	return &dhwData
}

func GetDomesticHotWaterData(state SystemStatus, index int) *DomesticHotWaterData {
	// Extracting correct State.DomesticHotWater element
	if len(state.State.DomesticHotWater) == 0 {
		return nil
	}
	var domesticHotWaterData DomesticHotWaterData
	for _, stateDomesticHotWater := range state.State.DomesticHotWater {
		if stateDomesticHotWater.Index == index || (stateDomesticHotWater.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			domesticHotWaterData.State = stateDomesticHotWater
			break
		}
	}
	for _, propDomesticHotWater := range state.Properties.DomesticHotWater {
		if propDomesticHotWater.Index == index || (propDomesticHotWater.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			domesticHotWaterData.Properties = propDomesticHotWater
			break
		}
	}
	for _, confDomesticHotWater := range state.Configuration.DomesticHotWater {
		if confDomesticHotWater.Index == index || (confDomesticHotWater.Index == HOTWATERINDEX_DEFAULT && index < 0) {
			domesticHotWaterData.Configuration = confDomesticHotWater
			break
		}
	}
	return &domesticHotWaterData
}

func GetZoneData(state SystemStatus, index int) *ZoneData {
	// Extracting correct State.Zones element
	if len(state.State.Zones) == 0 {
		return nil
	}
	var zoneData ZoneData
	for _, stateZone := range state.State.Zones {
		if stateZone.Index == index || (stateZone.Index == ZONEINDEX_DEFAULT && index < 0) {
			zoneData.State = stateZone
			break
		}
	}
	for _, propZone := range state.Properties.Zones {
		if propZone.Index == index || (propZone.Index == ZONEINDEX_DEFAULT && index < 0) {
			zoneData.Properties = propZone
			break
		}
	}
	for _, confZone := range state.Configuration.Zones {
		if confZone.Index == index || (confZone.Index == ZONEINDEX_DEFAULT && index < 0) {
			zoneData.Configuration = confZone
			break
		}
	}
	return &zoneData
}
