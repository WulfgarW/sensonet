package sensonet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/oauth2"
)

type Logger interface {
	Printf(msg string, arg ...any)
}

type Option func(*Connection)

func WithLogger(logger Logger) Option {
	return func(c *Connection) {
		c.logger = logger
	}
}

// Connection is the Sensonet connection
type Connection struct {
	client               *http.Client
	logger               Logger
	homesAndSystemsCache Cacheable[HomesAndSystems]
	cache                time.Duration
	currentQuickmode     string
	quickmodeStarted     time.Time
	quickmodeStopped     time.Time
}

type sensonetHeaders struct {
	http.RoundTripper
}

func (t *sensonetHeaders) RoundTrip(req *http.Request) (*http.Response, error) {
	for k, v := range (http.Header{
		"Accept-Language":           {"en-GB"},
		"Accept":                    {"application/json, text/plain, */*"},
		"x-app-identifier":          {"VAILLANT"},
		"x-client-locale":           {"en-GB"},
		"x-idm-identifier":          {"KEYCLOAK"},
		"ocp-apim-subscription-key": {"1e0a2f3511fb4c5bbb1c7f9fedd20b1c"},
	}) {
		for _, vv := range v {
			req.Header.Add(k, vv)
		}
	}
	return http.DefaultTransport.RoundTrip(req)
}

// NewConnection creates a new Sensonet device connection.
func NewConnection(client *http.Client, ts oauth2.TokenSource, opts ...Option) (*Connection, error) {
	client.Transport = &oauth2.Transport{
		Source: ts,
		Base: &sensonetHeaders{
			client.Transport,
		},
	}

	conn := &Connection{
		client:           client,
		cache:            90 * time.Second,
		currentQuickmode: "", // assuming that no quick mode is active
		quickmodeStarted: time.Now(),
		quickmodeStopped: time.Now().Add(-2 * time.Minute), // time stamp is set in the past so that first call of refreshCurrentQuickMode() changes currentQuickmode if necessary
	}
	for _, opt := range opts {
		opt(conn)
	}

	conn.homesAndSystemsCache = ResettableCached(func() (HomesAndSystems, error) {
		var res HomesAndSystems
		err := conn.getHomesAndSystems(&res)
		return res, err
	}, conn.cache)

	return conn, nil
}

func (conn *Connection) debug(fmt string, arg ...any) {
	if conn.logger != nil {
		conn.logger.Printf(fmt, arg)
	}
}
func (c *Connection) GetCurrentQuickMode() string {
	return c.currentQuickmode
}

// Reads all homes and the system report for all these homes
func (c *Connection) getHomesAndSystems(res *HomesAndSystems) error {
	url := API_URL_BASE + "/homes"
	req, _ := http.NewRequest("GET", url, nil)

	// var res Homes
	if err := doJSON(c.client, req, &res.Homes); err != nil {
		return fmt.Errorf("error getting homes: %w", err)
	}
	var state SystemStatus
	for i, home := range res.Homes {
		url := API_URL_BASE + fmt.Sprintf("/systems/%s/tli", home.SystemID)
		req, _ := http.NewRequest("GET", url, nil)
		if err := doJSON(c.client, req, &state); err != nil {
			return fmt.Errorf("error getting state for home %s: %w", home.HomeName, err)
		}
		var systemDevices SystemDevices
		url = API_URL_BASE + fmt.Sprintf(DEVICES_URL, home.SystemID)
		req, _ = http.NewRequest("GET", url, nil)
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
				c.debug("Idle mode active for less then 10 minutes. Keeping the idle mode")
			} else {
				c.debug(fmt.Sprintf("Old quickmode: \"%s\"   New quickmode: \"%s\"", c.currentQuickmode, newQuickMode))
				c.currentQuickmode = newQuickMode
				c.quickmodeStopped = time.Now()
			}
		}
		if newQuickMode != "" && time.Now().After(c.quickmodeStopped.Add(c.cache)) {
			c.debug(fmt.Sprintf("Old quickmode: \"%s\"   New quickmode: \"%s\"", c.currentQuickmode, newQuickMode))
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
	url := API_URL_BASE + fmt.Sprintf(ZONEQUICKVETO_URL, systemId, zone)
	data := map[string]float32{
		"desiredRoomTemperatureSetpoint": setpoint,
		"duration":                       duration,
	}
	b, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", url, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")

	if _, err := doBody(c.client, req); err != nil {
		return fmt.Errorf("could not start quick veto: %w", err)
	}

	return nil
}

