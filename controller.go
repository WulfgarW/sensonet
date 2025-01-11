package sensonet

import (
	"fmt"
	"time"
)

type Controller struct {
	conn               *Connection
	logger             Logger
	homesCache         Cacheable[Homes]
	systemsCache       Cacheable[AllSystems]
	systemDevicesCache Cacheable[AllSystemDevices]
	systemMpcDataCache Cacheable[AllSystemMpcData]
	currentQuickmode   string
	quickmodeStarted   time.Time
	quickmodeStopped   time.Time
}

const CACHE_DURATION_HOMES = 1800
const CACHE_DURATION_SYSTEMS = 90
const CACHE_DURATION_DEVICES = 1800
const CACHE_DURATION_MPCDATA = 90

// NewController creates a new Sensonet controller.
func NewController(conn *Connection, opts ...CtrlOption) (*Controller, error) {
	ctrl := &Controller{
		conn:             conn,
		quickmodeStarted: time.Now(),
		quickmodeStopped: time.Now().Add(-2 * time.Minute), // time stamp is set in the past so that first call of refreshCurrentQuickMode() changes currentQuickmode if necessary
	}

	for _, opt := range opts {
		opt(ctrl)
	}

	ctrl.homesCache = ResettableCached(func() (Homes, error) {
		//var res Homes
		res, err := ctrl.conn.GetHomes()
		return res, err
	}, CACHE_DURATION_HOMES*time.Second)

	ctrl.systemsCache = ResettableCached(func() (AllSystems, error) {
		var res AllSystems
		homes, err := ctrl.homesCache.Get()
		for i, home := range homes {
			var systemAndStatus SystemAndStatus
			systemAndStatus.SystemId = home.SystemID
			systemAndStatus.SystemStatus, err = ctrl.conn.GetSystem(home.SystemID)
			if err != nil {
				return res, err
			}
			if len(res.SystemsAndStatus) <= i {
				res.SystemsAndStatus = append(res.SystemsAndStatus, systemAndStatus)
			} else {
				res.SystemsAndStatus[i] = systemAndStatus
			}
			// For the beginning, currentQuickMode is only calculated from the system status of Homes[0].SystemId
			if i == 0 {
				ctrl.refreshCurrentQuickMode(&systemAndStatus.SystemStatus)
			}
		}
		return res, err
	}, CACHE_DURATION_SYSTEMS*time.Second)

	ctrl.systemDevicesCache = ResettableCached(func() (AllSystemDevices, error) {
		var res AllSystemDevices
		homes, err := ctrl.homesCache.Get()
		for i, home := range homes {
			var systemDevicesAndSystemId SystemDevicesAndSystemId
			systemDevicesAndSystemId.SystemId = home.SystemID
			systemDevicesAndSystemId.SystemDevices, err = ctrl.conn.GetSystemDevices(home.SystemID)
			if err != nil {
				return res, err
			}
			if len(res.SystemDevicesAndSystemId) <= i {
				res.SystemDevicesAndSystemId = append(res.SystemDevicesAndSystemId, systemDevicesAndSystemId)
			} else {
				res.SystemDevicesAndSystemId[i] = systemDevicesAndSystemId
			}
		}
		return res, err
	}, CACHE_DURATION_DEVICES*time.Second)

	ctrl.systemMpcDataCache = ResettableCached(func() (AllSystemMpcData, error) {
		var res AllSystemMpcData
		homes, err := ctrl.homesCache.Get()
		for i, home := range homes {
			var systemMpcData SystemMpcData
			systemMpcData.SystemId = home.SystemID
			systemMpcData.MpcData, err = ctrl.conn.GetMpcData(home.SystemID)
			if err != nil {
				return res, err
			}
			if len(res.SystemMpcData) <= i {
				res.SystemMpcData = append(res.SystemMpcData, systemMpcData)
			} else {
				res.SystemMpcData[i] = systemMpcData
			}
		}
		return res, err
	}, CACHE_DURATION_MPCDATA*time.Second)

	return ctrl, nil
}

func (c *Controller) debug(fmt string, arg ...any) {
	if c.logger != nil {
		c.logger.Printf(fmt, arg...)
	}
}

