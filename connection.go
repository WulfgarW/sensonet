package sensonet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2"
)

// Connection is the Sensonet connection
type Connection struct {
	client               *http.Client
	homesAndSystemsCache Cacheable[HomesAndSystems]
	cache                time.Duration
	currentQuickmode     string
	quickmodeStarted     time.Time
	quickmodeStopped     time.Time
}

// NewConnection creates a new Sensonet device connection.
func NewConnection(client *http.Client, ts oauth2.TokenSource) (*Connection, error) {
	client.Transport = &oauth2.Transport{
		Source: ts,
		Base:   client.Transport,
	}

	conn := &Connection{
		client:           client,
		cache:            90 * time.Second,
		currentQuickmode: "", // assuming that no quick mode is active
		quickmodeStarted: time.Now(),
		quickmodeStopped: time.Now().Add(-2 * time.Minute), // time stamp is set in the past so that first call of refreshCurrentQuickMode() changes currentQuickmode if necessary
	}

	conn.homesAndSystemsCache = ResettableCached(func() (HomesAndSystems, error) {
		var res HomesAndSystems
		err := conn.getHomesAndSystems(&res)
		return res, err
	}, conn.cache)

	return conn, nil
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

	// var res Homes
	if err := doJSON(c.client, req, &res.Homes); err != nil {
		return fmt.Errorf("error getting homes: %w", err)
	}
	var state SystemStatus
	for i, home := range res.Homes {
		url := API_URL_BASE + fmt.Sprintf("/systems/%s/tli", home.SystemID)
		req, _ := http.NewRequest("GET", url, nil)
		req.Header = c.getSensonetHttpHeader()
		if err := doJSON(c.client, req, &state); err != nil {
			return fmt.Errorf("error getting state for home %s: %w", home.HomeName, err)
		}
		var systemDevices SystemDevices
		url = API_URL_BASE + fmt.Sprintf(DEVICES_URL, home.SystemID)
		req, _ = http.NewRequest("GET", url, nil)
		req.Header = c.getSensonetHttpHeader()
		if err := doJSON(c.client, req, &systemDevices); err != nil {
			return fmt.Errorf("error getting device data: %w", err)
		}
		var systemAndId SystemAndId
		systemAndId.SystemId = home.SystemID
		systemAndId.SystemStatus = state
		systemAndId.SystemDevices = systemDevices
		if len(res.Systems) <= i {
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
		if newQuickMode == "" && time.Now().After(c.quickmodeStarted.Add(c.cache)) {
			if c.currentQuickmode == QUICKMODE_NOTHING && time.Now().Before(c.quickmodeStarted.Add(10*time.Minute)) {
				log.Println("Idle mode active for less then 10 minutes. Keeping the idle mode")
			} else {
				log.Printf("Old quickmode: \"%s\"   New quickmode: \"%s\"", c.currentQuickmode, newQuickMode)
				c.currentQuickmode = newQuickMode
				c.quickmodeStopped = time.Now()
			}
		}
		if newQuickMode != "" && time.Now().After(c.quickmodeStopped.Add(c.cache)) {
			log.Printf("Old quickmode: \"%s\"   New quickmode: \"%s\"", c.currentQuickmode, newQuickMode)
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
	for _, sys := range homesAndSystems.Systems {
		if sys.SystemId == systemid {
			return sys.SystemStatus, nil
		}
	}
	return systemAndId.SystemStatus, nil
}

// Returns the system devices for a specific systemId
func (c *Connection) getSystemDevices(systemid string) (SystemDevices, error) {
	var systemAndId SystemAndId
	homesAndSystems, err := c.homesAndSystemsCache.Get()
	if err != nil {
		return systemAndId.SystemDevices, fmt.Errorf("error getting sytem: %w", err)
	}
	for _, sys := range homesAndSystems.Systems {
		if sys.SystemId == systemid {
			return sys.SystemDevices, nil
		}
	}
	return systemAndId.SystemDevices, nil
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
	resp, err = doBody(c.client, req)
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
	resp, err = doBody(c.client, req)
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
	resp, err = doBody(c.client, req)
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
	resp, err = doBody(c.client, req)
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
			// if hotwater boost active, then stop it
			err = c.StopHotWaterBoost(systemId, hotwaterPar.Index)
			if err == nil {
				log.Println("Stopping Hotwater Boost")
			}
		}
		if c.currentQuickmode == QUICKMODE_HEATING {
			// if zone quick veto active, then stop it
			err = c.StopZoneQuickVeto(systemId, heatingPar.ZoneIndex)
			if err == nil {
				log.Println("Stopping Zone Quick Veto")
			}
		}
		c.currentQuickmode = QUICKMODE_NOTHING
		c.quickmodeStarted = time.Now()
		log.Println("Enable called but no quick mode possible. Starting idle mode")
	}

	c.homesAndSystemsCache.Reset()
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

	c.homesAndSystemsCache.Reset()
	return c.currentQuickmode, err
}

// This function checks the operation mode of heating and hotwater and the hotwater live temperature
// and returns, which quick mode should be started, when evcc sends an "Enable"
func (c *Connection) WhichQuickMode(dhwData *DhwData, zoneData *ZoneData, strategy int, heatingPar *HeatingParStruct, hotwater *HotwaterParStruct) int {
	log.Println("Strategy = ", strategy)
	// log.Printf("Checking if hot water boost possible. Operation Mode = %s, temperature setpoint= %02.2f, live temperature= %02.2f", res.Hotwater.OperationMode, res.Hotwater.HotwaterTemperatureSetpoint, res.Hotwater.HotwaterLiveTemperature)
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

// Returns the device data for give criteria
func (c *Connection) GetDeviceData(systemid string, whichDevices int) ([]DeviceAndInfo, error) {
	var devices []DeviceAndInfo
	systemDevices, err := c.getSystemDevices(systemid)
	if err != nil {
		return devices, fmt.Errorf("error getting sytem devices for %s: %w", systemid, err)
	}
	var deviceAndInfo DeviceAndInfo
	if systemDevices.PrimaryHeatGenerator.DeviceUUID != "" && (whichDevices == DEVICES_PRIMARY_HEATER || whichDevices == DEVICES_ALL) {
		deviceAndInfo.Device = systemDevices.PrimaryHeatGenerator
		deviceAndInfo.Info = "primary_heat_generator"
		devices = append(devices, deviceAndInfo)
	}
	if whichDevices == DEVICES_SECONDARY_HEATER || whichDevices == DEVICES_ALL {
		for _, secHeatGen := range systemDevices.SecondaryHeatGenerators {
			deviceAndInfo.Device = secHeatGen
			deviceAndInfo.Info = "secondary_heat_generator"
			devices = append(devices, deviceAndInfo)
		}
	}

	if systemDevices.ElectricBackupHeater.DeviceUUID != "" && (whichDevices == DEVICES_BACKUP_HEATER || whichDevices == DEVICES_ALL) {
		deviceAndInfo.Device = systemDevices.ElectricBackupHeater
		deviceAndInfo.Info = "electric_backup_heater"
		devices = append(devices, deviceAndInfo)
	}
	return devices, nil
}

// Returns the energy data systemId
func (c *Connection) GetEnergyData(systemid, deviceUuid, operationMode, energyType, resolution string, startDate, endDate time.Time) (EnergyData, error) {
	var energyData EnergyData
	v := url.Values{}
	v.Set("resolution", resolution)
	v.Add("operationMode", operationMode)
	v.Add("energyType", energyType)
	v.Add("startDate", startDate.Format("2006-01-02T15:04:05-07:00"))
	v.Add("endDate", endDate.Format("2006-01-02T15:04:05-07:00"))
	url := API_URL_BASE + fmt.Sprintf(ENERGY_URL, systemid, deviceUuid) + v.Encode()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header = c.getSensonetHttpHeader()
	if err := doJSON(c.client, req, &energyData); err != nil {
		return energyData, fmt.Errorf("error getting energy data for %s: %w", deviceUuid, err)
	}
	return energyData, nil
}