func (c *Connection) StopZoneQuickVeto(systemId string, zone int) error {
	if zone < 0 {
		zone = ZONEINDEX_DEFAULT
	} // if parameter "zone" is negative, then the default value is used

	url := API_URL_BASE + fmt.Sprintf(ZONEQUICKVETO_URL, systemId, zone)
	req, _ := http.NewRequest("DELETE", url, nil)

	if _, err := doBody(c.client, req); err != nil {
		return fmt.Errorf("could not stop quick veto: %w", err)
	}

	return nil
}

func (c *Connection) StartHotWaterBoost(systemId string, hotwaterIndex int) error {
	if hotwaterIndex < 0 {
		hotwaterIndex = HOTWATERINDEX_DEFAULT
	} // if parameter "hotwaterIndex" is negative, then the default value is used

	url := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, systemId, hotwaterIndex)
	req, _ := http.NewRequest("POST", url, strings.NewReader("{}"))
	req.Header.Set("Content-Type", "application/json")

	if _, err := doBody(c.client, req); err != nil {
		return fmt.Errorf("could not start hotwater boost: %w", err)
	}

	return nil
}

func (c *Connection) StopHotWaterBoost(systemId string, hotwaterIndex int) error {
	if hotwaterIndex < 0 {
		hotwaterIndex = HOTWATERINDEX_DEFAULT
	} // if parameter "hotwaterIndex" is negative, then the default value is used

	url := API_URL_BASE + fmt.Sprintf(HOTWATERBOOST_URL, systemId, hotwaterIndex)
	req, _ := http.NewRequest("DELETE", url, nil)

	if _, err := doBody(c.client, req); err != nil {
		return fmt.Errorf("could not stop hotwater boost: %w", err)
	}

	return nil
}

func (c *Connection) StartStrategybased(systemId string, strategy int, heatingPar *HeatingParStruct, hotwaterPar *HotwaterParStruct) (string, error) {
	c.homesAndSystemsCache.Reset()
	state, err := c.GetSystem(systemId)
	if err != nil {
		return "", fmt.Errorf("could not read homes and systems cache: %w", err)
	}
	c.refreshCurrentQuickMode(&state)
	// Extracting correct State.Dhw element
	dhwData := GetDhwData(state, hotwaterPar.Index)
	// Extracting correct State.Zone element
	zoneData := GetZoneData(state, heatingPar.ZoneIndex)

	if c.currentQuickmode != "" {
		c.debug(fmt.Sprint("System is already in quick mode:", c.currentQuickmode))
		c.debug("Is there any need to change that?")
		c.debug(fmt.Sprint("Special Function of Dhw: ", dhwData.State.CurrentSpecialFunction))
		c.debug(fmt.Sprint("Special Function of Heating Zone: ", zoneData.State.CurrentSpecialFunction))
		return QUICKMODE_ERROR_ALREADYON, err
	}

	whichQuickMode := c.WhichQuickMode(dhwData, zoneData, strategy, heatingPar, hotwaterPar)
	c.debug(fmt.Sprint("whichQuickMode=", whichQuickMode))

	switch whichQuickMode {
	case 1:
		err = c.StartHotWaterBoost(systemId, hotwaterPar.Index)
		if err == nil {
			c.currentQuickmode = QUICKMODE_HOTWATER
			c.quickmodeStarted = time.Now()
			c.debug("Starting hotwater boost")
		}
	case 2:
		err = c.StartZoneQuickVeto(systemId, heatingPar.ZoneIndex, heatingPar.VetoSetpoint, heatingPar.VetoDuration)
		if err == nil {
			c.currentQuickmode = QUICKMODE_HEATING
			c.quickmodeStarted = time.Now()
			c.debug("Starting zone quick veto")
		}
	default:
		if c.currentQuickmode == QUICKMODE_HOTWATER {
			// if hotwater boost active, then stop it
			err = c.StopHotWaterBoost(systemId, hotwaterPar.Index)
			if err == nil {
				c.debug("Stopping hotwater boost")
			}
		}
		if c.currentQuickmode == QUICKMODE_HEATING {
			// if zone quick veto active, then stop it
			err = c.StopZoneQuickVeto(systemId, heatingPar.ZoneIndex)
			if err == nil {
				c.debug("Stopping zone quick veto")
			}
		}
		c.currentQuickmode = QUICKMODE_NOTHING
		c.quickmodeStarted = time.Now()
		c.debug("Enable called but no quick mode possible. Starting idle mode")
	}

	c.homesAndSystemsCache.Reset()
	return c.currentQuickmode, err
}