// Returns all "homes" that belong to the current user under the myVaillant portal
func (c *Controller) GetHomes() (Homes, error) {
	homes, err := c.homesCache.Get()
	if err != nil {
		return nil, err
	}
	if len(homes) < 1 {
		return nil, fmt.Errorf("error: no homes")
	}
	return homes, nil
}

// Returns the system report (state, properties and configuration) for a specific systemId
func (c *Controller) GetSystem(systemId string) (SystemStatus, error) {
	var systemStatus SystemStatus
	systems, err := c.systemsCache.Get()
	if err != nil {
		return systemStatus, err
	}
	for _, sys := range systems.SystemsAndStatus {
		if sys.SystemId == systemId {
			return sys.SystemStatus, nil
		}
	}
	return systemStatus, fmt.Errorf("no data found for system %s", systemId)
}

// Returns the device data for given criteria
func (c *Controller) GetDeviceData(systemId string, whichDevices int) ([]DeviceAndInfo, error) {
	var devices []DeviceAndInfo
	systemDevices, err := c.GetSystemDevices(systemId)
	if err != nil {
		return devices, err
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

// Returns the energy data for systemId, deviceUuid and other given criteria
func (c *Controller) GetEnergyData(systemId, deviceUuid, operationMode, energyType, resolution string, startDate, endDate time.Time) (EnergyData, error) {
	return c.conn.GetEnergyData(systemId, deviceUuid, operationMode, energyType, resolution, startDate, endDate)
}

// Returns the mpc data for systemId
func (c *Controller) GetMpcData(systemId string) (MpcData, error) {
	var mpcData MpcData
	allSystemMpcData, err := c.systemMpcDataCache.Get()
	if err != nil {
		return mpcData, err
	}
	for _, systemMpcData := range allSystemMpcData.SystemMpcData {
		if systemMpcData.SystemId == systemId {
			return systemMpcData.MpcData, err
		}
	}
	return mpcData, fmt.Errorf("no mpc data found for system %s", systemId)
}

// Returns the system devices for a specific systemId
func (c *Controller) GetSystemDevices(systemId string) (SystemDevices, error) {
	var systemDevices SystemDevices
	allSystemDevices, err := c.systemDevicesCache.Get()
	if err != nil {
		return systemDevices, err
	}
	for _, sys := range allSystemDevices.SystemDevicesAndSystemId {
		if sys.SystemId == systemId {
			return sys.SystemDevices, nil
		}
	}
	return systemDevices, fmt.Errorf("no data found for system %s", systemId)
}

// Returns the current power consumption for systemId
func (c *Controller) GetSystemCurrentPower(systemId string) (float64, error) {
	mpcData, err := c.GetMpcData(systemId)
	if err != nil || len(mpcData.Devices) < 1 {
		return -1.0, err
	}
	totalPower := 0.0
	for _, dev := range mpcData.Devices {
		totalPower = totalPower + dev.CurrentPower
	}
	return totalPower, nil
}

// Returns the current power consumption and product name for deviceUuid. If "All" is given as deviceUuid, then the function return the power consumption and product name for all devices of systemId
func (c *Controller) GetDeviceCurrentPower(systemId, deviceUuid string) (DevicePowerMap, error) {
	devicePowerMap := make(DevicePowerMap)
	if deviceUuid == "All" {
		devicePowerMap["All"] = DevicePower{CurrentPower: -1.0, ProductName: "All Devices"}
	}
	mpcData, err := c.GetMpcData(systemId)
	if err != nil || len(mpcData.Devices) < 1 {
		return devicePowerMap, err
	}
	devices, err := c.GetDeviceData(systemId, DEVICES_ALL)
	if err != nil {
		return devicePowerMap, err
	}
	totalPower := 0.0
	for _, dev := range mpcData.Devices {
		totalPower = totalPower + dev.CurrentPower
		if dev.DeviceID == deviceUuid || deviceUuid == "All" {
			for _, dev2 := range devices {
				if dev.DeviceID == dev2.Device.DeviceUUID {
					devicePowerMap[dev.DeviceID] = DevicePower{CurrentPower: dev.CurrentPower, ProductName: dev2.Device.ProductName}
				}
			}
		}
	}
	devicePowerMap["All"] = DevicePower{CurrentPower: totalPower, ProductName: "All Devices"}
	return devicePowerMap, nil
}

func (c *Controller) GetCurrentQuickMode() string {
	return c.currentQuickmode
}

func (c *Controller) refreshCurrentQuickMode(state *SystemStatus) {
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
		if newQuickMode == "" && time.Now().After(c.quickmodeStarted.Add(CACHE_DURATION_SYSTEMS)) {
			if c.currentQuickmode == QUICKMODE_NOTHING && time.Now().Before(c.quickmodeStarted.Add(10*time.Minute)) {
				c.debug("Idle mode active for less then 10 minutes. Keeping the idle mode")
			} else {
				c.debug(fmt.Sprintf("Old quickmode: \"%s\"   New quickmode: \"%s\"", c.currentQuickmode, newQuickMode))
				c.currentQuickmode = newQuickMode
				c.quickmodeStopped = time.Now()
			}
		}
		if newQuickMode != "" && time.Now().After(c.quickmodeStopped.Add(CACHE_DURATION_SYSTEMS)) {
			c.debug(fmt.Sprintf("Old quickmode: \"%s\"   New quickmode: \"%s\"", c.currentQuickmode, newQuickMode))
			c.currentQuickmode = newQuickMode
			c.quickmodeStarted = time.Now()
		}
	}
}

func (c *Controller) StartZoneQuickVeto(systemId string, zone int, setpoint float32, duration float32) error {
	err := c.conn.StartZoneQuickVeto(systemId, zone, setpoint, duration)
	if err == nil && c.currentQuickmode != QUICKMODE_HOTWATER {
		c.currentQuickmode = QUICKMODE_HEATING
		c.quickmodeStarted = time.Now()
	}
	return err
}

func (c *Controller) StopZoneQuickVeto(systemId string, zone int) error {
	err := c.conn.StopZoneQuickVeto(systemId, zone)
	if err == nil && c.currentQuickmode != QUICKMODE_HOTWATER {
		c.currentQuickmode = ""
		c.quickmodeStopped = time.Now()
		c.systemsCache.Reset()
	}
	return err
}

func (c *Controller) StartHotWaterBoost(systemId string, hotwaterIndex int) error {
	err := c.conn.StartHotWaterBoost(systemId, hotwaterIndex)
	if err == nil {
		c.currentQuickmode = QUICKMODE_HOTWATER
		c.quickmodeStarted = time.Now()
	}
	return err
}

func (c *Controller) StopHotWaterBoost(systemId string, hotwaterIndex int) error {
	err := c.conn.StopHotWaterBoost(systemId, hotwaterIndex)
	if err == nil && c.currentQuickmode != QUICKMODE_HEATING {
		c.currentQuickmode = ""
		c.quickmodeStopped = time.Now()
		c.systemsCache.Reset()
	}
	return err
}

func (c *Controller) StartStrategybased(systemId string, strategy int, heatingPar *HeatingParStruct, hotwaterPar *HotwaterParStruct) (string, error) {
	c.systemsCache.Reset()
	state, err := c.GetSystem(systemId)
	if err != nil {
		return "", err
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

	c.systemsCache.Reset()
	return c.currentQuickmode, err
}

func (c *Controller) StopStrategybased(systemId string, heatingPar *HeatingParStruct, hotwaterPar *HotwaterParStruct) (string, error) {
	c.systemsCache.Reset()
	state, err := c.GetSystem(systemId)
	if err != nil {
		return "", err
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

	c.systemsCache.Reset()
	return c.currentQuickmode, err
}

// This function checks the operation mode of heating and hotwater and the hotwater live temperature
// and returns, which quick mode should be started, when evcc sends an "Enable"
func (c *Controller) WhichQuickMode(dhwData *DhwData, zoneData *ZoneData, strategy int, heatingPar *HeatingParStruct, hotwater *HotwaterParStruct) int {
	//c.debug(fmt.Sprint("Strategy = ", strategy))
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
