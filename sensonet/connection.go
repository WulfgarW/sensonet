package sensonet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"golang.org/x/oauth2"
)

// Connection is the Sensonet connection
type Connection struct {
	client   *Helper
	identity *Identity
}

// NewConnection creates a new Sensonet device connection.
func NewConnection(client *http.Client, credentials *CredentialsStruct, token *oauth2.Token) (*Connection, *oauth2.Token, error) {
	helper := NewHelper(client)
	identity, err := NewIdentity(helper, credentials)
	if err != nil {
		log.Fatal(err)
	}
	var ts oauth2.TokenSource
	if token == nil {
		log.Println("No token provided. Calling login")
		token, err = identity.Login()
		if err != nil {
			log.Fatal(err)
		}
	}

	ts, err = identity.TokenSource(token)
	if err != nil {
		log.Println("Error generating token source from provied token. Calling login")
		token, err = identity.Login()
		if err != nil {
			log.Fatal(err)
		}
		ts, err = identity.TokenSource(token)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Generating new token source successful")
	}

	client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   client.Transport,
	}

	conn := &Connection{
		client:   helper,
		identity: identity,
	}
	token, _ = ts.Token()
	return conn, token, nil
}

// Returns the http header for http requests to sensonet
func (c *Connection) getSensonetHttpHeader() http.Header {
	return http.Header{
		"Accept-Language":           {"en-GB"},
		"Accept":                    {"application/json, text/plain, */*"},
		"x-app-identifier":          {"VAILLANT"},
		"x-client-locale":           {"en-GB"},
		"x-idm-identifier":          {"KEYCLOAK"},
		"ocp-apim-subscription-key": {"1e0a2f3511fb4c5bbb1c7f9fedd20b1c"},
	}
}

// Returns all "homes" that belong to the current user under the myVaillant portal
func (c *Connection) GetHomes() (Homes, error) {
	uri := API_URL_BASE + "/homes"
	req, _ := http.NewRequest("GET", uri, nil)
	req.Header = c.getSensonetHttpHeader()

	var res Homes
	if err := c.client.DoJSON(req, &res); err != nil {
		return nil, fmt.Errorf("error getting homes: %w", err)
	}
	return res, nil
}

// Returns the system report (state, properties and configuration) for a specific systemId
func (c *Connection) GetSystem(systemId string) (SystemStatus, error) {
	var res SystemStatus

	url := API_URL_BASE + fmt.Sprintf("/systems/%s/tli", systemId)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header = c.getSensonetHttpHeader()
	err := c.client.DoJSON(req, &res)
	return res, err
}

func (c *Connection) StartZoneQuickVeto(systemId string, zone int, setpoint float32, duration float32) error {
	urlZoneQuickVeto := API_URL_BASE + fmt.Sprintf(ZONEQUICKVETO_URL, systemId, zone)
	data := map[string]float32{
		"desiredRoomTemperatureSetpoint": setpoint,
		"duration":                       duration,
	}
	mData, _ := json.Marshal(data)
	req, err := http.NewRequest("POST", urlZoneQuickVeto, bytes.NewReader(mData))
	if err != nil {
		err = fmt.Errorf("client: could not create request: %s", err)
		return err
	}
	req.Header = c.getSensonetHttpHeader()
	req.Header.Set("Content-Type", "application/json")
	var resp []byte
	log.Printf("Sending POST request to: %s\n", urlZoneQuickVeto)
	resp, err = c.client.DoBody(req)
	if err != nil {
		err = fmt.Errorf("could not start quick veto. Error: %s", err)
		log.Printf("Response: %s\n", resp)
		return err
	}
	return err
}

func (c *Connection) StopZoneQuickVeto(systemId string, zone int) error {
	urlZoneQuickVeto := API_URL_BASE + fmt.Sprintf(ZONEQUICKVETO_URL, systemId, zone)
	req, err := http.NewRequest("DELETE", urlZoneQuickVeto, bytes.NewBuffer(nil))
	if err != nil {
		err = fmt.Errorf("client: could not create request: %s", err)
		return err
	}
	req.Header = c.getSensonetHttpHeader()
	var resp []byte
	log.Printf("Sending DELETE request to: %s\n", urlZoneQuickVeto)
	resp, err = c.client.DoBody(req)
	if err != nil {
		err = fmt.Errorf("could not stop quick veto. Error: %s", err)
		log.Printf("Response: %s\n", resp)
		return err
	}
	return err
}

func (c *Connection) StartHotWaterBoost(systemId string, hotwaterIndex int) error {
	if hotwaterIndex < 0 {
		hotwaterIndex = HOTWATERINDEX_DEFAULT
	} // if parameter "hotwaterIndex" is negative, then the default value is used
	urlHotwaterBoost := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, systemId, hotwaterIndex)
	mData, _ := json.Marshal(map[string]string{})
	req, err := http.NewRequest("POST", urlHotwaterBoost, bytes.NewReader(mData))
	if err != nil {
		err = fmt.Errorf("client: could not create request: %s", err)
		return err
	}
	req.Header = c.getSensonetHttpHeader()
	req.Header.Set("Content-Type", "application/json")
	var resp []byte
	resp, err = c.client.DoBody(req)
	if err != nil {
		err = fmt.Errorf("could not start hotwater boost. Error: %s", err)
		log.Printf("Response: %s\n", resp)
		return err
	}
	return err
}

func (c *Connection) StopHotWaterBoost(systemId string, hotwaterIndex int) error {
	if hotwaterIndex < 0 {
		hotwaterIndex = HOTWATERINDEX_DEFAULT
	} // if parameter "hotwaterIndex" is negative, then the default value is used
	urlHotwaterBoost := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, systemId, hotwaterIndex)
	req, err := http.NewRequest("DELETE", urlHotwaterBoost, bytes.NewBuffer(nil))
	if err != nil {
		err = fmt.Errorf("client: could not create request: %s", err)
		return err
	}
	req.Header = c.getSensonetHttpHeader()
	var resp []byte
	resp, err = c.client.DoBody(req)
	if err != nil {
		err = fmt.Errorf("could not stop hotwater boost. Error: %s", err)
		log.Printf("Response: %s\n", resp)
		return err
	}
	return err
}