func (c *Connection) StopStrategybased(systemId string, heatingPar *HeatingParStruct, hotwaterPar *HotwaterParStruct) (string, error) {
	c.homesAndSystemsCache.Reset()
	state, err := c.GetSystem(systemId)
	if err != nil {
		return "", fmt.Errorf("could not read system state: %w", err)
	}
	c.refreshCurrentQuickMode(&state)
	// Extracting correct State.Dhw element
	dhwData := GetDhwData(state, hotwaterPar.Index)
	// Extracting correct State.Zone element
	zoneData := GetZoneData(state, heatingPar.ZoneIndex)

	c.debug(fmt.Sprint("Operationg Mode of Dhw: ", dhwData.State.CurrentSpecialFunction))
	c.debug(fmt.Sprint("Operationg Mode of Heating: ", zoneData.State.CurrentSpecialFunction))

	switch c.currentQuickmode {
	case QUICKMODE_HOTWATER:
		err = c.StopHotWaterBoost(systemId, hotwaterPar.Index)
		if err == nil {
			c.debug(fmt.Sprint("Stopping quick mode", c.currentQuickmode))
		}
	case QUICKMODE_HEATING:
		err = c.StopZoneQuickVeto(systemId, heatingPar.ZoneIndex)
		if err == nil {
			c.debug("Stopping zone quick veto")
		}
	case QUICKMODE_NOTHING:
		c.debug("Stopping idle quick mode")
	default:
		c.debug("Nothing to do, no quick mode active")
	}
	c.currentQuickmode = ""
	c.quickmodeStopped = time.Now()

	c.homesAndSystemsCache.Reset()
	return c.currentQuickmode, err
}

// This function checks the operation mode of heating and hotwater and the hotwater live temperature
// and returns, which quick mode should be started, when evcc sends an "Enable"
func (c *Connection) WhichQuickMode(dhwData *DhwData, zoneData *ZoneData, strategy int, heatingPar *HeatingParStruct, hotwater *HotwaterParStruct) int {
	c.debug(fmt.Sprint("Strategy = ", strategy))
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
			c.debug("Strategy = hotwater, but hotwater boost not possible")
		}
	case STRATEGY_HEATING:
		if heatingQuickVetoPossible {
			whichQuickMode = 2
		} else {
			c.debug("Strategy = heating, but zone quick veto not possible")
		}
	case STRATEGY_HOTWATER_THEN_HEATING:
		if hotWaterBoostPossible {
			whichQuickMode = 1
		} else {
			if heatingQuickVetoPossible {
				whichQuickMode = 2
			} else {
				c.debug("Strategy = hotwater_then_heating, but both not possible")
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
	v := url.Values{
		"resolution":    {resolution},
		"operationMode": {operationMode},
		"energyType":    {energyType},
		"startDate":     {startDate.Format("2006-01-02T15:04:05-07:00")},
		"endDate":       {endDate.Format("2006-01-02T15:04:05-07:00")},
	}

	url := API_URL_BASE + fmt.Sprintf(ENERGY_URL, systemid, deviceUuid) + v.Encode()
	req, _ := http.NewRequest("GET", url, nil)
	if err := doJSON(c.client, req, &energyData); err != nil {
		return energyData, fmt.Errorf("error getting energy data for %s: %w", deviceUuid, err)
	}
	return energyData, nil
}
