package sensonet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	//"github.com/evcc-io/evcc/provider"
	"golang.org/x/oauth2"
)

// Connection is the Sensonet connection
type Connection struct {
	client               *Helper
	identity             *Identity
	homesAndSystemsCache Cacheable[HomesAndSystems]
	cache                time.Duration
	currentQuickmode     string
	quickmodeStarted     time.Time
	quickmodeStopped     time.Time
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
		client:           helper,
		identity:         identity,
		cache:            90 * time.Second,
		currentQuickmode: "", // assuming that no quick mode is active
		quickmodeStarted: time.Now(),
		quickmodeStopped: time.Now().Add(-2 * time.Minute), //time stamp is set in the past so that first call of refreshCurrentQuickMode() changes currentQuickmode if necessary
	}
	token, _ = ts.Token()

	conn.homesAndSystemsCache = ResettableCached(func() (HomesAndSystems, error) {
		var res HomesAndSystems
		err := conn.getHomesAndSystems(&res)
		return res, err
	}, conn.cache)

	return conn, token, nil
}

func (c *Connection) GetCurrentQuickMode() string {
	return c.currentQuickmode
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

// Reads all homes and the system report for all these homes
func (c *Connection) getHomesAndSystems(res *HomesAndSystems) error {
	uri := API_URL_BASE + "/homes"
	req, _ := http.NewRequest("GET", uri, nil)
	req.Header = c.getSensonetHttpHeader()

	//var res Homes
	if err := c.client.DoJSON(req, &res.Homes); err != nil {
		return fmt.Errorf("error getting homes: %w", err)
	}
	var state SystemStatus
	for i, home := range res.Homes {
		url := API_URL_BASE + fmt.Sprintf("/systems/%s/tli", home.SystemID)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header = c.getSensonetHttpHeader()
		if err := c.client.DoJSON(req, &state); err != nil {
			return fmt.Errorf("error getting state for home %s: %w", home.HomeName, err)
		}
		var systemAndId SystemAndId
		systemAndId.SystemId = home.SystemID
		systemAndId.SystemStatus = state
		if len(res.Systems) <= i {
			//If relData.Zones array is not big enough, new elements are appended, especially at first ExtractSystem call
			//At the moment, relData.Zones is not shortened, if later GetSystem calls returns less system.Body.Zones
			res.Systems = append(res.Systems, systemAndId)

		} else {
			res.Systems[i] = systemAndId
		}
		// For the beginning, currentQuickMode is only calculated from the system status of Homes[0].SystemId
		if i == 0 {
			c.refreshCurrentQuickMode(&state)
		}
	}
	return nil
}

func (c *Connection) refreshCurrentQuickMode(state *SystemStatus) {
	newQuickMode := ""
	for _, dhw := range state.State.Dhw {
		if dhw.CurrentSpecialFunction == "CYLINDER_BOOST" {
			newQuickMode = QUICKMODE_HOTWATER
			break
		}
	}
	for _, zone := range state.State.Zones {
		if zone.CurrentSpecialFunction == "QUICK_VETO" {
			newQuickMode = QUICKMODE_HEATING
			break
		}
	}
	if newQuickMode != c.currentQuickmode {
		if newQuickMode != "" && time.Now().Before(c.quickmodeStarted.Add(c.cache)) {
			log.Printf("Old quickmode: %s   New quickmode: %s", c.currentQuickmode, newQuickMode)
			c.currentQuickmode = newQuickMode
			c.quickmodeStopped = time.Now()
		}
		if newQuickMode == "" && time.Now().Before(c.quickmodeStopped.Add(c.cache)) {
			log.Printf("Old quickmode: %s   New quickmode: %s", c.currentQuickmode, newQuickMode)
			c.currentQuickmode = newQuickMode
			c.quickmodeStarted = time.Now()
		}
	}
}

// Returns all "homes" that belong to the current user under the myVaillant portal
func (c *Connection) GetHomes() (Homes, error) {
	homesAndSystems, err := c.homesAndSystemsCache.Get()
	if err != nil {
		return nil, fmt.Errorf("error getting homes: %w", err)
	}
	if len(homesAndSystems.Homes) < 1 {
		return nil, fmt.Errorf("error: no homes")
	}
	return homesAndSystems.Homes, nil
}

// Returns the system report (state, properties and configuration) for a specific systemId
func (c *Connection) GetSystem(systemid string) (SystemStatus, error) {
	var systemAndId SystemAndId
	homesAndSystems, err := c.homesAndSystemsCache.Get()
	if err != nil {
		return systemAndId.SystemStatus, fmt.Errorf("error getting sytem: %w", err)
	}
	for _, state := range homesAndSystems.Systems {
		if state.SystemId == systemid {
			return state.SystemStatus, nil
		}
	}
	return systemAndId.SystemStatus, nil
}

func (c *Connection) StartZoneQuickVeto(systemId string, zone int, setpoint float32, duration float32) error {
	if zone < 0 {
		zone = ZONEINDEX_DEFAULT
	} // if parameter "zone" is negative, then the default value is used
	if setpoint < 0.0 {
		setpoint = ZONEVETOSETPOINT_DEFAULT
	} // if parameter "setpoint" is negative, then the default value is used
	if duration < 0.0 {
		duration = ZONEVETODURATION_DEFAULT
	} // if parameter "duration" is negative, then the default value is used
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
	if zone < 0 {
		zone = ZONEINDEX_DEFAULT
	} // if parameter "zone" is negative, then the default value is used
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

func (c *Connection) StartStrategybased(systemId string, strategy int, heatingPar *HeatingParStruct, hotwaterPar *HotwaterParStruct) (string, error) {
	c.homesAndSystemsCache.Reset()
	state, err := c.GetSystem(systemId)
	if err != nil {
		err = fmt.Errorf("could not read homes and systems cache: %s", err)
		return "", err
	}
	c.refreshCurrentQuickMode(&state)
	// Extracting correct State.Dhw element
	dhwData := GetDhwData(state, hotwaterPar.Index)
	// Extracting correct State.Zone element
	zoneData := GetZoneData(state, heatingPar.ZoneIndex)

	if c.currentQuickmode != "" {
		log.Println("System is already in quick mode: ", c.currentQuickmode)
		log.Println("Is there any need to change that?")
		log.Println("Special Function of Dhw: ", dhwData.State.CurrentSpecialFunction)
		log.Println("Special Function of Heating Zone: ", zoneData.State.CurrentSpecialFunction)
		return QUICKMODE_ERROR_ALREADYON, err
	}

	/*if d.currentQuickmode == "" && d.quickmodeStopped.After(d.quickmodeStarted) && d.quickmodeStopped.Add(2*time.Minute).After(time.Now()) {
		enable = false
	}*/
	whichQuickMode := c.WhichQuickMode(dhwData, zoneData, strategy, heatingPar, hotwaterPar)
	log.Println("WhichQuickMode()=", whichQuickMode)

	switch whichQuickMode {
	case 1:
		err = c.StartHotWaterBoost(systemId, hotwaterPar.Index)
		if err == nil {
			c.currentQuickmode = QUICKMODE_HOTWATER
			c.quickmodeStarted = time.Now()
			log.Println("Starting quick mode (hotwater boost)", c.currentQuickmode)
		}
	case 2:
		err = c.StartZoneQuickVeto(systemId, heatingPar.ZoneIndex, heatingPar.VetoSetpoint, heatingPar.VetoDuration)
		if err == nil {
			c.currentQuickmode = QUICKMODE_HEATING
			c.quickmodeStarted = time.Now()
			log.Println("Starting zone quick veto")
		}
	default:
		if c.currentQuickmode == QUICKMODE_HOTWATER {
			//if hotwater boost active, then stop it
			err = c.StopHotWaterBoost(systemId, hotwaterPar.Index)
			if err == nil {
				log.Println("Stopping Hotwater Boost")
			}
		}
		if c.currentQuickmode == QUICKMODE_HEATING {
			//if zone quick veto active, then stop it
			err = c.StopZoneQuickVeto(systemId, heatingPar.ZoneIndex)
			if err == nil {
				log.Println("Stopping Zone Quick Veto")
			}
		}
		c.currentQuickmode = QUICKMODE_NOTHING
		c.quickmodeStarted = time.Now()
		log.Println("Enable called but no quick mode possible. Starting idle mode")
	}

	return c.currentQuickmode, err
}

func (c *Connection) StopStrategybased(systemId string, strategy int, heatingPar *HeatingParStruct, hotwaterPar *HotwaterParStruct) (string, error) {
	c.homesAndSystemsCache.Reset()
	state, err := c.GetSystem(systemId)
	if err != nil {
		err = fmt.Errorf("could not read system state: %s", err)
		return "", err
	}
	c.refreshCurrentQuickMode(&state)
	// Extracting correct State.Dhw element
	dhwData := GetDhwData(state, hotwaterPar.Index)
	// Extracting correct State.Zone element
	zoneData := GetZoneData(state, heatingPar.ZoneIndex)

	log.Println("Operationg Mode of Dhw: ", dhwData.State.CurrentSpecialFunction)
	log.Println("Operationg Mode of Heating: ", zoneData.State.CurrentSpecialFunction)

	switch c.currentQuickmode {
	case QUICKMODE_HOTWATER:
		err = c.StopHotWaterBoost(systemId, hotwaterPar.Index)
		if err == nil {
			log.Println("Stopping Quick Mode", c.currentQuickmode)
		}
	case QUICKMODE_HEATING:
		err = c.StopZoneQuickVeto(systemId, heatingPar.ZoneIndex)
		if err == nil {
			log.Println("Stopping Zone Quick Veto")
		}
	case QUICKMODE_NOTHING:
		log.Println("Stopping idle quick mode")
	default:
		log.Println("Nothing to do, no quick mode active")
	}
	c.currentQuickmode = ""
	c.quickmodeStopped = time.Now()

	return c.currentQuickmode, err
}

// This function checks the operation mode of heating and hotwater and the hotwater live temperature
// and returns, which quick mode should be started, when evcc sends an "Enable"
func (c *Connection) WhichQuickMode(dhwData *DhwData, zoneData *ZoneData, strategy int, heatingPar *HeatingParStruct, hotwater *HotwaterParStruct) int {
	log.Println("Strategy = ", strategy)
	//log.Printf("Checking if hot water boost possible. Operation Mode = %s, temperature setpoint= %02.2f, live temperature= %02.2f", res.Hotwater.OperationMode, res.Hotwater.HotwaterTemperatureSetpoint, res.Hotwater.HotwaterLiveTemperature)
	// For strategy=STRATEGY_HOTWATER, a hotwater boost is possible when hotwater storage temperature is less than the temperature setpoint.
	// For other strategies, a hotwater boost is possible when hotwater storage temperature is less than the temperature setpoint minus 5°C
	addOn := -5.0
	if strategy == STRATEGY_HOTWATER {
		addOn = 0.0
	}
	hotWaterBoostPossible := false
	if dhwData != nil {
		if dhwData.State.CurrentDhwTemperature < dhwData.Configuration.TappingSetpoint+addOn &&
			dhwData.Configuration.OperationModeDhw == OPERATIONMODE_TIME_CONTROLLED {
			hotWaterBoostPossible = true
		}
	}
	heatingQuickVetoPossible := false
	if zoneData != nil {
		if zoneData.Configuration.Heating.OperationModeHeating == OPERATIONMODE_TIME_CONTROLLED {
			heatingQuickVetoPossible = true
		}
	}

	whichQuickMode := 0
	switch strategy {
	case STRATEGY_HOTWATER:
		if hotWaterBoostPossible {
			whichQuickMode = 1
		} else {
			log.Println("Strategy = hotwater, but hotwater boost not possible")
		}
	case STRATEGY_HEATING:
		if heatingQuickVetoPossible {
			whichQuickMode = 2
		} else {
			log.Println("Strategy = heating, but heating quick veto not possible")
		}
	case STRATEGY_HOTWATER_THEN_HEATING:
		if hotWaterBoostPossible {
			whichQuickMode = 1
		} else {
			if heatingQuickVetoPossible {
				whichQuickMode = 2
			} else {
				log.Println("Strategy = hotwater_then_heating, but both not possible")
			}
		}
	}
	return whichQuickMode
}
